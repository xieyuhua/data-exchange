<template>
  <div class="panel">
    <div class="panel-head"><h2>厂家列表</h2>
      <div class="head-actions">
        <button class="btn btn-primary" @click="openForm(null)">+ 新增厂家</button>
      </div>
    </div>
    <div class="filter-bar">
      <div class="filter-item search-box">
        <svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="7"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <input class="filter-control" v-model="keyword" @keyup.enter="load(1)" placeholder="搜索名称 / 编码 / 描述…">
        <button v-if="keyword" class="search-clear" @click="clearSearch" title="清除">×</button>
      </div>
      <button class="btn" @click="load(1)">搜索</button>
      <div class="spacer"></div>
      <span class="muted">共 {{ total }} 个厂家</span>
    </div>
    <div class="panel-body p0">
      <div class="table-scroll" v-if="list.length">
      <table><thead><tr><th>ID</th><th>名称</th><th>编码</th><th>状态</th><th>描述</th><th>操作</th></tr></thead><tbody>
        <tr v-for="v in list" :key="v.id">
          <td>{{ v.id }}</td><td>{{ v.name }}</td><td class="cell-mono">{{ v.code }}</td>
          <td><span class="badge" :class="v.enabled?'badge-on':'badge-off'">{{ v.enabled?'启用':'停用' }}</span></td>
          <td class="muted">{{ v.description }}</td>
          <td>
            <button class="btn btn-ghost btn-sm" @click="$router.push('/tasks?vendor='+v.id)">查看任务</button>
            <button class="btn btn-ghost btn-sm" @click="openForm(v)">编辑</button>
            <button class="btn btn-danger btn-sm" @click="delVendor(v.id)">删除</button>
          </td>
        </tr>
      </tbody></table>
      </div>
      <div v-else class="empty">暂无厂家，点击右上角新增</div>
    </div>
    <div class="pager" v-if="total > pageSize">
      <button class="btn btn-sm" :disabled="page <= 1" @click="load(page - 1)">上一页</button>
      <span class="muted">第 {{ page }} / {{ Math.ceil(total / pageSize) }} 页（共 {{ total }} 条）</span>
      <button class="btn btn-sm" :disabled="page >= Math.ceil(total / pageSize)" @click="load(page + 1)">下一页</button>
    </div>

    <div v-if="showModal" class="modal-mask" @click.self="showModal=false">
      <div class="modal">
        <div class="modal-head"><h3>{{ editing?'编辑厂家':'新增厂家' }}</h3><button class="modal-close" @click="showModal=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-grid">
            <div class="form-row full"><label>名称 *</label><input v-model="form.name"></div>
            <div class="form-row full"><label>编码 *</label><input v-model="form.code" :disabled="editing"></div>
            <div class="form-row full"><label>描述</label><input v-model="form.description"></div>
            <div class="form-row full"><label>状态</label><select v-model="form.enabled"><option :value="1">启用</option><option :value="0">停用</option></select></div>
          </div>
        </div>
        <div class="modal-foot"><button class="btn" @click="showModal=false">取消</button><button class="btn btn-primary" @click="save">保存</button></div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
import { getPageSize } from '../configStore'
export default {
  data() { return { list:[], total:0, page:1, pageSize:getPageSize(), keyword:'', showModal:false, editing:false, form:{id:0,name:'',code:'',description:'',enabled:1} } },
  inject: ['toast'],
  async mounted() { await this.load(1) },
  methods: {
    async load(p) {
      this.page = p || this.page
      const r = await api.get('/vendors', { keyword: this.keyword, page: this.page, page_size: this.pageSize })
      this.list = r.data || []
      this.total = r.total || 0
    },
    openForm(v) {
      this.editing = !!v
      this.form = v ? { ...v } : { id:0,name:'',code:'',description:'',enabled:1 }
      this.showModal = true
    },
    clearSearch() {
      this.keyword = ''
      this.load(1)
    },
    async save() {
      if (!this.form.name||!this.form.code) return this.toast('名称和编码不能为空','error')
      const r = await api.post('/vendors', this.form)
      if (r.code===0) { this.showModal=false; this.toast('已保存','success'); this.load() }
      else this.toast(r.message,'error')
    },
    async delVendor(id) { if(!confirm('确认删除？关联任务和FTP也会删除。')) return; const r=await api.del('/vendors/'+id); if(r.code===0){this.toast('已删除','success');this.load()}else this.toast(r.message,'error') }
  }
}
</script>
