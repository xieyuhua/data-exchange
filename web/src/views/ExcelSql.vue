<template>
  <div>
    <div class="filter-bar">
      <div class="filter-item">
        <label class="switch"><input type="radio" value="insert" v-model="mode"> 生成 INSERT</label>
      </div>
      <div class="filter-item">
        <label class="switch"><input type="radio" value="with" v-model="mode"> 仅生成 WITH 临时表</label>
      </div>
      <div class="filter-item">
        <label class="switch">数据库
          <select v-model="db" class="filter-control sm">
            <option value="oracle">Oracle</option>
            <option value="mysql">MySQL</option>
          </select>
        </label>
      </div>
      <div class="filter-item">
        <label class="switch"><input type="checkbox" v-model="quoteAliasOn"> 别名加引号（Oracle 双引号 / MySQL 反引号）</label>
      </div>
      <div class="filter-item grow">
        <input v-model.trim="targetTable" class="filter-control" placeholder="目标表名（INSERT 模式需要，如 SCHEMA.TABLE）" :disabled="mode === 'with'">
      </div>
    </div>

    <div class="filter-bar">
      <div class="filter-item grow">
        <input v-model.trim="whereCond" class="filter-control" placeholder="WHERE 筛选条件（可选，使用列别名 a/b/c…，如 a > 30 AND b LIKE 'A%'）">
      </div>
      <div class="filter-item">
        <label class="switch"><input type="checkbox" v-model="allString"> 非日期列全部按字符串</label>
      </div>
      <button class="btn" @click="fillDemo">示例数据</button>
      <button class="btn btn-primary" @click="generate">生成 SQL</button>
      <button class="btn" @click="clearAll">清空</button>
    </div>

    <div class="col-set" v-if="columns.length">
      <div class="col-set-head">列字段设置（WITH 列名自动使用 a/b/c…；可指定类型，日期列支持范围查询）</div>
      <div class="col-grid">
        <div class="col-card" v-for="(c, i) in columns" :key="i">
          <span class="col-key">{{ c.key }}</span>
          <span class="col-head" :title="c.header">{{ c.header || ('COL' + (i + 1)) }}</span>
          <select v-model="c.type" class="filter-control sm">
            <option value="auto">自动 · 识别为 {{ detLabel(c.detected) }}</option>
            <option value="text">文本</option>
            <option value="number">数字</option>
            <option value="date">日期</option>
          </select>
          <span class="det-tag" :class="'det-' + c.detected" title="系统根据数据自动识别的类型">{{ detLabel(c.detected) }}</span>
          <template v-if="c.type === 'date'">
            <input type="date" v-model="c.start" class="filter-control sm" title="起始日期（含）">
            <span class="tilde">~</span>
            <input type="date" v-model="c.end" class="filter-control sm" title="结束日期（含）">
          </template>
        </div>
      </div>
    </div>

    <div class="layout">
      <div class="panel panel-left">
        <div class="panel-head"><h2>粘贴 Excel 数据（含表头）</h2><span class="muted">{{ stats }}</span></div>
        <div class="panel-body p0">
          <textarea v-model="raw" class="paste-area" placeholder="从 Excel 复制后粘贴到此，第一行作为表头。支持 Tab / 逗号 / 分号 / 空格分隔。&#10;示例：&#10;ID	姓名	出生日期	分数&#10;1	张三	1990-01-01	95.5&#10;2	李四	1992-05-20	88" @input="onInput"></textarea>
        </div>
      </div>

      <div class="panel panel-right">
        <div class="panel-head">
          <h2>生成结果</h2>
          <div class="head-actions">
            <button class="btn btn-sm" @click="copySql" :disabled="!sql">复制</button>
            <button class="btn btn-sm" @click="downloadSql" :disabled="!sql">下载 .sql</button>
          </div>
        </div>
        <div class="panel-body p0">
          <pre v-if="sql" class="sql-output">{{ sql }}</pre>
          <div v-else class="empty">点击「生成 SQL」后在此显示语句</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ExcelSql',
  data() {
    return {
      raw: '',
      mode: 'insert',
      db: 'oracle',
      targetTable: '',
      whereCond: '',
      allString: false,
      quoteAliasOn: true,
      sql: '',
      stats: '',
      columns: [],
    }
  },
  methods: {
    onInput() { this.sql = ''; this.stats = ''; this.syncColumns() },
    clearAll() {
      this.raw = ''; this.sql = ''; this.stats = ''
      this.targetTable = ''; this.whereCond = ''; this.columns = []
    },
    fillDemo() {
      this.raw = 'ID\t姓名\t出生日期\t分数\n1\t张三\t1990-01-01\t95.5\n2\t李四\t1992-05-20\t88\n3\t王五\t1988-11-11\tNULL'
      this.onInput()
    },
    colKey(i) { return i < 26 ? String.fromCharCode(97 + i) : 'col' + (i + 1) },
    detectDelimiter(line) {
      if (line.indexOf('\t') !== -1) return '\t'
      if (line.indexOf(';') !== -1) return ';'
      if (line.indexOf(',') !== -1) return ','
      return /\s{2,}/
    },
    parse() {
      const lines = this.raw.split(/\r?\n/).map(l => l.trim()).filter(l => l.length)
      if (lines.length < 2) return { error: '至少需要表头行 + 一行数据' }
      const delim = this.detectDelimiter(lines[0])
      const split = l => typeof delim === 'string'
        ? l.split(delim).map(s => s.trim())
        : l.split(delim).map(s => s.trim()).filter(s => s.length)
      const headers = split(lines[0])
      const rows = lines.slice(1).map(split)
      const ncol = headers.length
      const bad = rows.findIndex(r => r.length !== ncol)
      if (bad !== -1) return { error: `第 ${bad + 2} 行有 ${rows[bad].length} 列，与表头（${ncol} 列）不一致` }
      return { headers, rows }
    },
    autoDetectType(values) {
      const vals = values.filter(v => v !== '' && (v || '').toUpperCase() !== 'NULL')
      if (!vals.length) return 'text'
      const dateRe = /^\d{4}[-/]\d{1,2}[-/]\d{1,2}([ T]\d{1,2}:\d{2}(:\d{2})?)?$/
      if (vals.every(v => dateRe.test(v.trim()))) return 'date'
      if (vals.every(v => /^-?\d+(\.\d+)?$/.test(v.trim()))) return 'number'
      return 'text'
    },
    detLabel(t) {
      return ({ text: '文本', number: '数字', date: '日期' })[t] || '文本'
    },
    syncColumns() {
      const parsed = this.parse()
      if (parsed.error) { this.columns = []; return }
      const prev = {}
      this.columns.forEach(c => { prev[c.header] = c })
      this.columns = parsed.headers.map((h, i) => {
        const old = prev[h]
        const detected = this.autoDetectType(parsed.rows.map(r => r[i]))
        return {
          key: this.colKey(i),
          header: h,
          type: old ? old.type : 'auto',
          detected,
          start: old ? old.start : '',
          end: old ? old.end : '',
        }
      })
    },
    effType(c) { return c.type === 'auto' ? (c.detected || 'text') : c.type },
    quoteAlias(header, i) {
      const h = (header || '').trim() || ('COL' + (i + 1))
      if (!this.quoteAliasOn) return h
      if (this.db === 'oracle') return '"' + h.replace(/"/g, '""') + '"'
      return '`' + h.replace(/`/g, '``') + '`'
    },
    strLit(v) { return "'" + v.replace(/'/g, "''") + "'" },
    dateLit(val) {
      let v = String(val).trim()
      if (!v || v.toUpperCase() === 'NULL') return 'NULL'
      const norm = v.replace(/\//g, '-')
      const fmt = /\d{1,2}:\d{2}/.test(v) ? 'YYYY-MM-DD HH24:MI:SS' : 'YYYY-MM-DD'
      return this.db === 'oracle'
        ? `TO_DATE('${norm.replace(/'/g, "''")}','${fmt}')`
        : `DATE('${norm.replace(/'/g, "''")}')`
    },
    formatVal(cell, c) {
      let v = (cell == null ? '' : String(cell)).trim()
      if (v === '') return 'NULL'
      const t = this.effType(c)
      if (t === 'date') return this.dateLit(v)
      if (this.allString) return this.strLit(v)
      if (v.toUpperCase() === 'NULL') return 'NULL'
      if (t === 'number' || /^-?\d+(\.\d+)?$/.test(v)) return v
      return this.strLit(v)
    },
    buildWhere() {
      const conds = []
      if (this.whereCond.trim()) conds.push('(' + this.whereCond.trim() + ')')
      this.columns.forEach(c => {
        if (this.effType(c) === 'date') {
          if (c.start) conds.push(`${c.key} >= ${this.dateLit(c.start)}`)
          if (c.end) conds.push(`${c.key} <= ${this.dateLit(c.end)}`)
        }
      })
      return conds.length ? '\nWHERE ' + conds.join('\n  AND ') : ''
    },
    generate() {
      this.syncColumns()
      const parsed = this.parse()
      if (parsed.error) { this.sql = ''; this.stats = parsed.error; return }
      if (!this.columns.length) { this.stats = '未解析到列'; return }
      const { headers, rows } = parsed
      const keys = this.columns.map(c => c.key)
      const where = this.buildWhere()
      const tuples = rows
        .map(r => '  SELECT ' + r.map((cell, i) => this.formatVal(cell, this.columns[i])).join(', ') + ' FROM dual')
        .join('\n  UNION ALL\n')
      const withBlock =
`WITH t_excel (${keys.join(', ')}) AS (
${tuples}
)`
      let sql
      if (this.mode === 'with') {
        const sel = this.columns.map((c, i) => `${c.key} AS ${this.quoteAlias(c.header, i)}`).join(', ')
        sql = `${withBlock}\nSELECT ${sel}\nFROM t_excel${where};`
      } else {
        if (!this.targetTable.trim()) { this.stats = 'INSERT 模式需要填写目标表名'; return }
        const tcols = this.columns.map((c, i) => this.quoteAlias(c.header, i)).join(', ')
        sql =
`${withBlock}
INSERT INTO ${this.targetTable.trim()} (${tcols})
SELECT ${keys.join(', ')}\nFROM t_excel${where};`
      }
      this.sql = sql
      this.stats = `已解析 ${rows.length} 行 × ${headers.length} 列`
    },
    async copySql() {
      try {
        await navigator.clipboard.writeText(this.sql)
        this.$root.toastMsg('已复制 SQL', 'success')
      } catch (e) {
        this.$root.toastMsg('复制失败，请手动选择', 'error')
      }
    },
    downloadSql() {
      const blob = new Blob([this.sql], { type: 'text/plain;charset=utf-8' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = 'excel_to_sql.sql'
      a.click()
      URL.revokeObjectURL(a.href)
    },
  },
}
</script>

<style scoped>
.layout { display: flex; gap: 16px; align-items: stretch; }
.panel-left, .panel-right { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.paste-area {
  width: 100%; min-height: 380px; border: 0; resize: vertical; padding: 12px;
  font-family: 'Consolas', 'Courier New', monospace; font-size: 13px; line-height: 1.5;
  outline: none; box-sizing: border-box; color: #1e293b; background: #fff;
}
.sql-output {
  margin: 0; padding: 12px; min-height: 380px; white-space: pre-wrap; word-break: break-word;
  font-family: 'Consolas', 'Courier New', monospace; font-size: 13px; line-height: 1.5;
  color: #0f172a; background: #f8fafc;
}
.head-actions { display: flex; gap: 8px; }
.switch { display: inline-flex; align-items: center; gap: 4px; font-size: 13px; color: #475569; cursor: pointer; }
.switch select { margin-left: 4px; }
.filter-item.grow { flex: 1; }
.filter-item.grow .filter-control { width: 100%; }
.filter-control.sm { width: auto; padding: 4px 6px; font-size: 12px; }
.col-set { margin-bottom: 16px; border: 1px solid #e2e8f0; border-radius: 8px; background: #fff; }
.col-set-head { padding: 8px 12px; font-size: 13px; color: #475569; border-bottom: 1px solid #eef2f7; background: #f8fafc; }
.col-grid { max-height: 220px; overflow: auto; padding: 10px 12px; display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.col-card { display: inline-flex; align-items: center; gap: 6px; font-size: 13px; flex: 0 1 auto; padding: 4px 8px; border: 1px solid #eef2f7; border-radius: 6px; background: #f8fafc; }
.col-key { display: inline-block; min-width: 24px; text-align: center; font-family: monospace; background: #e2e8f0; color: #334155; border-radius: 4px; padding: 2px 6px; flex: none; }
.col-head { max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #0f172a; }
.tilde { color: #94a3b8; }
.det-tag { font-size: 11px; padding: 1px 6px; border-radius: 4px; flex: none; color: #334155; background: #e2e8f0; }
.det-tag.det-text { color: #475569; background: #e2e8f0; }
.det-tag.det-number { color: #9a3412; background: #ffedd5; }
.det-tag.det-date { color: #1d4ed8; background: #dbeafe; }
</style>
