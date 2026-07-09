<template>
  <div class="panel">
    <div class="panel-head ftp-files-head">
      <div class="fh-left">
        <button class="btn btn-ghost btn-sm" @click="goBack">&larr; 返回</button>
        <h2>远程文件</h2>
        <span v-if="acc" class="acc-tag">
          {{ acc.name }}
          <span class="badge" :class="'badge-'+acc.protocol">{{ acc.protocol }}</span>
        </span>
      </div>
      <div class="file-tools">
        <label class="btn btn-primary btn-sm">
          {{ uploading ? '上传中…' : '上传文件' }}
          <input type="file" style="display:none" :disabled="uploading" @change="onUpload">
        </label>
        <button class="btn btn-sm" :disabled="filesLoading" @click="loadFiles">刷新</button>
      </div>
    </div>

    <div class="panel-body p0">
      <div class="path-bar muted" v-if="acc">远程路径：<span class="cell-mono">{{ acc.remote_path }}</span> · 主机：<span class="cell-mono">{{ acc.host }}:{{ acc.port }}</span></div>
      <div class="list-toolbar">
        <div class="search-box">
          <span class="search-ico" aria-hidden="true">&#128269;</span>
          <input class="inp search-input" v-model.trim="keyword" placeholder="搜索文件名…" @keyup.enter="doSearch">
          <button class="search-clear" v-if="keyword" @click="clearSearch" title="清除搜索">&times;</button>
        </div>
        <button class="btn btn-primary btn-sm" @click="doSearch">搜索</button>
        <span class="muted" v-if="total > 0">共 {{ total }} 项</span>
      </div>
      <div class="table-scroll" v-if="files.length && !filesLoading" style="max-height:none">
        <table class="ftp-table">
          <thead><tr><th>名称</th><th class="num-col">大小</th><th>修改时间</th><th>类型</th><th class="op-col">操作</th></tr></thead>
          <tbody>
            <tr v-for="f in files" :key="f.name" :class="{ 'is-dir': f.is_dir }">
              <td class="name-cell">
                <span v-if="f.is_dir" class="dir-ico">&#128193;</span>
                <span v-else class="file-ico">&#128196;</span>
                <span class="fname">{{ f.name }}</span>
              </td>
              <td class="num-col">{{ f.is_dir ? '—' : formatSize(f.size) }}</td>
              <td class="cell-mono">{{ f.mod_time }}</td>
              <td><span class="type-badge" :class="f.is_dir ? 'dir' : 'file'">{{ f.is_dir ? '目录' : '文件' }}</span></td>
              <td class="op-col"><button class="btn btn-danger btn-sm" @click="delFile(f.name)">删除</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else-if="filesLoading" class="empty">加载中…</div>
      <div v-else class="empty">远程目录为空或无法访问</div>
      <div class="pager" v-if="total > pageSize">
        <button class="btn btn-sm" :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
        <span class="muted">第 {{ page }} / {{ totalPages }} 页</span>
        <button class="btn btn-sm" :disabled="page >= totalPages" @click="changePage(page + 1)">下一页</button>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
import { getPageSize, refreshPageSize } from '../configStore'
export default {
  data() {
    return {
      accountId: parseInt(this.$route.params.id, 10) || 0,
      acc: null,
      files: [],
      filesLoading: false,
      uploading: false,
      keyword: '',
      page: 1,
      pageSize: getPageSize(),
      total: 0
    }
  },
  computed: {
    totalPages() {
      return Math.max(1, Math.ceil(this.total / this.pageSize))
    }
  },
  inject: ['toast'],
  async mounted() {
    await this.loadAcc()
    await refreshPageSize()
    this.pageSize = getPageSize()
    await this.loadFiles()
  },
  methods: {
    async loadAcc() {
      try {
        const r = await api.get('/ftp-accounts')
        this.acc = (r.data || []).find(a => a.id === this.accountId) || null
      } catch (e) { this.acc = null }
    },
    goBack() {
      if (window.history.length > 1) this.$router.back()
      else this.$router.push('/ftp')
    },
    doSearch() {
      this.page = 1
      this.loadFiles()
    },
    clearSearch() {
      this.keyword = ''
      this.page = 1
      this.loadFiles()
    },
    changePage(p) {
      if (p < 1 || p > this.totalPages) return
      this.page = p
      this.loadFiles()
    },
    async loadFiles() {
      if (!this.accountId) return
      this.filesLoading = true
      try {
        const r = await api.get('/ftp-accounts/' + this.accountId + '/files', {
          keyword: this.keyword,
          page: this.page,
          page_size: this.pageSize
        })
        this.files = (r.data && r.data.list) || []
        this.total = (r.data && r.data.total) || 0
      } catch (e) { this.toast('获取远程文件失败', 'error'); this.files = []; this.total = 0 }
      finally { this.filesLoading = false }
    },
    async delFile(name) {
      if (!confirm('确认删除远程文件 ' + name + '？')) return
      const r = await api.del('/ftp-accounts/' + this.accountId + '/files?path=' + encodeURIComponent(name))
      if (r.code === 0) { this.toast('已删除', 'success'); this.loadFiles() } else this.toast(r.message, 'error')
    },
    async onUpload(e) {
      const file = e.target.files[0]
      if (!file) return
      const fd = new FormData()
      fd.append('file', file)
      this.uploading = true
      try {
        const r = await api.post('/ftp-accounts/' + this.accountId + '/files', fd)
        if (r.code === 0) { this.toast('上传成功', 'success'); this.loadFiles() } else this.toast(r.message, 'error')
      } catch (err) { this.toast('上传失败', 'error') }
      finally { this.uploading = false; e.target.value = '' }
    },
    formatSize(bytes) {
      if (bytes < 1024) return bytes + ' B'
      if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
      return (bytes / 1024 / 1024).toFixed(1) + ' MB'
    }
  }
}
</script>
