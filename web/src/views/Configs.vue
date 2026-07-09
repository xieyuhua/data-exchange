<template>
  <div class="panel">
    <div class="panel-head"><h2>系统配置</h2><button class="btn btn-primary" @click="openForm(null)">+ 新增配置</button></div>
    <div class="panel-body p0">
      <div class="table-scroll" v-if="list.length">
      <table><thead><tr><th>ID</th><th>键名</th><th>值</th><th>描述</th><th>操作</th></tr></thead><tbody>
        <tr v-for="c in list" :key="c.id">
          <td>{{ c.id }}</td>
          <td class="cell-mono">{{ c.config_key }}</td>
          <td class="cell-mono">{{ c.config_value }}</td>
          <td class="muted">{{ c.description }}</td>
          <td>
            <button class="btn btn-ghost btn-sm" @click="openForm(c)">编辑</button>
          </td>
        </tr>
      </tbody></table>
      </div>
      <div v-else class="empty">暂无配置</div>
    </div>

    <div v-if="showModal" class="modal-mask" @click.self="showModal=false">
      <div class="modal">
        <div class="modal-head"><h3>{{ editing?'编辑配置':'新增配置' }}</h3><button class="modal-close" @click="showModal=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-row full"><label>键名 *</label><input v-model="form.config_key" :disabled="editing" placeholder="如 max_parallel_tasks"></div>
          <div class="form-row full"><label>值</label><input v-model="form.config_value"></div>
          <div class="form-row full"><label>描述</label><input v-model="form.description"></div>
        </div>
        <div class="modal-foot"><button class="btn" @click="showModal=false">取消</button><button class="btn btn-primary" @click="save">保存</button></div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
import { refreshPageSize } from '../configStore'
export default {
  data() { return { list:[], showModal:false, editing:false, form:{id:0,config_key:'',config_value:'',description:''} } },
  inject: ['toast'],
  async mounted() { await this.load() },
  methods: {
    async load() { const r=await api.get('/configs'); this.list=r.data || [] },
    openForm(c) { this.editing=!!c; this.form=c?{...c}:{id:0,config_key:'',config_value:'',description:''}; this.showModal=true },
    async save() {
      if(!this.form.config_key) return this.toast('键名不能为空','error')
      const r=await api.post('/configs',{ id:this.form.id, config_key:this.form.config_key, config_value:this.form.config_value, description:this.form.description })
      if(r.code===0){ this.showModal=false; this.toast('已保存','success'); this.load(); refreshPageSize() } else this.toast(r.message,'error')
    }
  }
}
</script>
