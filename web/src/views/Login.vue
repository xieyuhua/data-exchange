<template>
  <div class="login-wrap">
    <form class="login-card" @submit.prevent="doLogin">
      <div class="login-logo"><span class="logo-icon">&#8660;</span> 数据交换系统</div>
      <div class="login-sub">请登录后继续</div>
      <label>用户名</label>
      <input v-model.trim="form.username" type="text" autocomplete="username" placeholder="用户名" :disabled="loading" />
      <label>密码</label>
      <input v-model.trim="form.password" type="password" autocomplete="current-password" placeholder="密码" :disabled="loading" @keyup.enter="doLogin" />
      <div v-if="err" class="login-err">{{ err }}</div>
      <button type="submit" :disabled="loading">{{ loading ? '登录中...' : '登 录' }}</button>
    </form>
  </div>
</template>

<script>
import api from '../api'
export default {
  data() {
    return { form: { username: '', password: '' }, loading: false, err: '' }
  },
  methods: {
    async doLogin() {
      if (!this.form.username || !this.form.password) { this.err = '请输入用户名和密码'; return }
      this.loading = true; this.err = ''
      try {
        const r = await api.post('/auth/login', this.form)
        if (r.code === 0) {
          localStorage.setItem('token', r.data.token)
          localStorage.setItem('username', r.data.username)
          localStorage.setItem('role', r.data.role || '')
          this.$router.replace(this.$route.query.redirect || '/')
        } else {
          this.err = r.message || '登录失败'
        }
      } catch (e) {
        this.err = '网络错误，请稍后重试'
      } finally {
        this.loading = false
      }
    }
  }
}
</script>

<style scoped>
.login-wrap { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg,#1e293b,#0f172a); }
.login-card { width: 340px; background: #fff; border-radius: 12px; padding: 28px 26px; box-shadow: 0 12px 40px rgba(0,0,0,.35); }
.login-logo { font-size: 20px; font-weight: 700; text-align: center; color: #1e293b; }
.logo-icon { color: #2563eb; margin-right: 4px; }
.login-sub { text-align: center; color: #94a3b8; margin: 6px 0 18px; font-size: 13px; }
.login-card label { display: block; font-size: 13px; color: #475569; margin: 12px 0 6px; }
.login-card input { width: 100%; box-sizing: border-box; padding: 10px 12px; border: 1px solid #cbd5e1; border-radius: 8px; font-size: 14px; outline: none; }
.login-card input:focus { border-color: #2563eb; }
.login-err { color: #dc2626; font-size: 13px; margin-top: 12px; text-align: center; }
.login-card button { width: 100%; margin-top: 18px; padding: 11px; border: none; border-radius: 8px; background: #2563eb; color: #fff; font-size: 15px; cursor: pointer; }
.login-card button:disabled { opacity: .6; cursor: not-allowed; }
</style>
