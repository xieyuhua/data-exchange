<template>
  <div>
    <div class="filter-bar">
      <div class="filter-item">
        <input v-model.trim="username" class="filter-control" placeholder="按用户名筛选…" @keyup.enter="load(1)">
      </div>
      <div class="filter-item">
        <select v-model="successFilter" class="filter-control" @change="load(1)">
          <option value="">全部结果</option>
          <option value="1">成功</option>
          <option value="0">失败</option>
        </select>
      </div>
      <div class="filter-item">
        <input v-model.trim="keyword" class="filter-control" placeholder="搜索操作 / 模块 / 路径…" @keyup.enter="load(1)">
      </div>
      <button class="btn" @click="load(1)">搜索</button>
      <div class="spacer"></div>
      <button class="btn btn-danger" @click="clearAll">清空日志</button>
    </div>

    <div class="panel">
      <div class="panel-head"><h2>操作日志</h2></div>
      <div class="panel-body p0">
        <div class="table-scroll" v-if="list.length">
          <table>
            <thead><tr>
              <th>ID</th><th>用户</th><th>模块</th><th>操作</th><th>方法</th><th>路径</th><th>结果</th><th>耗时</th><th>IP</th><th>时间</th><th>操作</th>
            </tr></thead>
            <tbody>
              <tr v-for="l in list" :key="l.id">
                <td>{{ l.id }}</td>
                <td class="cell-mono">{{ l.username || '—' }}</td>
                <td>{{ l.module }}</td>
                <td>
                  {{ l.action }}
                  <span v-if="l.detail" class="detail-ico" :title="l.detail">&#9432;</span>
                </td>
                <td><span class="type-badge" :class="'m-' + (l.method || '').toLowerCase()">{{ l.method }}</span></td>
                <td class="cell-mono muted path-cell">{{ l.path }}</td>
                <td><span class="badge" :class="l.success === 1 ? 'badge-on' : 'badge-off'">{{ l.success === 1 ? '成功' : '失败' }}</span></td>
                <td class="cell-mono">{{ l.duration_ms }} ms</td>
                <td class="cell-mono muted">{{ l.ip }}</td>
                <td class="cell-mono">{{ l.created_at }}</td>
                <td class="op-cell"><button class="btn btn-danger btn-sm" @click="del(l.id)">删除</button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty">暂无操作日志</div>
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
import { getPageSize } from '../configStore'
export default {
  data() {
    return { list: [], total: 0, page: 1, pageSize: getPageSize(), username: '', keyword: '', successFilter: '' }
  },
  inject: ['toast'],
  async mounted() { await this.load(1) },
  methods: {
    async load(p) {
      this.page = p || this.page
      const r = await api.get('/operation-logs', {
        page: this.page, page_size: this.pageSize,
        username: this.username, keyword: this.keyword, success: this.successFilter
      })
      this.list = r.data || []
      this.total = r.total || 0
    },
    async del(id) {
      if (!confirm('确认删除该日志？')) return
      const r = await api.del('/operation-logs/' + id)
      if (r.code === 0) { this.toast('已删除', 'success'); this.load() } else this.toast(r.message, 'error')
    },
    async clearAll() {
      if (!confirm('确认清空所有操作日志？此操作不可恢复')) return
      const r = await api.del('/operation-logs')
      if (r.code === 0) { this.toast('已清空', 'success'); this.load(1) } else this.toast(r.message, 'error')
    }
  }
}
</script>

<style scoped>
.path-cell { max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-ico { color: #94a3b8; cursor: help; margin-left: 4px; }
.type-badge { padding: 1px 6px; border-radius: 4px; font-size: 12px; background: #eef2f7; color: #475569; }
.type-badge.m-post { background: #dbeafe; color: #1d4ed8; }
.type-badge.m-put { background: #fef3c7; color: #b45309; }
.type-badge.m-delete { background: #fee2e2; color: #b91c1c; }
</style>
