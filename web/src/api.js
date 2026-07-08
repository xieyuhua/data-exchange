import axios from 'axios'

const api = axios.create({ baseURL: '/api', timeout: 30000 })

// 全局 loading 计数：在途请求 > 0 时通过事件通知 App 显示遮罩，防止重复点击
let loadingCount = 0
function bumpLoading() {
  loadingCount++
  window.dispatchEvent(new CustomEvent('api-loading', { detail: loadingCount }))
}
function dropLoading() {
  loadingCount = Math.max(0, loadingCount - 1)
  window.dispatchEvent(new CustomEvent('api-loading', { detail: loadingCount }))
}

// 请求拦截：自动附带 JWT，并开启 loading
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  bumpLoading()
  return config
})

// 响应拦截：401 视为登录失效，清除 token 并跳转登录页
api.interceptors.response.use(
  resp => { dropLoading(); return resp },
  err => {
    dropLoading()
    if (err.response && err.response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      if (location.hash.indexOf('#/login') === -1) location.hash = '#/login'
    }
    return Promise.reject(err)
  }
)

export default {
  get(url, params) { return api.get(url, { params }).then(r => r.data) },
  post(url, data) { return api.post(url, data).then(r => r.data) },
  del(url) { return api.delete(url).then(r => r.data) },
  // 文件下载：返回完整 axios 响应（responseType=blob），由调用方判断成功/错误
  file(url, params) { return api.get(url, { params, responseType: 'blob' }) },
}
