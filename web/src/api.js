import axios from 'axios'

const api = axios.create({ baseURL: '/api', timeout: 30000 })

export default {
  get(url, params) { return api.get(url, { params }).then(r => r.data) },
  post(url, data) { return api.post(url, data).then(r => r.data) },
  del(url) { return api.delete(url).then(r => r.data) },
}
