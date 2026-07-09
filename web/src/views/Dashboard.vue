<template>
  <div>
    <div class="stat-grid">
      <div class="stat-card"><div class="stat-val">{{ stats.vendor_count }}</div><div class="stat-label">厂家数量</div></div>
      <div class="stat-card"><div class="stat-val">{{ stats.task_count }}</div><div class="stat-label">启用任务</div></div>
      <div class="stat-card"><div class="stat-val">{{ stats.ftp_count }}</div><div class="stat-label">FTP/SFTP账号</div></div>
      <div class="stat-card green"><div class="stat-val">{{ stats.success_count }}</div><div class="stat-label">成功执行</div></div>
      <div class="stat-card red"><div class="stat-val">{{ stats.fail_count }}</div><div class="stat-label">失败执行</div></div>
      <div class="stat-card amber"><div class="stat-val">{{ stats.log_count }}</div><div class="stat-label">总执行次数</div></div>
    </div>
    <div class="panel">
      <div class="panel-head"><h2>最近执行记录</h2><span class="muted f12">备份保留: {{ stats.backup_keep }} 个</span></div>
      <div class="panel-body p0">
        <div class="table-scroll" v-if="stats.recent_logs && stats.recent_logs.length">
        <table><thead><tr><th>ID</th><th>任务</th><th>厂家</th><th>状态</th><th>文件</th><th>记录数</th><th>耗时</th><th>开始时间</th></tr></thead><tbody>
          <tr v-for="l in stats.recent_logs" :key="l.id">
            <td>{{ l.id }}</td><td>{{ l.task_name }}</td><td>{{ l.vendor_name }}</td>
            <td><span class="badge" :class="l.status==='success'?'badge-success':'badge-failed'">{{ l.status }}</span></td>
            <td class="cell-mono">{{ l.csv_filename }}</td><td>{{ l.record_count }}</td><td class="cell-mono">{{ formatDuration(l.duration_ms) }}</td><td class="muted">{{ l.started_at }}</td>
          </tr>
        </tbody></table>
        </div>
        <div v-else class="empty">暂无执行记录</div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
export default {
  data() { return { stats: { vendor_count:0,task_count:0,ftp_count:0,log_count:0,success_count:0,fail_count:0,recent_logs:[],backup_keep:30 } } },
  async mounted() {
    try {
      const r = await api.get('/dashboard/stats')
      const data = r.data || {}
      this.stats = Object.assign({}, this.stats, data)
      if (!Array.isArray(this.stats.recent_logs)) this.stats.recent_logs = []
    } catch (e) {
      this.stats.recent_logs = []
    }
  },
  methods: {
    formatDuration(ms) {
      if (!ms || ms < 0) return '—'
      if (ms < 1000) return ms + ' ms'
      const s = ms / 1000
      if (s < 60) return s.toFixed(2) + ' s'
      const m = Math.floor(s / 60)
      const rs = (s - m * 60).toFixed(0)
      if (m < 60) return m + ' 分 ' + rs + ' 秒'
      const h = Math.floor(m / 60)
      const rm = m % 60
      return h + ' 时 ' + rm + ' 分 ' + rs + ' 秒'
    }
  }
}
</script>
