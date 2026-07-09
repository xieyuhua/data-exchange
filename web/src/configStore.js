import api from './api'

const LS_KEY = 'page_size'

// 缓存的分页大小，模块加载时优先读本地存储，缺省 20
let cached = parseInt(localStorage.getItem(LS_KEY) || '', 10)
if (!Number.isFinite(cached) || cached < 1) cached = 20

// getPageSize 返回当前生效的分页大小
export function getPageSize() {
  return cached
}

// initPageSize 从后端 /configs 拉取 page_size 并缓存（应用在启动时调用）
export async function initPageSize() {
  try {
    const r = await api.get('/configs')
    const list = (r && r.data) || []
    const item = list.find(c => c.config_key === 'page_size')
    if (item && item.config_value) {
      const n = parseInt(item.config_value, 10)
      if (Number.isFinite(n) && n >= 1) {
        cached = n
        localStorage.setItem(LS_KEY, String(n))
      }
    }
  } catch (e) {
    // 保持当前值不变
  }
}

// refreshPageSize 在系统配置保存后调用，刷新分页大小缓存
export function refreshPageSize() {
  return initPageSize()
}
