<template>
  <div v-if="$route.name === 'login'">
    <router-view />
  </div>
  <div v-else id="app-root">
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="logo"><span class="logo-icon">&#8660;</span><span class="logo-text">数据交换系统</span></div>
      <nav class="nav">
        <router-link v-for="r in nav" :key="r.path" :to="r.path" class="nav-item" active-class="active" @click.native="closeSidebar">
          <span class="nav-ico">{{ r.icon }}</span>{{ r.label }}
        </router-link>
      </nav>
    </aside>
    <main class="main">
      <header class="topbar">
        <button class="menu-toggle" @click="toggleSidebar">&#9776;</button>
        <h1>{{ $route.meta.title }}</h1>
        <div class="topbar-right">
          <span class="user">&#128100; {{ username }}</span>
          <button class="logout-btn" @click="openPwd">修改密码</button>
          <button class="logout-btn" @click="logout">退出</button>
          <span class="clock">{{ now }}</span>
        </div>
      </header>
      <section class="content"><router-view /></section>
    </main>
    <div class="sidebar-overlay" :class="{ show: sidebarOpen }" @click="closeSidebar"></div>
    <div v-if="toast.visible" class="toast" :class="toast.type">{{ toast.msg }}</div>

    <div v-if="showPwd" class="modal-mask" @click.self="showPwd=false">
      <div class="modal">
        <div class="modal-head"><h3>修改密码</h3><button class="modal-close" @click="showPwd=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-row full"><label>原密码 *</label><input v-model="pwdForm.old_password" type="password" autocomplete="current-password"></div>
          <div class="form-row full"><label>新密码 *</label><input v-model="pwdForm.new_password" type="password" autocomplete="new-password" placeholder="至少 6 位"></div>
          <div class="form-row full"><label>确认新密码 *</label><input v-model="pwdForm.confirm" type="password" autocomplete="new-password"></div>
          <div v-if="pwdErr" class="err-tip">{{ pwdErr }}</div>
        </div>
        <div class="modal-foot">
          <button class="btn" @click="showPwd=false">取消</button>
          <button class="btn btn-primary" :disabled="pwdSaving" @click="savePwd">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { initPageSize } from './configStore'
import api from './api'
export default {
  data() {
    return {
      now: '',
      username: localStorage.getItem('username') || '',
      toast: { visible: false, msg: '', type: '' },
      loading: false,
      sidebarOpen: false,
      showPwd: false,
      pwdSaving: false,
      pwdErr: '',
      pwdForm: { old_password: '', new_password: '', confirm: '' },
      nav: [
        { path: '/', label: '仪表盘', icon: '\u25A3' },
        { path: '/vendors', label: '厂家管理', icon: '\u25A4' },
        { path: '/tasks', label: 'SQL任务', icon: '\u25A6' },
        { path: '/db', label: '数据库连接', icon: '\u26C1' },
        { path: '/ftp', label: 'FTP/SFTP账号', icon: '\u21EA' },
        { path: '/constants', label: '系统常量', icon: '\u2736' },
        { path: '/configs', label: '系统配置', icon: '\u2699' },
        { path: '/logs', label: '执行日志', icon: '\u2630' },
        { path: '/files', label: '文件管理', icon: '\u26C0' },
      ]
    }
  },
  methods: {
    tick() {
      const d = new Date(), p = n => String(n).padStart(2, '0')
      this.now = `${d.getFullYear()}-${p(d.getMonth()+1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
    },
    toggleSidebar() { this.sidebarOpen = !this.sidebarOpen },
    closeSidebar() { this.sidebarOpen = false },
    logout() {
      if (!confirm('确认退出登录？')) return
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      this.username = ''
      location.hash = '#/login'
    },
    openPwd() { this.pwdErr = ''; this.pwdForm = { old_password: '', new_password: '', confirm: '' }; this.showPwd = true },
    async savePwd() {
      this.pwdErr = ''
      if (!this.pwdForm.old_password || !this.pwdForm.new_password) { this.pwdErr = '请填写原密码和新密码'; return }
      if (this.pwdForm.new_password.length < 6) { this.pwdErr = '新密码长度至少 6 位'; return }
      if (this.pwdForm.new_password !== this.pwdForm.confirm) { this.pwdErr = '两次输入的新密码不一致'; return }
      this.pwdSaving = true
      try {
        const r = await api.post('/auth/change-password', this.pwdForm)
        if (r.code === 0) { this.showPwd = false; this.toastMsg('密码修改成功', 'success') }
        else this.pwdErr = r.message
      } catch (e) { this.pwdErr = '修改失败，请重试' }
      finally { this.pwdSaving = false }
    },
    toastMsg(msg, type = '') {
      this.toast = { visible: true, msg, type }
      setTimeout(() => this.toast.visible = false, 2400)
    },
    onLoading(e) { this.loading = e.detail > 0 }
  },
  mounted() {
    this.tick(); setInterval(() => this.tick(), 1000)
    window.addEventListener('api-loading', this.onLoading)
    initPageSize()
  },
  beforeDestroy() { window.removeEventListener('api-loading', this.onLoading) },
  provide() { return { toast: this.toastMsg } }
}
</script>
