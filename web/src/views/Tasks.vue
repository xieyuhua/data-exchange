<template>
  <div class="page-wrap">
    <div class="filter-bar">
      <div class="filter-item">
        <span class="filter-label">厂家</span>
        <div class="filter-select">
          <SearchableSelect
            v-model="vendorId"
            :options="vendorOptions"
            placeholder="选择厂家"
            search-placeholder="搜索厂家名称 / 编码"
            :allow-clear="false"
            @change="loadTasks"
          />
        </div>
      </div>
      <div class="filter-item">
        <span class="filter-label">状态</span>
        <div class="filter-select">
          <SearchableSelect
            v-model="statusFilter"
            :options="statusOptions"
            placeholder="全部状态"
            search-placeholder="搜索状态"
          />
        </div>
      </div>
      <div class="filter-item">
        <input v-model="keyword" class="filter-control" placeholder="搜索任务名称…">
      </div>
      <div class="spacer"></div>
      <button class="btn btn-primary" @click="goNew">+ 新增任务</button>
    </div>

    <div class="panel">
      <div class="panel-head"><h2>任务列表（{{ filtered.length }}/{{ maxTasks }}）</h2></div>
      <div class="panel-body">
        <div v-if="filtered.length" class="task-list">
          <div v-for="t in filtered" :key="t.id" class="task-card">
            <div class="task-top">
              <span class="task-name">{{ t.task_name }}</span>
              <span class="badge" :class="t.enabled ? 'badge-on' : 'badge-off'">{{ t.enabled ? '启用' : '停用' }}</span>
              <span class="muted f12">#{{ t.id }}</span>
              <div class="spacer"></div>
              <button class="btn btn-ghost btn-sm" :disabled="running.includes(Number(t.id))" @click="execTask(t.id)">{{ running.includes(Number(t.id)) ? '执行中…' : '立即执行' }}</button>
              <button class="btn btn-ghost btn-sm" @click="toggleTask(t.id)">{{ t.enabled ? '停用' : '启用' }}</button>
              <button class="btn btn-ghost btn-sm" @click="goEdit(t.id)">编辑</button>
              <button class="btn btn-danger btn-sm" @click="delTask(t.id)">删除</button>
            </div>
            <div class="task-meta">
              模式：{{ t.execution_mode === 'upload' ? '导出并上传' : '仅导出' }} | 排序：{{ t.sort_order }} | Cron：{{ t.cron_expression }}<br>
              数据库连接：{{ t.db_connection_name || '未设置' }} | FTP账号：{{ t.ftp_account_name || '—' }}<br>
              文件名模板：{{ t.csv_filename_template }}<br>
              <span v-if="t.next_run_at">下次执行：<span class="badge badge-info">{{ t.next_run_at }}</span></span>
              <span v-else class="muted">未设置定时（停用或空 Cron）</span><br>
              <span v-if="t.last_run_at">上次执行：{{ t.last_run_at }} 状态：<span class="badge" :class="t.last_status === 'success' ? 'badge-success' : 'badge-failed'">{{ t.last_status }}</span></span>
              <span v-else>尚未执行</span>
            </div>
          </div>
        </div>
        <div v-else class="empty">{{ tasks.length ? '没有符合条件的任务' : '该厂家暂无任务（最多 ' + maxTasks + ' 个）' }}</div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
import SearchableSelect from '../components/SearchableSelect.vue'
export default {
  components: { SearchableSelect },
  data() {
    return {
      vendors: [], tasks: [], dbs: [], ftps: [], vendorId: '', maxTasks: 4,
      statusFilter: '', keyword: '', running: [], pollTimer: null
    }
  },
  inject: ['toast'],
  computed: {
    vendorOptions() {
      return (this.vendors || []).map(v => ({ value: String(v.id), label: v.name, hint: v.code }))
    },
    statusOptions() {
      return [
        { value: '1', label: '启用' },
        { value: '0', label: '停用' }
      ]
    },
    filtered() {
      const kw = this.keyword.trim().toLowerCase()
      return this.tasks.filter(t => {
        if (this.statusFilter !== '' && String(t.enabled) !== this.statusFilter) return false
        if (kw && !t.task_name.toLowerCase().includes(kw)) return false
        return true
      })
    }
  },
  async mounted() {
    const [vr, dr] = await Promise.all([api.get('/vendors'), api.get('/db-connections')])
    this.vendors = vr.data || []
    this.dbs = dr.data || []
    const qv = this.$route.query.vendor
    this.vendorId = (qv ? String(qv) : (this.vendors[0]?.id ? String(this.vendors[0].id) : ''))
    if (this.vendorId) await this.loadTasks()
    this.pollTimer = setInterval(() => this.loadRunning(), 3000)
  },
  beforeDestroy() {
    if (this.pollTimer) clearInterval(this.pollTimer)
  },
  methods: {
    async loadTasks() {
      if (!this.vendorId) return
      const [tr, fr] = await Promise.all([
        api.get('/vendors/' + this.vendorId + '/tasks'),
        api.get('/ftp-accounts?vendor_id=' + this.vendorId)
      ])
      this.tasks = tr.data || []
      this.ftps = fr.data || []
      this.maxTasks = tr.max || 4
      this.loadRunning()
    },
    async loadRunning() {
      try {
        const r = await api.get('/tasks/running')
        this.running = (r.data || []).map(Number)
      } catch (e) { /* 忽略轮询错误 */ }
    },
    goNew() { this.$router.push('/tasks/new?vendor=' + this.vendorId) },
    goEdit(id) { this.$router.push('/tasks/edit/' + id) },
    async delTask(id) {
      if (!confirm('确认删除？')) return
      const r = await api.del('/tasks/' + id)
      if (r.code === 0) { this.toast('已删除', 'success'); this.loadTasks() }
      else this.toast(r.message, 'error')
    },
    async toggleTask(id) {
      const r = await api.post('/tasks/' + id + '/toggle', {})
      if (r.code === 0) { this.toast('已切换', 'success'); this.loadTasks() }
      else this.toast(r.message, 'error')
    },
    async execTask(id) {
      if (this.running.includes(Number(id))) {
        this.toast('任务正在执行中，请稍候', 'error')
        return
      }
      const r = await api.post('/tasks/' + id + '/execute', {})
      if (r.code === 0) {
        this.running.push(Number(id))
        this.toast(r.message || '已提交', 'success')
        this.loadRunning()
      } else {
        this.toast(r.message, 'error')
      }
    }
  }
}
</script>
