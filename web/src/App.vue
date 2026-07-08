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
          <button class="logout-btn" @click="logout">退出</button>
          <span class="clock">{{ now }}</span>
        </div>
      </header>
      <section class="content"><router-view /></section>
    </main>
    <div class="sidebar-overlay" :class="{ show: sidebarOpen }" @click="closeSidebar"></div>
    <div v-if="toast.visible" class="toast" :class="toast.type">{{ toast.msg }}</div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      now: '',
      username: localStorage.getItem('username') || '',
      toast: { visible: false, msg: '', type: '' },
      loading: false,
      sidebarOpen: false,
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
    toastMsg(msg, type = '') {
      this.toast = { visible: true, msg, type }
      setTimeout(() => this.toast.visible = false, 2400)
    },
    onLoading(e) { this.loading = e.detail > 0 }
  },
  mounted() {
    this.tick(); setInterval(() => this.tick(), 1000)
    window.addEventListener('api-loading', this.onLoading)
  },
  beforeDestroy() { window.removeEventListener('api-loading', this.onLoading) },
  provide() { return { toast: this.toastMsg } }
}
</script>
