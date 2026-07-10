<template>
  <div>
    <div class="filter-bar">
      <div class="filter-item">
        <input v-model.trim="keyword" class="filter-control" placeholder="搜索用户名 / 昵称…" @keyup.enter="load(1)">
      </div>
      <button class="btn" @click="load(1)">搜索</button>
      <div class="spacer"></div>
      <button class="btn btn-primary" @click="openForm(null)">+ 新增用户</button>
    </div>

    <div class="panel">
      <div class="panel-head"><h2>用户管理</h2></div>
      <div class="panel-body p0">
        <div class="table-scroll" v-if="list.length">
          <table>
            <thead><tr><th>ID</th><th>用户名</th><th>昵称</th><th>角色</th><th>状态</th><th>创建时间</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="u in list" :key="u.id">
                <td>{{ u.id }}</td>
                <td class="cell-mono">{{ u.username }}</td>
                <td>{{ u.nickname || '—' }}</td>
                <td><span class="badge" :class="u.role === 'admin' ? 'badge-info' : ''">{{ u.role === 'admin' ? '管理员' : '只读' }}</span></td>
                <td><span class="badge" :class="u.status === 1 ? 'badge-on' : 'badge-off'">{{ u.status === 1 ? '启用' : '禁用' }}</span></td>
                <td class="cell-mono muted">{{ u.created_at }}</td>
                <td class="op-cell">
                  <button class="btn btn-ghost btn-sm" @click="openForm(u)">编辑</button>
                  <button class="btn btn-ghost btn-sm" @click="openReset(u)">重置密码</button>
                  <button class="btn btn-danger btn-sm" @click="del(u)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty">暂无用户</div>
      </div>
      <div class="pager" v-if="total > pageSize">
        <button class="btn btn-sm" :disabled="page <= 1" @click="load(page - 1)">上一页</button>
        <span class="muted">第 {{ page }} / {{ Math.ceil(total / pageSize) }} 页（共 {{ total }} 条）</span>
        <button class="btn btn-sm" :disabled="page >= Math.ceil(total / pageSize)" @click="load(page + 1)">下一页</button>
      </div>
    </div>

    <!-- 新增/编辑用户 -->
    <div v-if="showForm" class="modal-mask" @click.self="showForm=false">
      <div class="modal">
        <div class="modal-head"><h3>{{ editing ? '编辑用户' : '新增用户' }}</h3><button class="modal-close" @click="showForm=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-row full"><label>用户名 *</label><input v-model.trim="form.username" :disabled="editing" placeholder="登录用户名"></div>
          <div class="form-row full" v-if="!editing"><label>密码 *</label><input v-model="form.password" type="password" placeholder="至少 6 位"></div>
          <div class="form-row full"><label>昵称</label><input v-model.trim="form.nickname" placeholder="显示名称"></div>
          <div class="form-row full"><label>角色</label>
            <select v-model="form.role" class="inp">
              <option value="viewer">只读（viewer）</option>
              <option value="admin">管理员（admin）</option>
            </select>
          </div>
          <div class="form-row full" v-if="editing"><label>状态</label>
            <select v-model.number="form.status" class="inp">
              <option :value="1">启用</option>
              <option :value="0">禁用</option>
            </select>
          </div>
          <div v-if="formErr" class="err-tip">{{ formErr }}</div>
        </div>
        <div class="modal-foot"><button class="btn" @click="showForm=false">取消</button><button class="btn btn-primary" :disabled="saving" @click="save">保存</button></div>
      </div>
    </div>

    <!-- 重置密码 -->
    <div v-if="showReset" class="modal-mask" @click.self="showReset=false">
      <div class="modal">
        <div class="modal-head"><h3>重置密码 · {{ resetUser && resetUser.username }}</h3><button class="modal-close" @click="showReset=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-row full"><label>新密码 *</label><input v-model="resetPwd" type="password" placeholder="至少 6 位"></div>
          <div v-if="resetErr" class="err-tip">{{ resetErr }}</div>
        </div>
        <div class="modal-foot"><button class="btn" @click="showReset=false">取消</button><button class="btn btn-primary" :disabled="saving" @click="doReset">确定重置</button></div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
import { getPageSize } from '../configStore'
export default {
  data() {
    return {
      list: [], total: 0, page: 1, pageSize: getPageSize(), keyword: '',
      showForm: false, editing: false, saving: false, formErr: '',
      form: { id: 0, username: '', password: '', nickname: '', role: 'viewer', status: 1 },
      showReset: false, resetUser: null, resetPwd: '', resetErr: ''
    }
  },
  inject: ['toast'],
  async mounted() { await this.load(1) },
  methods: {
    async load(p) {
      this.page = p || this.page
      const r = await api.get('/users', { page: this.page, page_size: this.pageSize, keyword: this.keyword })
      this.list = r.data || []
      this.total = r.total || 0
    },
    openForm(u) {
      this.editing = !!u
      this.formErr = ''
      this.form = u
        ? { id: u.id, username: u.username, password: '', nickname: u.nickname, role: u.role, status: u.status }
        : { id: 0, username: '', password: '', nickname: '', role: 'viewer', status: 1 }
      this.showForm = true
    },
    async save() {
      this.formErr = ''
      if (!this.editing && !this.form.username) { this.formErr = '用户名不能为空'; return }
      if (!this.editing && (!this.form.password || this.form.password.length < 6)) { this.formErr = '密码长度至少 6 位'; return }
      this.saving = true
      try {
        let r
        if (this.editing) {
          r = await api.put('/users/' + this.form.id, { nickname: this.form.nickname, role: this.form.role, status: this.form.status })
        } else {
          r = await api.post('/users', { username: this.form.username, password: this.form.password, nickname: this.form.nickname, role: this.form.role })
        }
        if (r.code === 0) { this.showForm = false; this.toast('已保存', 'success'); this.load() }
        else this.formErr = r.message
      } catch (e) { this.formErr = '保存失败，请重试' }
      finally { this.saving = false }
    },
    openReset(u) { this.resetUser = u; this.resetPwd = ''; this.resetErr = ''; this.showReset = true },
    async doReset() {
      this.resetErr = ''
      if (!this.resetPwd || this.resetPwd.length < 6) { this.resetErr = '密码长度至少 6 位'; return }
      this.saving = true
      try {
        const r = await api.post('/users/' + this.resetUser.id + '/reset-password', { password: this.resetPwd })
        if (r.code === 0) { this.showReset = false; this.toast('密码已重置', 'success') }
        else this.resetErr = r.message
      } catch (e) { this.resetErr = '重置失败，请重试' }
      finally { this.saving = false }
    },
    async del(u) {
      if (!confirm('确认删除用户 ' + u.username + '？')) return
      const r = await api.del('/users/' + u.id)
      if (r.code === 0) { this.toast('已删除', 'success'); this.load() } else this.toast(r.message, 'error')
    }
  }
}
</script>
