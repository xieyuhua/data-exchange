<template>
  <div class="panel">
    <div class="panel-head"><h2>数据库连接</h2><button class="btn btn-primary" @click="openForm(null)">+ 新增连接</button></div>
    <div class="panel-body p0">
      <div class="table-scroll" v-if="list.length">
      <table><thead><tr><th>ID</th><th>名称</th><th>类型</th><th>主机</th><th>端口</th><th>数据库</th><th>状态</th><th>操作</th></tr></thead><tbody>
        <tr v-for="c in list" :key="c.id">
          <td>{{ c.id }}</td><td>{{ c.name }}</td><td><span class="badge" :class="'badge-'+c.db_type">{{ c.db_type }}</span></td>
          <td class="cell-mono">{{ c.host }}</td><td>{{ c.port }}</td><td class="cell-mono">{{ c.database_name }}</td>
          <td><span class="badge" :class="c.enabled?'badge-on':'badge-off'">{{ c.enabled?'启用':'停用' }}</span></td>
          <td>
            <button class="btn btn-ghost btn-sm" @click="testConn(c)">测试</button>
            <button class="btn btn-ghost btn-sm" @click="openForm(c)">编辑</button>
            <button class="btn btn-danger btn-sm" @click="del(c.id)">删除</button>
          </td>
        </tr>
      </tbody></table>
      </div>
      <div v-else class="empty">暂无数据库连接</div>
    </div>

    <div v-if="showModal" class="modal-mask" @click.self="showModal=false">
      <div class="modal">
        <div class="modal-head"><h3>{{ editing?'编辑连接':'新增连接' }}</h3><button class="modal-close" @click="showModal=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-grid">
            <div class="form-row"><label>名称 *</label><input v-model="form.name"></div>
            <div class="form-row"><label>类型</label><select v-model="form.db_type"><option v-for="t in types" :key="t" :value="t">{{ t }}</option></select></div>
            <div class="form-row"><label>主机</label><input v-model="form.host"></div>
            <div class="form-row"><label>端口</label><input v-model.number="form.port" type="number"></div>
            <div class="form-row"><label>用户名</label><input v-model="form.username"></div>
            <div class="form-row"><label>密码</label><input v-model="form.password" type="password"></div>
            <div class="form-row"><label>数据库名</label><input v-model="form.database_name"></div>
            <div class="form-row"><label>额外参数</label><input v-model="form.extra_params" placeholder="如 charset=utf8"></div>
            <div class="form-row full"><label>状态</label><select v-model.number="form.enabled"><option :value="1">启用</option><option :value="0">停用</option></select></div>
          </div>
        </div>
        <div class="modal-foot"><button class="btn" @click="testConn(form)">测试连接</button><button class="btn" @click="showModal=false">取消</button><button class="btn btn-primary" @click="save">保存</button></div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
export default {
  data() { return { list:[], types:['mysql','oracle','postgresql','mssql'], showModal:false, editing:false, form:{ id:0,name:'',db_type:'mysql',host:'',port:3306,username:'',password:'',database_name:'',extra_params:'',enabled:1 } } },
  inject: ['toast'],
  async mounted() { await this.load() },
  methods: {
    async load() { const r = await api.get('/db-connections'); this.list = r.data },
    openForm(c) { this.editing=!!c; this.form=c?{...c}:{id:0,name:'',db_type:'mysql',host:'',port:3306,username:'',password:'',database_name:'',extra_params:'',enabled:1}; this.showModal=true },
    async save() { if(!this.form.name) return this.toast('名称不能为空','error'); const r=await api.post('/db-connections',this.form); if(r.code===0){this.showModal=false;this.toast('已保存','success');this.load()}else this.toast(r.message,'error') },
    async testConn(c) { const r=await api.post('/db-connections/test',c); if(r.code===0)this.toast(r.data||'连接成功','success'); else this.toast(r.message,'error') },
    async del(id) { if(!confirm('确认删除？')) return; const r=await api.del('/db-connections/'+id); if(r.code===0){this.toast('已删除','success');this.load()}else this.toast(r.message,'error') }
  }
}
</script>
