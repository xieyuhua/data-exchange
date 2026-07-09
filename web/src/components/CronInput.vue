<template>
  <div class="cron-field">
    <!-- 主输入行：表达式直接绑定到父组件的 value，显示即提交值 -->
    <div class="cron-main">
      <input
        class="cron-text"
        v-model="model"
        placeholder="分 时 日 月 周，如 0 2 * * *"
        spellcheck="false"
        aria-label="cron 表达式"
      >
      <button type="button" class="cron-adv-btn" @click="advanced = !advanced">
        {{ advanced ? '收起 ▲' : '高级 ▼' }}
      </button>
    </div>

    <!-- 常用预设 -->
    <div class="cron-presets">
      <button
        v-for="p in presets"
        :key="p.expr"
        type="button"
        :class="['cron-chip', { active: expr === p.expr }]"
        @click="applyPreset(p.expr)"
      >{{ p.label }}</button>
    </div>

    <!-- 使用说明 -->
    <div class="cron-help">
      <button type="button" class="cron-help-toggle" @click="showHelp = !showHelp">
        {{ showHelp ? '收起使用说明 ▲' : '使用说明 ▼' }}
      </button>
      <div class="cron-help-body" v-if="showHelp">
        <p>标准 cron 表达式由 5 个字段组成，以空格分隔：<code>分 时 日 月 周</code></p>
        <ul>
          <li><code>*</code> 任意值（每分/每时/每日…）</li>
          <li><code>*/n</code> 每 n 个单位，如 <code>*/15</code> 表示每 15 分钟</li>
          <li><code>1,2,3</code> 枚举，如 <code>1,15</code> 表示第 1 和 15 日</li>
          <li><code>1-5</code> 区间，如 <code>1-5</code> 表示周一到周五</li>
          <li>示例：<code>0 2 * * *</code> 每天 02:00；<code>0 9 * * 1-5</code> 工作日 09:00</li>
        </ul>
      </div>
    </div>

    <!-- 即时反馈：下次执行 + 最近 5 次执行时间 -->
    <div class="cron-feedback">
      <span v-if="nextRun" class="cron-ok">下次执行：<b>{{ nextRun }}</b></span>
      <span v-else-if="nextError" class="cron-bad">表达式无效：{{ nextError }}</span>
      <span v-else class="cron-muted">—</span>
    </div>
    <div class="cron-preview" v-if="previewList.length">
      <div class="cron-preview-title">最近 5 次执行时间：</div>
      <div v-for="(t, i) in previewList" :key="i" class="cron-preview-item">{{ i + 1 }}. {{ t }}</div>
    </div>

    <!-- 高级：图形化构建 -->
    <div class="cron-adv" v-if="advanced">
      <div class="cron-adv-grid">
        <label>分钟<input v-model="fields.min" @input="applyFields" placeholder="*"></label>
        <label>小时<input v-model="fields.hour" @input="applyFields" placeholder="*"></label>
        <label>日<input v-model="fields.dom" @input="applyFields" placeholder="*"></label>
        <label>月<input v-model="fields.month" @input="applyFields" placeholder="*"></label>
        <label>周<input v-model="fields.dow" @input="applyFields" placeholder="*"></label>
      </div>
      <div class="hint">
        标准 cron：分 时 日 月 周（空格分隔）。<code>*</code> 任意，<code>*/n</code> 每 n 个单位，<code>1,2</code> 枚举，<code>1-5</code> 区间。
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'

function normalize(e) {
  return (e || '').replace(/\s+/g, ' ').trim()
}
const DEFAULT_EXPR = '0 2 * * *'

