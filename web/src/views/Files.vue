<template>
  <div class="panel">
    <div class="panel-head">
      <h2>文件管理</h2>
      <div class="head-actions">
        <button class="btn" :class="{active:tab==='output'}" @click="switchTab('output')">导出文件</button>
        <button class="btn" :class="{active:tab==='backup'}" @click="switchTab('backup')">备份文件</button>
      </div>
    </div>
    <div class="panel-body p0">
      <div class="table-scroll">
      <table v-if="tab==='output' && output.length"><thead><tr><th>文件名</th><th>大小</th><th>修改时间</th><th>操作</th></tr></thead><tbody>
        <tr v-for="f in output" :key="f.name">
          <td class="cell-mono">{{ f.name }}</td>
          <td>{{ fmtSize(f.size) }}</td>
          <td class="cell-mono">{{ f.mod_time }}</td>
          <td><button class="btn btn-ghost btn-sm" @click="download(f.name, 'output')">下载</button></td>
        </tr>
      </tbody></table>
      <table v-else-if="tab==='backup' && backup.length"><thead><tr><th>文件名</th><th>大小</th><th>修改时间</th><th>操作</th></tr></thead><tbody>
        <tr v-for="f in backup" :key="f.name">
          <td class="cell-mono">{{ f.name }}</td>
          <td>{{ fmtSize(f.size) }}</td>
          <td class="cell-mono">{{ f.mod_time }}</td>
          <td><button class="btn btn-ghost btn-sm" @click="download(f.name, 'backup')">下载</button></td>
        </tr>
      </tbody></table>
      </div>
      <div v-if="!(tab==='output' && output.length) && !(tab==='backup' && backup.length)" class="empty">{{ tab==='output' ? '暂无导出文件' : '暂无备份文件' }}</div>
    </div>
    <div class="pager" v-if="total > pageSize">
      <button class="btn btn-sm" :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
      <span class="muted">第 {{ page }} / {{ Math.ceil(total / pageSize) }} 页（共 {{ total }} 条）</span>
      <button class="btn btn-sm" :disabled="page >= Math.ceil(total / pageSize)" @click="changePage(page + 1)">下一页</button>
    </div>
    <div class="panel-foot" v-if="tab==='backup'">
      <span class="muted">保留数量: {{ keepCount }}</span>
      <button class="btn btn-danger" @click="clean">清理旧备份</button>
    </div>
  </div>
</template>

<script>
import api from '../api'
import { getPageSize } from '../configStore'
export default {
  data() { return { tab:'output', output:[], backup:[], keepCount:'30', page:1, pageSize:getPageSize(), total:0 } },
  inject: ['toast'],
  async mounted() { await this.loadOutput() },
  methods: {
    async switchTab(t) { this.tab = t; this.page = 1; if(t==='output') await this.loadOutput(); else await this.loadBackup() },
    changePage(p) { this.page = p; if(this.tab==='output') this.loadOutput(); else this.loadBackup() },
    async loadOutput() {
      const r = await api.get('/files/output', { page: this.page, page_size: this.pageSize })
      this.output = r.data || []
      this.total = r.total || 0
    },
    async loadBackup() {
      const r = await api.get('/files/backup', { page: this.page, page_size: this.pageSize })
      this.backup = (r.data && r.data.files) || []
      this.keepCount = (r.data && r.data.keep_count) || '30'
      this.total = (r.data && r.data.total) || 0
    },
    async download(name, dir) {
      try {
        const resp = await api.file('/files/download', { filename: name, dir })
        const blob = resp.data
        // 后端出错时仍返回 200 + JSON（code=1），blob 为 JSON 文本，需解析提示
        if (blob && blob.type && blob.type.indexOf('application/json') !== -1) {
          let msg = '下载失败'
          try { msg = (JSON.parse(await blob.text()).message) || msg } catch (e) {}
          this.toast(msg, 'error')
          return
        }
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = name
        document.body.appendChild(a); a.click(); document.body.removeChild(a)
        URL.revokeObjectURL(url)
      } catch (e) {
        this.toast('下载请求失败', 'error')
      }
    },
    fmtSize(n) {
      if(n<1024) return n+' B'
      if(n<1024*1024) return (n/1024).toFixed(1)+' KB'
      return (n/1024/1024).toFixed(2)+' MB'
    },
    async clean() {
      if(!confirm('确认清理超出保留数量的旧备份？')) return
      const r = await api.post('/files/clean-backups')
      if(r.code===0){ this.toast('清理完成','success'); this.page=1; this.loadBackup() } else this.toast(r.message,'error')
    }
  }
}
</script>
