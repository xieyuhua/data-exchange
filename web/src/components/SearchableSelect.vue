<template>
  <div class="ss" :class="{ open: open }" v-click-outside="close">
    <div class="ss-control" @click="toggle">
      <span v-if="display" class="ss-value">{{ display }}</span>
      <span v-else class="ss-placeholder">{{ placeholder }}</span>
      <span class="ss-caret">▾</span>
    </div>
    <div v-if="open" class="ss-panel">
      <input
        ref="search"
        v-model="q"
        class="ss-search"
        :placeholder="searchPlaceholder"
        @input="onInput"
      >
      <div class="ss-options">
        <div
          v-if="allowClear"
          class="ss-option"
          :class="{ 'ss-active': modelValue === '' || modelValue === null }"
          @click="pick('')"
        >{{ clearLabel }}</div>
        <div
          v-for="o in filtered"
          :key="o.value"
          class="ss-option"
          :class="{ 'ss-active': String(o.value) === String(modelValue) }"
          @click="pick(o.value)"
        >
          <span>{{ o.label }}</span>
          <span v-if="o.hint" class="ss-hint">{{ o.hint }}</span>
        </div>
        <div v-if="!filtered.length" class="ss-empty">无匹配项</div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SearchableSelect',
  props: {
    modelValue: { default: '' },
    options: { type: Array, default: () => [] },
    placeholder: { type: String, default: '请选择' },
    searchPlaceholder: { type: String, default: '搜索…' },
    clearLabel: { type: String, default: '全部' },
    allowClear: { type: Boolean, default: true },
    filterKey: { type: String, default: 'label' }
  },
  data() { return { open: false, q: '' } },
  computed: {
    display() {
      const o = this.options.find(x => String(x.value) === String(this.modelValue))
      return o ? o.label : ''
    },
    filtered() {
      const kw = this.q.trim().toLowerCase()
      if (!kw) return this.options
      return this.options.filter(o => {
        const hay = (o[this.filterKey] || '') + ' ' + (o.hint || '')
        return hay.toLowerCase().includes(kw)
      })
    }
  },
  methods: {
    toggle() {
      this.open = !this.open
      if (this.open) this.$nextTick(() => this.$refs.search && this.$refs.search.focus())
    },
    close() { this.open = false; this.q = '' },
    onInput() {},
    pick(v) {
      this.$emit('update:modelValue', v)
      this.$emit('change', v)
      this.close()
    }
  },
  directives: {
    clickOutside: {
      bind(el, binding) {
        el._ssHandler = e => { if (!el.contains(e.target)) binding.value() }
        document.addEventListener('click', el._ssHandler)
      },
      unbind(el) {
        document.removeEventListener('click', el._ssHandler)
      }
    }
  }
}
</script>

<style scoped>
.ss { position: relative; flex: 1; min-width: 0; }
.ss-control {
  height: 38px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  background: #fff;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 14px;
  transition: border-color .15s, box-shadow .15s;
}
.ss.open .ss-control { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-bg); }
.ss-value { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ss-placeholder { flex: 1; color: var(--text-muted); }
.ss-caret { color: var(--text-muted); font-size: 12px; }
.ss-panel {
  position: absolute;
  top: calc(100% + 4px);
  left: 0; right: 0;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow);
  z-index: 60;
  overflow: hidden;
}
.ss-search {
  width: 100%;
  box-sizing: border-box;
  border: none;
  border-bottom: 1px solid var(--border);
  padding: 9px 12px;
  font-size: 14px;
  outline: none;
}
.ss-options { max-height: 240px; overflow-y: auto; padding: 4px; }
.ss-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 14px;
}
.ss-option:hover { background: var(--bg); }
.ss-option.ss-active { background: var(--primary-bg); color: var(--primary); font-weight: 600; }
.ss-hint { color: var(--text-muted); font-size: 12px; font-family: var(--mono); }
.ss-empty { padding: 12px; text-align: center; color: var(--text-muted); font-size: 13px; }
</style>