export default {
  name: 'CronInput',
  props: {
    // Vue 3 v-model 约定：modelValue 接收父组件值
    modelValue: { type: String, default: '' }
  },
  data() {
    return {
      advanced: false,
      showHelp: false,
      // 本地 expr 为唯一真值：输入框显示、预设/高级字段、回写均以此为准，
      // 避免依赖父组件 modelValue 回流导致的「点击预设不生效」
      expr: normalize(this.modelValue) || DEFAULT_EXPR,
      fields: { min: '0', hour: '2', dom: '*', month: '*', dow: '*' },
      presets: [
        { label: '每天 00:00', expr: '0 0 * * *' },
        { label: '每天 02:00', expr: '0 2 * * *' },
        { label: '每天 12:00', expr: '0 12 * * *' },
        { label: '每小时', expr: '0 * * * *' },
        { label: '每15分钟', expr: '*/15 * * * *' },
        { label: '每30分钟', expr: '*/30 * * * *' },
        { label: '每周一 02:00', expr: '0 2 * * 1' },
        { label: '每月1号 02:00', expr: '0 2 1 * *' }
      ],
      nextRun: '',
      nextError: '',
      previewList: [],
      _debounce: null
    }
  },
  computed: {
    // 输入框绑定本地 expr：显示即提交值，且始终即时反映用户操作
    model: {
      get() { return this.expr },
      set(v) { this.setExpr(v) }
    }
  },
  watch: {
    // 父组件回填（含异步加载已有任务）时同步到本地 expr
    modelValue(v) {
      const nv = normalize(v) || DEFAULT_EXPR
      if (nv !== this.expr) {
        this.expr = nv
        this.parseToFields()
        this.fetchNext()
      }
    }
  },
  methods: {
    // 统一设置表达式：更新本地 + 回写父组件（Vue 3 用 update:modelValue）
    setExpr(v) {
      const nv = normalize(v) || DEFAULT_EXPR
      if (nv === this.expr) return
      this.expr = nv
      this.$emit('update:modelValue', nv)
      this.scheduleFetch()
    },
    applyPreset(expr) {
      this.setExpr(expr)
      this.parseToFields()
      this.fetchNext()
    },
    parseToFields() {
      const parts = normalize(this.expr).split(/\s+/).filter(Boolean)
      if (parts.length === 5) {
        this.fields = { min: parts[0], hour: parts[1], dom: parts[2], month: parts[3], dow: parts[4] }
      }
    },
    applyFields() {
      this.setExpr([this.fields.min, this.fields.hour, this.fields.dom, this.fields.month, this.fields.dow].join(' '))
      this.fetchNext()
    },
    scheduleFetch() {
      if (this._debounce) clearTimeout(this._debounce)
      this._debounce = setTimeout(() => this.fetchNext(), 500)
    },
    async fetchNext() {
      const e = normalize(this.expr)
      if (!e) { this.nextRun = ''; this.nextError = '表达式不能为空'; this.previewList = []; return }
      try {
        const r = await api.get('/tasks/cron-next', { expr: e, n: 5 })
        if (r.code === 0) {
          const list = r.data || []
          this.nextRun = list[0] || ''
          this.nextError = ''
          this.previewList = list
        } else {
          this.nextRun = ''
          this.nextError = r.message
          this.previewList = []
        }
      } catch (err) {
        this.nextRun = ''
        this.nextError = '无法获取预览'
        this.previewList = []
      }
    }
  }
}
</script>

<style scoped>
.cron-field {
  border: 1px solid var(--border, #e2e8f0);
  border-radius: 10px;
  padding: 12px 14px;
  background: #fafbfc;
}
.cron-main {
  display: flex;
  gap: 10px;
  align-items: stretch;
}
.cron-text {
  flex: 1;
  min-width: 0;
  padding: 9px 12px;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 14px;
  color: #0f172a;
  background: #fff;
  transition: border-color .15s, box-shadow .15s;
}
.cron-text:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, .15);
}
.cron-adv-btn {
  flex: none;
  border: 1px solid #cbd5e1;
  background: #fff;
  border-radius: 8px;
  padding: 0 14px;
  font-size: 13px;
  cursor: pointer;
  color: #334155;
  white-space: nowrap;
  transition: all .12s;
}
.cron-adv-btn:hover { border-color: #3b82f6; color: #3b82f6; }

.cron-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.cron-chip {
  border: 1px solid #cbd5e1;
  background: #fff;
  border-radius: 999px;
  padding: 5px 13px;
  font-size: 12.5px;
  cursor: pointer;
  color: #475569;
  transition: all .12s;
}
.cron-chip:hover { border-color: #3b82f6; color: #3b82f6; }
.cron-chip.active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
  box-shadow: 0 1px 4px rgba(59, 130, 246, .35);
}

.cron-feedback { margin-top: 10px; font-size: 13px; min-height: 18px; }
.cron-ok { color: #0369a1; }
.cron-ok b { color: #0c4a6e; font-weight: 600; }
.cron-bad { color: #b91c1c; }
.cron-muted { color: #94a3b8; }

.cron-help { margin-top: 10px; }
.cron-help-toggle {
  border: none;
  background: transparent;
  color: #3b82f6;
  font-size: 12.5px;
  cursor: pointer;
  padding: 0;
}
.cron-help-toggle:hover { text-decoration: underline; }
.cron-help-body {
  margin-top: 8px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 12.5px;
  color: #475569;
  line-height: 1.7;
}
.cron-help-body p { margin: 0 0 6px; }
.cron-help-body ul { margin: 0; padding-left: 18px; }
.cron-help-body code {
  background: #e2e8f0;
  padding: 0 4px;
  border-radius: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.cron-adv {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px dashed #e2e8f0;
}
.cron-adv-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.cron-adv-grid label {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  color: #64748b;
  gap: 5px;
}
.cron-adv-grid input {
  width: 72px;
  padding: 7px 9px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  background: #fff;
}
.cron-adv-grid input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, .15);
}

.cron-preview {
  margin-top: 12px;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13px;
  color: #065f46;
}
.cron-preview-title { font-weight: 600; margin-bottom: 6px; }
.cron-preview-item { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; line-height: 1.7; }

.hint { margin-top: 10px; font-size: 12px; color: #94a3b8; }
.hint code {
  background: #e2e8f0;
  padding: 0 4px;
  border-radius: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
</style>
