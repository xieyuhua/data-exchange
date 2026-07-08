<template>
  <div class="panel">
    <div class="panel-head"><h2>FTP/SFTP 账号</h2><button class="btn btn-primary" @click="openForm(null)">+ 新增账号</button></div>
    <div class="panel-body p0">
      <table v-if="list.length"><thead><tr><th>ID</th><th>名称</th><th>厂家</th><th>协议</th><th>主机</th><th>端口</th><th>路径</th><th>状态</th><th>操作</th></tr></thead><tbody>
        <tr v-for="a in list" :key="a.id">
          <td>{{ a.id }}</td><td>{{ a.name }}</td><td class="muted">{{ a.vendor_name }}</td>
          <td><span class="badge" :class="'badge-'+a.protocol">{{ a.protocol }}</span></td>
          <td class="cell-mono">{{ a.host }}</td><td>{{ a.port }}</td><td class="cell-mono">{{ a.remote_path }}</td>
          <td><span class="badge" :class="a.enabled?'badge-on':'badge-off'">{{ a.enabled?'启用':'停用' }}</span></td>
          <td>
            <button class="btn btn-ghost btn-sm" @click="openForm(a)">编辑</button>
            <button class="btn btn-danger btn-sm" @click="del(a.id)">删除</button>
          </td>
        </tr>
      </tbody></table>
      <div v-else class="empty">暂无FTP/SFTP账号</div>
    </div>

    <div v-if="showModal" class="modal-mask" @click.self="showModal=false">
      <div class="modal">
        <div class="modal-head"><h3>{{ editing?'编辑账号':'新增账号' }}</h3><button class="modal-close" @click="showModal=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-grid">
            <div class="form-row"><label>名称 *</label><input v-model="form.name"></div>
            <div class="form-row"><label>厂家</label><select v-model.number="form.vendor_id"><option v-for="v in vendors" :key="v.id" :value="v.id">{{ v.name }}</option></select></div>
            <div class="form-row"><label>协议</label><select v-model="form.protocol"><option value="sftp">sftp</option><option value="ftp">ftp</option></select></div>
            <div class="form-row"><label>端口</label><input v-model.number="form.port" type="number"></div>
            <div class="form-row"><label>主机</label><input v-model="form.host"></div>
            <div class="form-row"><label>远程路径</label><input v-model="form.remote_path" placeholder="/"></div>
            <div class="form-row"><label>用户名</label><input v-model="form.username"></div>
            <div class="form-row"><label>密码</label><input v-model="form.password" type="password"></div>
            <div class="form-row full"><label>状态</label><select v-model.number="form.enabled"><option :value="1">启用</option><option :value="0">停用</option></select></div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn" :disabled="testing" @click="testConn">{{ testing?'测试中...':'测试连接' }}</button>
          <button class="btn" @click="showModal=false">取消</button>
          <button class="btn btn-primary" @click="save">保存</button></div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
export default {
  data() { return { list:[], vendors:[], showModal:false, editing:false, testing:false, form:{ id:0,vendor_id:0,name:'',protocol:'sftp',host:'',port:22,username:'',password:'',remote_path:'/',enabled:1 } } },
  inject: ['toast'],
  async mounted() { const [ar,vr]=await Promise.all([api.get('/ftp-accounts'),api.get('/vendors')]); this.list=ar.data; this.vendors=vr.data },
  methods: {
    openForm(a) {
      this.editing=!!a
      this.form=a?{...a}:{id:0,vendor_id:this.vendors[0]?.id||0,name:'',protocol:'sftp',host:'',port:22,username:'',password:'',remote_path:'/',enabled:1}
      this.showModal=true
    },
    async save() { if(!this.form.name) return this.toast('名称不能为空','error'); const r=await api.post('/ftp-accounts',this.form); if(r.code===0){this.showModal=false;this.toast('已保存','success');this.load()}else this.toast(r.message,'error') },
    async testConn() { this.testing=true; try{const r=await api.post('/ftp-accounts/test',this.form); if(r.code===0)this.toast('FTP/SFTP连接成功','success'); else this.toast(r.message,'error') }catch(e){this.toast('测试请求失败','error')}finally{this.testing=false} },
    async del(id) { if(!confirm('确认删除？')) return; const r=await api.del('/ftp-accounts/'+id); if(r.code===0){this.toast('已删除','success');this.load()}else this.toast(r.message,'error') },
    async load() { const r=await api.get('/ftp-accounts'); this.list=r.data }
  }
}
</script>
