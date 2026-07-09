import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'
import './style.css'
import api from './api'

import Login from './views/Login.vue'
import Dashboard from './views/Dashboard.vue'
import Vendors from './views/Vendors.vue'
import Tasks from './views/Tasks.vue'
import TaskForm from './views/TaskForm.vue'
import DB from './views/DB.vue'
import FTP from './views/FTP.vue'
import FTPFiles from './views/FTPFiles.vue'
import Constants from './views/Constants.vue'
import Configs from './views/Configs.vue'
import Logs from './views/Logs.vue'
import Files from './views/Files.vue'

const routes = [
  { path: '/login', name: 'login', component: Login, meta: { public: true } },
  { path: '/', name: 'dashboard', component: Dashboard, meta: { title: '仪表盘' } },
  { path: '/vendors', name: 'vendors', component: Vendors, meta: { title: '厂家管理' } },
  { path: '/tasks', name: 'tasks', component: Tasks, meta: { title: 'SQL任务' } },
  { path: '/tasks/new', name: 'task-new', component: TaskForm, meta: { title: '新增任务' } },
  { path: '/tasks/edit/:id', name: 'task-edit', component: TaskForm, meta: { title: '编辑任务' } },
  { path: '/db', name: 'db', component: DB, meta: { title: '数据库连接' } },
  { path: '/ftp', name: 'ftp', component: FTP, meta: { title: 'FTP/SFTP账号' } },
  { path: '/ftp/:id/files', name: 'ftp-files', component: FTPFiles, meta: { title: '远程文件' } },
  { path: '/constants', name: 'constants', component: Constants, meta: { title: '系统常量' } },
  { path: '/configs', name: 'configs', component: Configs, meta: { title: '系统配置' } },
  { path: '/logs', name: 'logs', component: Logs, meta: { title: '执行日志' } },
  { path: '/files', name: 'files', component: Files, meta: { title: '文件管理' } },
]

const router = createRouter({ history: createWebHashHistory(), routes })

// 路由守卫：未登录跳转到登录页
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.public) {
    if (token) next('/')
    else next()
  } else {
    if (token) next()
    else next({ path: '/login', query: { redirect: to.fullPath } })
  }
})

const app = createApp(App)
app.use(router)
app.mount('#app')
