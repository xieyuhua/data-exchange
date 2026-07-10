<template>
  <div class="task-form-page page-wrap" :class="{ 'sql-expanded': sqlFullscreen }">
    <div class="page-bar">
      <button class="btn btn-ghost" @click="goBack">&larr; 返回任务列表</button>
      <span class="page-bar-title">{{ isEdit ? '编辑任务' : '新增任务' }}</span>
    </div>

    <div class="panel">
      <div class="panel-body">
        <section class="form-section">
          <h3 class="section-title">基础信息</h3>
          <div class="form-grid">
            <div class="form-row"><label>所属厂家 *</label>
              <select v-model.number="form.vendor_id">
                <option v-for="v in vendors" :key="v.id" :value="v.id">{{ v.name }} ({{ v.code }})</option>
              </select>
            </div>
            <div class="form-row"><label>任务名称 *</label><input v-model="form.task_name" placeholder="如 每日订单导出"></div>
            <div class="form-row"><label>执行模式</label>
              <select v-model="form.execution_mode">
                <option value="export_only">仅导出CSV</option>
                <option value="upload">导出并上传</option>
              </select>
            </div>
            <div class="form-row"><label>数据库连接</label>
              <select v-model.number="form.db_connection_id">
                <option :value="null">（不使用）</option>
                <option v-for="d in dbs" :key="d.id" :value="d.id">{{ d.name }}</option>
              </select>
            </div>
            <div class="form-row"><label>FTP/SFTP账号</label>
              <select v-model.number="form.ftp_account_id">
                <option :value="null">（不上传）</option>
                <option v-for="f in ftps" :key="f.id" :value="f.id">{{ f.name }}</option>
              </select>
            </div>
            <div class="form-row"><label>状态</label>
              <select v-model.number="form.enabled">
                <option :value="1">启用</option>
                <option :value="0">停用</option>
              </select>
            </div>
          </div>
        </section>

        <section class="form-section">
          <h3 class="section-title">调度与输出</h3>
          <div class="form-grid">
            <div class="form-row full"><label>CSV文件名模板</label>
              <input v-model="form.csv_filename_template" placeholder="{date}{HH}{MM}{SS}.csv">
              <div class="hint">可用：{vendor_code} {task_name} {date} {datetime} {yyyy} {mm} {dd} {HH} {MM} {SS} {yesterday} {yesterday_datetime}</div>
            </div>
            <div class="form-row full"><label>Cron表达式</label><CronInput v-if="loaded" v-model="form.cron_expression" /></div>
          </div>
        </section>

        <section class="form-section sql-section">
          <h3 class="section-title">SQL 内容
            <button type="button" class="btn btn-ghost btn-sm sql-fs-toggle" @click="toggleFullscreen">{{ sqlFullscreen ? '退出全屏' : '全屏' }}</button>
            <button v-if="isEdit" type="button" class="btn btn-ghost btn-sm" @click="openHistory">历史版本</button>
          </h3>
          <div class="sql-block" :class="{ 'sql-fullscreen': sqlFullscreen }">
            <div ref="sqlEditor" class="sql-editor"></div>
            <div class="hint" v-pre>支持常量占位符，如 SELECT * FROM t WHERE d='{{ yesterday }}'；输入时自动提示 SQL 关键字与函数（Ctrl+Space 手动触发）</div>
          </div>
        </section>

        <div class="form-actions">
          <button class="btn" :disabled="testing" @click="testSQL">{{ testing ? '执行中...' : '测试 SQL' }}</button>
          <div class="spacer"></div>
          <button class="btn" @click="goBack">取消</button>
          <button class="btn btn-primary" @click="save">保存任务</button>
        </div>

        <div v-if="sqlResult !== null" class="sql-test-result">
          <div class="sql-test-title">
            <span>SQL测试结果（最多10行预览，导出为全部数据）</span>
            <button v-if="!sqlResult.error && sqlResult.columns.length" class="btn btn-ghost btn-sm" :disabled="exporting" @click="exportResult">{{ exporting ? '导出中...' : '导出全部 CSV' }}</button>
          </div>
          <div v-if="sqlResult.error" class="sql-test-error">{{ sqlResult.error }}</div>
          <div v-else class="result-scroll">
            <table class="result-table">
              <thead><tr><th v-for="c in sqlResult.columns" :key="c">{{ c }}</th></tr></thead>
              <tbody>
                <tr v-for="(row, i) in sqlResult.rows" :key="i">
                  <td v-for="(v, j) in row" :key="j">{{ v }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="sqlResult.row_count === 0 && !sqlResult.error" class="muted">查询无结果返回</div>
        </div>
      </div>
    </div>

    <!-- SQL 内容历史版本 -->
    <div v-if="showHistory" class="modal-mask" @click.self="closeHistory">
      <div class="modal modal-lg">
        <div class="modal-head">
          <h3>SQL 历史版本 · {{ form.task_name }}</h3>
          <button class="modal-close" @click="closeHistory">&times;</button>
        </div>
        <div class="modal-body">
          <div class="hist-toolbar">
            <label class="switch"><input type="checkbox" v-model="compareCurrent"> 包含「当前版本」参与对比</label>
            <span class="muted f12">勾选 1~2 个历史版本进行比对（与「当前版本」组合时为 1 个历史）</span>
            <div class="spacer"></div>
            <button class="btn btn-primary btn-sm" :disabled="!canCompare" @click="doCompare">对比差异</button>
          </div>
          <div v-if="historyLoading" class="empty">加载中…</div>
          <div v-else-if="!historyList.length" class="empty">暂无历史版本（SQL 内容变更后会自动记录）</div>
          <div v-else class="hist-list">
            <div v-for="h in historyList" :key="h.id" class="hist-item">
              <div class="hist-meta">
                <label class="cmp-check" :class="{ on: compareIds.includes(h.id) }">
                  <input type="checkbox" :checked="compareIds.includes(h.id)" @change="toggleCompare(h)">
                </label>
                <span class="badge badge-info">#{{ h.id }}</span>
                <span class="muted">{{ h.created_at }}</span>
                <span v-if="h.changed_by" class="muted">· {{ h.changed_by }}</span>
                <span v-if="h.remark" class="muted">· {{ h.remark }}</span>
                <div class="spacer"></div>
                <button class="btn btn-ghost btn-sm" @click="toggleView(h)">{{ viewId === h.id ? '收起' : '查看' }}</button>
                <button class="btn btn-primary btn-sm" @click="restore(h)">恢复此版本</button>
              </div>
              <pre v-if="viewId === h.id" class="hist-sql">{{ h.sql_content }}</pre>
            </div>
          </div>
        </div>
        <div class="modal-foot"><button class="btn" @click="closeHistory">关闭</button></div>
      </div>
    </div>

    <!-- 版本差异对比 -->
    <div v-if="showDiff" class="modal-mask" @click.self="closeDiff">
      <div class="modal modal-xl">
        <div class="modal-head">
          <h3>版本差异对比</h3>
          <button class="modal-close" @click="closeDiff">&times;</button>
        </div>
        <div class="modal-body">
          <div class="diff-legend">
            <span class="diff-chip del">- {{ diffLabels.a }}</span>
            <span class="diff-chip add">+ {{ diffLabels.b }}</span>
            <span class="muted">共 {{ diffStats.add }} 处新增 / {{ diffStats.del }} 处删除</span>
            <div class="spacer"></div>
            <button class="btn btn-ghost btn-sm" :disabled="!diffPosList.length" @click="jumpDiff(-1)">↑ 上一处</button>
            <button class="btn btn-ghost btn-sm" :disabled="!diffPosList.length" @click="jumpDiff(1)">
              {{ diffPosList.length ? (diffCursor + 1) + ' / ' + diffPosList.length : '0 / 0' }} 下一处 ↓
            </button>
          </div>
          <div class="diff-scroll" ref="diffScroll">
            <div v-for="(d, i) in diffRows" :key="i" class="diff-line" :class="['diff-line', 'd-' + d.t, { 'diff-cur': i === diffCur }]">
              <span class="dl-no">{{ d.o !== '' ? d.o : '' }}</span>
              <span class="dl-no">{{ d.n !== '' ? d.n : '' }}</span>
              <span class="dl-sign">{{ d.t === ' ' ? ' ' : d.t }}</span>
              <span class="dl-text">{{ d.text }}</span>
            </div>
          </div>
        </div>
        <div class="modal-foot"><button class="btn" @click="closeDiff">关闭</button></div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
import CronInput from '../components/CronInput.vue'
import { EditorView, basicSetup } from 'codemirror'
import { sql } from '@codemirror/lang-sql'
import { syntaxHighlighting, HighlightStyle } from '@codemirror/language'
import { tags as t } from '@lezer/highlight'

// 高级暗色主题：深蓝灰底 + 青蓝高亮
const sqlDarkTheme = EditorView.theme({
  '&': {
    color: '#e2e8f0',
    backgroundColor: '#0f172a',
    height: '100%',
    fontSize: '14px'
  },
  '.cm-content': { caretColor: '#38bdf8' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: '#38bdf8' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: '#1e3a5f'
  },
  '.cm-gutters': {
    backgroundColor: '#0f172a',
    color: '#475569',
    border: 'none'
  },
  '.cm-activeLineGutter': { backgroundColor: '#1e293b', color: '#94a3b8' },
  '.cm-activeLine': { backgroundColor: '#1e293b40' },
  '.cm-selectionMatch': { backgroundColor: '#334155' },
  '.cm-lineNumbers .cm-gutterElement': { padding: '0 12px 0 8px' },
  '.cm-foldPlaceholder': { backgroundColor: '#1e293b', border: 'none', color: '#94a3b8' }
}, { dark: true })

// 暗色语法高亮（覆盖默认亮色方案，确保深色背景下可读）
const sqlDarkHighlight = HighlightStyle.define([
  { tag: t.keyword, color: '#c084fc', fontWeight: '600' },
  { tag: [t.string, t.special(t.string)], color: '#86efac' },
  { tag: [t.number, t.bool, t.null], color: '#fdba74' },
  { tag: t.comment, color: '#64748b', fontStyle: 'italic' },
  { tag: [t.operator, t.punctuation], color: '#7dd3fc' },
  { tag: [t.function(t.variableName), t.function(t.propertyName)], color: '#7dd3fc' },
  { tag: [t.variableName, t.propertyName], color: '#e2e8f0' },
  { tag: [t.typeName, t.className], color: '#5eead4' },
  { tag: t.definitionKeyword, color: '#c084fc', fontWeight: '600' }
])

export default {
  components: { CronInput },
  data() {
    return {
      isEdit: false,
      vendors: [], dbs: [], ftps: [],
      testing: false, exporting: false, sqlResult: null, editor: null, sqlFullscreen: false, loaded: false,
      form: {
        id: 0, vendor_id: 0, task_name: '', execution_mode: 'export_only',
        db_connection_id: null, ftp_account_id: null, cron_expression: '0 2 * * *',
        sort_order: 0, csv_filename_template: '{date}{HH}{MM}{SS}_{task_name}.csv',
        sql_content: '', enabled: 1
      },
      showHistory: false, historyList: [], historyLoading: false, viewId: 0,
      compareIds: [], compareCurrent: false,
      showDiff: false, diffRows: [], diffLabels: { a: '', b: '' }, diffStats: { add: 0, del: 0 },
      diffCur: -1, diffCursor: 0
    }
  },
  inject: ['toast'],
  computed: {
    canCompare() {
      return this.compareIds.length === 2 || (this.compareIds.length === 1 && this.compareCurrent)
    },
    // 所有“有差异”的行在 diffRows 中的全局索引（跳过无变化的行）
    diffPosList() {
      const out = []
      this.diffRows.forEach((r, idx) => { if (r.t !== ' ') out.push(idx) })
      return out
    }
  },
  async mounted() {
    const [vr, dr, fr] = await Promise.all([
      api.get('/vendors'), api.get('/db-connections'), api.get('/ftp-accounts')
    ])
    this.vendors = vr.data || []
    this.dbs = dr.data || []
    this.ftps = fr.data || []
    this.isEdit = !!this.$route.params.id

    if (this.isEdit) {
      const r = await api.get('/tasks/' + this.$route.params.id)
      if (r.code === 0) {
        const t = r.data
        this.form = {
          id: t.id, vendor_id: t.vendor_id, task_name: t.task_name,
          execution_mode: t.execution_mode, db_connection_id: t.db_connection_id,
          ftp_account_id: t.ftp_account_id, cron_expression: t.cron_expression,
          sort_order: t.sort_order, csv_filename_template: t.csv_filename_template,
          sql_content: t.sql_content, enabled: t.enabled
        }
      } else {
        this.toast(r.message, 'error')
      }
    } else {
      const qv = Number(this.$route.query.vendor)
      this.form.vendor_id = qv || this.vendors[0]?.id || 0
    }
    // 数据齐备后再渲染 CronInput，避免异步回填覆盖用户已输入的值
    this.loaded = true
    this.$nextTick(() => this.initEditor())
  },
  beforeDestroy() {
    this.destroyEditor()
    document.removeEventListener('keydown', this.onEsc)
    document.body.classList.remove('sql-fs-open')
  },
  created() { this.onEsc = e => { if (e.key === 'Escape' && this.sqlFullscreen) this.toggleFullscreen() } },
  methods: {
    initEditor() {
      if (this.editor || !this.$refs.sqlEditor) return
      this.editor = new EditorView({
        doc: this.form.sql_content || '',
        parent: this.$refs.sqlEditor,
        extensions: [
          basicSetup,
          sql(),
          sqlDarkTheme,
          syntaxHighlighting(sqlDarkHighlight),
          EditorView.lineWrapping,
          EditorView.updateListener.of(u => {
            if (u.docChanged) this.form.sql_content = u.state.doc.toString()
          })
        ]
      })
    },
    syncEditor() { if (this.editor) this.form.sql_content = this.editor.state.doc.toString() },
    toggleFullscreen() {
      this.sqlFullscreen = !this.sqlFullscreen
      if (this.sqlFullscreen) {
        document.body.classList.add('sql-fs-open')
        document.addEventListener('keydown', this.onEsc)
      } else {
        document.body.classList.remove('sql-fs-open')
        document.removeEventListener('keydown', this.onEsc)
      }
      this.$nextTick(() => { if (this.editor) this.editor.requestMeasure() })
    },
    buildCSV(columns, rows, filename) {
      const esc = v => {
        const s = v === null || v === undefined ? '' : String(v)
        return /[",\n]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s
      }
      const lines = [columns.map(esc).join(',')]
      for (const row of rows) lines.push(row.map(esc).join(','))
      const csv = '﻿' + lines.join('\n')
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a); a.click(); document.body.removeChild(a)
      URL.revokeObjectURL(url)
    },
    async exportResult() {
      if (!this.form.db_connection_id) return this.toast('请先选择数据库连接', 'error')
      if (!this.form.sql_content) return this.toast('SQL内容不能为空', 'error')
      this.exporting = true
      try {
        // 导出全部数据：调用后端全量查询接口（不受预览 10 行限制）
        const r = await api.post('/tasks/test-sql-export', { db_connection_id: this.form.db_connection_id, sql_content: this.form.sql_content })
        if (r.code === 0 && r.data.columns && r.data.columns.length) {
          this.buildCSV(r.data.columns, r.data.rows, 'sql_export_' + new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-') + '.csv')
          this.toast('已导出 ' + r.data.row_count + ' 行', 'success')
        } else if (r.code !== 0) {
          this.toast(r.message, 'error')
        } else {
          this.toast('查询无数据可导出', 'error')
        }
      } catch (e) {
        this.toast('导出请求失败', 'error')
      } finally { this.exporting = false }
    },
    destroyEditor() { if (this.editor) { this.editor.destroy(); this.editor = null } },
    goBack() { this.$router.push('/tasks' + (this.form.vendor_id ? '?vendor=' + this.form.vendor_id : '')) },
    async testSQL() {
      if (!this.form.db_connection_id) return this.toast('请先选择数据库连接', 'error')
      if (!this.form.sql_content) return this.toast('SQL内容不能为空', 'error')
      this.testing = true
      try {
        const r = await api.post('/tasks/test-sql', { db_connection_id: this.form.db_connection_id, sql_content: this.form.sql_content })
        if (r.code === 0) {
          this.sqlResult = { columns: r.data.columns, rows: r.data.rows, row_count: r.data.row_count, error: null }
          this.toast('SQL执行成功，返回 ' + r.data.row_count + ' 行', 'success')
        } else {
          this.sqlResult = { error: r.message, columns: [], rows: [], row_count: 0 }
          this.toast(r.message, 'error')
        }
      } catch (e) {
        this.sqlResult = { error: '请求失败', columns: [], rows: [], row_count: 0 }
        this.toast('测试请求失败', 'error')
      } finally { this.testing = false }
    },
    async save() {
      if (!this.form.task_name) return this.toast('任务名称不能为空', 'error')
      if (!this.form.vendor_id) return this.toast('请选择所属厂家', 'error')
      this.syncEditor()
      const r = await api.post('/tasks', this.form)
      if (r.code === 0) { this.toast('已保存', 'success'); this.goBack() }
      else this.toast(r.message, 'error')
    },
    async openHistory() {
      if (!this.form.id) return
      this.viewId = 0
      this.compareIds = []
      this.compareCurrent = false
      this.showDiff = false
      this.showHistory = true
      await this.loadHistory()
    },
    async loadHistory() {
      if (!this.form.id) return
      this.historyLoading = true
      try {
        const r = await api.get('/tasks/' + this.form.id + '/history', { page: 1, page_size: 100 })
        this.historyList = r.data || []
      } catch (e) { this.toast('获取历史失败', 'error'); this.historyList = [] }
      finally { this.historyLoading = false }
    },
    closeHistory() { this.showHistory = false; this.historyList = []; this.viewId = 0; this.compareIds = []; this.compareCurrent = false },
    toggleView(h) { this.viewId = this.viewId === h.id ? 0 : h.id },
    toggleCompare(h) {
      const i = this.compareIds.indexOf(h.id)
      if (i === -1) {
        if (this.compareIds.length >= 2) { this.toast('最多选择 2 个历史版本进行对比', 'error'); return }
        this.compareIds.push(h.id)
      } else {
        this.compareIds.splice(i, 1)
      }
    },
    doCompare() {
      let aText, aLabel, bText, bLabel
      if (this.compareIds.length === 2) {
        const [idA, idB] = this.compareIds
        const ha = this.historyList.find(x => x.id === idA)
        const hb = this.historyList.find(x => x.id === idB)
        if (!ha || !hb) return
        aText = ha.sql_content; aLabel = '历史 #' + ha.id + '（' + ha.created_at + '）'
        bText = hb.sql_content; bLabel = '历史 #' + hb.id + '（' + hb.created_at + '）'
      } else {
        const ha = this.historyList.find(x => x.id === this.compareIds[0])
        if (!ha) return
        aText = ha.sql_content; aLabel = '历史 #' + ha.id + '（' + ha.created_at + '）'
        bText = (this.form && this.form.sql_content) || ''; bLabel = '当前版本'
      }
      this.diffRows = this.diffLines(aText.split('\n'), bText.split('\n'))
      let add = 0, del = 0
      this.diffRows.forEach(r => { if (r.t === '+') add++; else if (r.t === '-') del++ })
      this.diffStats = { add, del }
      this.diffLabels = { a: aLabel, b: bLabel }
      this.diffCur = -1
      this.diffCursor = 0
      this.showDiff = true
    },
    closeDiff() { this.showDiff = false },
    // 在差异行之间跳转：dir=1 下一处，dir=-1 上一处（循环）
    jumpDiff(dir) {
      const list = this.diffPosList
      if (!list.length) return
      let c = this.diffCursor + dir
      if (c < 0) c = list.length - 1
      if (c >= list.length) c = 0
      this.diffCursor = c
      this.diffCur = list[c]
      this.$nextTick(() => {
        const els = this.$refs.diffScroll && this.$refs.diffScroll.querySelectorAll('.diff-line')
        const el = els && els[this.diffCur]
        if (el) el.scrollIntoView({ block: 'center', behavior: 'smooth' })
      })
    },
    // 基于 LCS 的行级差异：返回 [{t:' '/'−'/'+', o:旧行号, n:新行号, text}]
    diffLines(aLines, bLines) {
      const n = aLines.length, m = bLines.length
      const dp = Array.from({ length: n + 1 }, () => new Array(m + 1).fill(0))
      for (let i = n - 1; i >= 0; i--) {
        for (let j = m - 1; j >= 0; j--) {
          dp[i][j] = aLines[i] === bLines[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1])
        }
      }
      const res = []
      let i = 0, j = 0, o = 0, nn = 0
      while (i < n && j < m) {
        if (aLines[i] === bLines[j]) { res.push({ t: ' ', o: ++o, n: ++nn, text: aLines[i] }); i++; j++ }
        else if (dp[i + 1][j] >= dp[i][j + 1]) { res.push({ t: '-', o: ++o, n: '', text: aLines[i] }); i++ }
        else { res.push({ t: '+', o: '', n: ++nn, text: bLines[j] }); j++ }
      }
      while (i < n) { res.push({ t: '-', o: ++o, n: '', text: aLines[i] }); i++ }
      while (j < m) { res.push({ t: '+', o: '', n: ++nn, text: bLines[j] }); j++ }
      return res
    },
    setEditorContent(sql) {
      const s = sql || ''
      if (this.editor) {
        this.editor.dispatch({ changes: { from: 0, to: this.editor.state.doc.length, insert: s } })
      }
      this.form.sql_content = s
    },
    async restore(h) {
      if (!confirm('确认将该任务的 SQL 恢复到版本 #' + h.id + '？当前内容会自动备份为新历史。')) return
      const r = await api.post('/task-history/' + h.id + '/restore', {})
      if (r.code === 0) {
        this.toast('已恢复到版本 #' + h.id, 'success')
        this.setEditorContent((this.historyList.find(x => x.id === h.id) || {}).sql_content || '')
        await this.loadHistory()
      } else this.toast(r.message, 'error')
    }
  }
}
</script>

<style scoped>
.modal-lg { width: 760px; max-width: 92vw; }
.modal-xl { width: 920px; max-width: 95vw; }
.hist-list { max-height: 56vh; overflow: auto; }
.hist-item { border: 1px solid #e2e8f0; border-radius: 8px; padding: 10px 12px; margin-bottom: 10px; }
.hist-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.hist-toolbar { display: flex; align-items: center; gap: 12px; padding: 8px 2px 12px; flex-wrap: wrap; border-bottom: 1px solid #eef2f7; margin-bottom: 10px; }
.cmp-check { display: inline-flex; align-items: center; cursor: pointer; }
.cmp-check.on { color: #2563eb; }
.cmp-check input { width: 16px; height: 16px; cursor: pointer; }
.hist-sql { margin-top: 8px; background: #0f172a; color: #e2e8f0; padding: 10px 12px; border-radius: 6px; font-size: 12px; white-space: pre-wrap; word-break: break-all; max-height: 320px; overflow: auto; }
.diff-legend { display: flex; align-items: center; gap: 12px; padding: 6px 2px 10px; flex-wrap: wrap; }
.diff-legend .spacer { flex: 1; }
.diff-line.diff-cur { outline: 2px solid #f59e0b; outline-offset: -2px; box-shadow: 0 0 0 2px rgba(245,158,11,0.25); }
.diff-chip { padding: 2px 8px; border-radius: 4px; font-size: 12px; }
.diff-chip.del { background: #fee2e2; color: #b91c1c; }
.diff-chip.add { background: #dcfce7; color: #15803d; }
.diff-scroll { max-height: 60vh; overflow: auto; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; font-family: 'Consolas', 'Courier New', monospace; font-size: 12px; }
.diff-line { display: flex; gap: 6px; padding: 0 8px; line-height: 1.6; white-space: pre-wrap; word-break: break-all; }
.diff-line.d- { background: #fff; }
.diff-line.d-- { background: #fee2e2; }
.diff-line.d-\+ { background: #dcfce7; }
.dl-no { width: 34px; text-align: right; color: #94a3b8; user-select: none; flex: none; }
.dl-sign { width: 14px; text-align: center; color: #64748b; flex: none; }
.dl-text { flex: 1; }
.switch { display: inline-flex; align-items: center; gap: 4px; font-size: 13px; color: #475569; cursor: pointer; }
</style>
