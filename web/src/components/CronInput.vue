<template>
  <div class="cron-input">
    <!-- 单行：表达式输入框 + 高级开关 -->
    <div class="cron-line">
      <input
        :value="expr"
        @input="onRawInput"
        placeholder="0 2 * * *"
        class="cron-text"
        aria-label="cron 表达式"
        spellcheck="false"
      >
      <button type="button" class="cron-toggle" @click="advanced = !advanced">
        {{ advanced ? '收起 ▲' : '高级 ▼' }}
      </button>
    </div>

    <!-- 常用预设：按钮组，选中即生效并回写 -->
    <div class="cron-presets">
      <button
        v-for="p in presets"
        :key="p.expr"
        type="button"
        :class="['preset-chip', { active: expr === p.expr }]"
        @click="selectPreset(p.expr)"
      >{{ p.label }}</button>
    </div>

    <!-- 即时反馈：下次执行时间 -->
    <div class="cron-next" v-if="nextRun">下次执行：<b>{{ nextRun }}</b></div>
    <div class="cron-next err" v-else-if="nextError">表达式无效：{{ nextError }}</div>

    <!-- 高级：图形化构建 + 未来多次预览（默认折叠） -->
    <div class="cron-advanced" v-if="advanced">
      <div class="cron-fields">
        <label>分<input v-model="fields.min" @input="syncFromFields" placeholder="*"></label>
        <label>时<input v-model="fields.hour" @input="syncFromFields" placeholder="*"></label>
        <label>日<input v-model="fields.dom" @input="syncFromFields" placeholder="*"></label>
        <label>月<input v-model="fields.month" @input="syncFromFields" placeholder="*"></label>
        <label>周<input v-model="fields.dow" @input="syncFromFields" placeholder="*"></label>
      </div>
      <div class="hint">标准 cron 格式：分 时 日 月 周（空格分隔）。<code>*</code> 任意，<code>*/n</code> 每 n 个单位，<code>1,2</code> 枚举，<code>1-5</code> 区间。</div>

      <div class="cron-preview" v-if="previewList.length">
        <div class="cron-preview-title">未来 {{ previewList.length }} 次执行时间：</div>
        <div v-for="(t, i) in previewList" :key="i" class="cron-preview-item">{{ i + 1 }}. {{ t }}</div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'CronInput',
  props: {
    value: { type: String, default: '' }
  },
  data() {
    return {
      expr: this.value || '0 2 * * *',
      advanced: false,
      touched: false,
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
  watch: {
    value(v) {
      // 用户已交互过则尊重用户当前输入，不被外部回填覆盖
      if (this.touched) return
      if (v !== this.expr) {
        this.expr = v || '0 2 * * *'
        this.parseToFields()
        this.fetchNext()
      }
    }
  },
  mounted() {
    // v-if 渲染保证此时 value 已是父组件真实值
    if (this.value) {
      this.expr = this.value
    }
    this.parseToFields()
    this.fetchNext()
  },
  methods: {
    emit() {
      this.$emit('input', this.expr)
    },
    // 选择预设：直接设置表达式并回写
    selectPreset(expr) {
      this.touched = true
      this.expr = expr
      this.parseToFields()
      this.emit()
      this.fetchNext()
    },
    onRawInput(e) {
      this.touched = true
      this.expr = e.target.value
      this.parseToFields()
      this.emit()
      this.scheduleFetch()
    },
    parseToFields() {
      const parts = (this.expr || '').trim().split(/\s+/)
      if (parts.length === 5) {
        this.fields = { min: parts[0], hour: parts[1], dom: parts[2], month: parts[3], dow: parts[4] }
      }
    },
    syncFromFields() {
      this.touched = true
      const f = this.fields
      this.expr = [f.min, f.hour, f.dom, f.month, f.dow].join(' ').trim()
      this.emit()
      this.fetchNext()
    },
    // 输入时防抖请求预览，避免频繁打接口
    scheduleFetch() {
      if (this._debounce) clearTimeout(this._debounce)
      this._debounce = setTimeout(() => this.fetchNext(), 500)
    },
    // 拉取「下次执行」做即时反馈
    async fetchNext() {
      const e = (this.expr || '').trim()
      if (!e) { this.nextRun = ''; this.nextError = ''; this.previewList = []; return }
      try {
        const r = await api.get('/tasks/cron-next', { expr: e, n: this.advanced ? 5 : 1 })
        if (r.code === 0) {
          const list = r.data || []
          this.nextRun = list[0] || ''
          this.nextError = ''
          this.previewList = this.advanced ? list : []
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
.cron-input {
  border: 1px solid var(--border, #e2e8f0);
  border-radius: 8px;
  padding: 8px 10px;
  background: #fafbfc;
}
.cron-line {
  display: flex;
  gap: 8px;
  align-items: center;
}
.cron-text {
  flex: 1;
  min-width: 0;
  padding: 7px 10px;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  font-family: monospace;
  font-size: 14px;
}
.cron-toggle {
  border: 1px solid #cbd5e1;
  background: #fff;
  border-radius: 6px;
  padding: 7px 10px;
  font-size: 13px;
  cursor: pointer;
  color: #334155;
  white-space: nowrap;
}
.cron-toggle:hover { border-color: #3b82f6; color: #3b82f6; }
.cron-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
.preset-chip {
  border: 1px solid #cbd5e1;
  background: #fff;
  border-radius: 999px;
  padding: 4px 12px;
  font-size: 12.5px;
  cursor: pointer;
  color: #475569;
  transition: all .12s;
}
.preset-chip:hover { border-color: #3b82f6; color: #3b82f6; }
.preset-chip.active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
}
.cron-next {
  margin-top: 8px;
  font-size: 13px;
  color: #0369a1;
}
.cron-next.err { color: #b91c1c; }
.cron-advanced {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed #e2e8f0;
}
.cron-fields {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.cron-fields label {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  color: #64748b;
  gap: 4px;
}
.cron-fields input {
  width: 64px;
  padding: 6px 8px;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  font-size: 13px;
}
.cron-preview {
  margin-top: 10px;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 13px;
  color: #065f46;
}
.cron-preview-title { font-weight: 600; margin-bottom: 4px; }
.cron-preview-item { font-family: monospace; }
.hint { margin-top: 8px; font-size: 12px; color: #94a3b8; }
.hint code {
  background: #e2e8f0;
  padding: 0 4px;
  border-radius: 4px;
  font-family: monospace;
}
</style>
