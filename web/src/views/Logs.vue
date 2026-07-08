<template>
  <div>
    <div class="filter-bar">
      <div class="filter-item">
        <span class="filter-label">状态</span>
        <div class="filter-select">
          <SearchableSelect
            v-model="statusFilter"
            :options="statusOptions"
            placeholder="全部状态"
            search-placeholder="搜索状态"
            @change="load(1)"
          />
        </div>
      </div>
      <div class="filter-item">
        <input v-model="keyword" class="filter-control" placeholder="搜索任务 / 厂家 / 文件名…" @keyup.enter="load(1)">
      </div>
      <button class="btn" @click="load(1)">搜索</button>
      <div class="spacer"></div>
      <button class="btn btn-danger" @click="clearAll">清空日志</button>
    </div>

    <div class="panel">
    <div class="panel-head">
      <h2>执行日志</h2>
    </div>
    <div class="panel-body p0">
      <div class="table-scroll" v-if="list.length">
      <table>
        <thead><tr>
          <th>ID</th><th>任务</th><th>厂家</th><th>状态</th><th>模式</th><th>CSV文件</th><th>记录数</th><th>耗时(ms)</th><th>错误信息</th><th>开始时间</th><th>操作</th>
        </tr></thead>
        <tbody>
          <tr v-for="l in list" :key="l.id">
            <td>{{ l.id }}</td>
            <td class="cell-mono">{{ l.task_name }}</td>
            <td class="muted">{{ l.vendor_name }}</td>
            <td><span class="badge" :class="l.status === 'success' ? 'badge-on' : 'badge-off'">{{ l.status }}</span></td>
            <td>{{ l.execution_mode }}</td>
            <td class="cell-mono">{{ l.csv_filename }}</td>
            <td>{{ l.record_count }}</td>
            <td>{{ l.duration_ms }}</td>
            <td class="muted err-cell">{{ l.error_message }}</td>
            <td class="cell-mono">{{ l.started_at }}</td>
            <td><button class="btn btn-danger btn-sm" @click="del(l.id)">删除</button></td>
          </tr>
        </tbody>
      </table>
      </div>
      <div v-else class="empty">暂无日志</div>
    </div>
    <div class="pager" v-if="total > pageSize">
      <button class="btn btn-sm" :disabled="page <= 1" @click="load(page - 1)">上一页</button>
      <span class="muted">第 {{ page }} / {{ Math.ceil(total / pageSize) }} 页（共 {{ total }} 条）</span>
      <button class="btn btn-sm" :disabled="page >= Math.ceil(total / pageSize)" @click="load(page + 1)">下一页</button>
    </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
import SearchableSelect from '../components/SearchableSelect.vue'
export default {
  components: { SearchableSelect },
  data() { return { list: [], total: 0, page: 1, pageSize: 20, statusFilter: '', keyword: '' } },
  computed: {
    statusOptions() {
      return [
        { value: 'success', label: '成功' },
        { value: 'failed', label: '失败' }
      ]
    }
  },
  inject: ['toast'],
  async mounted() { await this.load(1) },
  methods: {
    async load(p) {
      this.page = p || this.page
      const r = await api.get('/logs', {
        page: this.page, page_size: this.pageSize,
        status: this.statusFilter, keyword: this.keyword
      })
      this.list = r.data || []
      this.total = r.total || 0
    },
    async del(id) {
      if (!confirm('确认删除该日志？')) return
      const r = await api.del('/logs/' + id)
      if (r.code === 0) { this.toast('已删除', 'success'); this.load() } else this.toast(r.message, 'error')
    },
    async clearAll() {
      if (!confirm('确认清空所有日志？此操作不可恢复')) return
      const r = await api.del('/logs')
      if (r.code === 0) { this.toast('已清空', 'success'); this.load(1) } else this.toast(r.message, 'error')
    }
  }
}
</script>
