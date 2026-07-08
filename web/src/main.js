import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

import Dashboard from './views/Dashboard.vue'
import Vendors from './views/Vendors.vue'
import Tasks from './views/Tasks.vue'
import DB from './views/DB.vue'
import FTP from './views/FTP.vue'
import Constants from './views/Constants.vue'
import Configs from './views/Configs.vue'
import Logs from './views/Logs.vue'
import Files from './views/Files.vue'

const routes = [
  { path: '/', name: 'dashboard', component: Dashboard, meta: { title: '仪表盘' } },
  { path: '/vendors', name: 'vendors', component: Vendors, meta: { title: '厂家管理' } },
  { path: '/tasks', name: 'tasks', component: Tasks, meta: { title: 'SQL任务' } },
  { path: '/db', name: 'db', component: DB, meta: { title: '数据库连接' } },
  { path: '/ftp', name: 'ftp', component: FTP, meta: { title: 'FTP/SFTP账号' } },
  { path: '/constants', name: 'constants', component: Constants, meta: { title: '系统常量' } },
  { path: '/configs', name: 'configs', component: Configs, meta: { title: '系统配置' } },
  { path: '/logs', name: 'logs', component: Logs, meta: { title: '执行日志' } },
  { path: '/files', name: 'files', component: Files, meta: { title: '文件管理' } },
]

const router = createRouter({ history: createWebHashHistory(), routes })

const app = createApp(App)
app.use(router)
app.mount('#app')
