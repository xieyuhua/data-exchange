<template>
  <div>
    <div class="toolbar">
      <label class="muted">厂家：</label>
      <select v-model="vendorId" @change="loadTasks" style="width:auto;min-width:220px">
        <option v-for="v in vendors" :key="v.id" :value="v.id">{{ v.name }} ({{ v.code }})</option>
      </select>
      <div class="spacer"></div>
      <button class="btn btn-primary" @click="openForm(null)">+ 新增任务</button>
    </div>
    <div v-if="tasks.length" class="panel"><div class="panel-head"><h2>任务列表（{{ tasks.length }}/{{ maxTasks }}）</h2></div><div class="panel-body">
      <div v-for="t in tasks" :key="t.id" class="task-card">
        <div class="task-top">
          <span class="task-name">{{ t.task_name }}</span>
          <span class="badge" :class="t.enabled?'badge-on':'badge-off'">{{ t.enabled?'启用':'停用' }}</span>
          <span class="muted f12">#{{ t.id }}</span>
          <div class="spacer"></div>
          <button class="btn btn-ghost btn-sm" @click="execTask(t.id)">立即执行</button>
          <button class="btn btn-ghost btn-sm" @click="toggleTask(t.id)">{{ t.enabled?'停用':'启用' }}</button>
          <button class="btn btn-ghost btn-sm" @click="openForm(t)">编辑</button>
          <button class="btn btn-danger btn-sm" @click="delTask(t.id)">删除</button>
        </div>
        <div class="task-meta">
          模式：{{ t.execution_mode==='upload'?'导出并上传':'仅导出' }} | 排序：{{ t.sort_order }} | Cron：{{ t.cron_expression }}<br>
          数据库连接：{{ t.db_connection_name||'未设置' }} | FTP账号：{{ t.ftp_account_name||'—' }}<br>
          文件名模板：{{ t.csv_filename_template }}<br>
          <span v-if="t.last_run_at">上次执行：{{ t.last_run_at }} 状态：<span class="badge" :class="t.last_status==='success'?'badge-success':'badge-failed'">{{ t.last_status }}</span></span>
          <span v-else>尚未执行</span>
        </div>
      </div>
    </div></div>
    <div v-else class="panel"><div class="panel-body"><div class="empty">该厂家暂无任务（最多 {{ maxTasks }} 个）</div></div></div>

    <!-- 模态框 -->
    <div v-if="showModal" class="modal-mask" @click.self="showModal=false">
      <div class="modal">
        <div class="modal-head"><h3>{{ editing?'编辑任务':'新增任务' }}</h3><button class="modal-close" @click="showModal=false">&times;</button></div>
        <div class="modal-body">
          <div class="form-grid">
            <div class="form-row"><label>任务名称 *</label><input v-model="form.task_name"></div>
            <div class="form-row"><label>执行模式</label><select v-model="form.execution_mode"><option value="export_only">仅导出CSV</option><option value="upload">导出并上传</option></select></div>
            <div class="form-row"><label>数据库连接</label><select v-model.number="form.db_connection_id"><option :value="null">（不使用）</option><option v-for="d in dbs" :key="d.id" :value="d.id">{{ d.name }}</option></select></div>
            <div class="form-row"><label>FTP/SFTP账号</label><select v-model.number="form.ftp_account_id"><option :value="null">（不上传）</option><option v-for="f in ftps" :key="f.id" :value="f.id">{{ f.name }}</option></select></div>
            <div class="form-row"><label>Cron表达式</label><input v-model="form.cron_expression" placeholder="0 2 * * *"></div>
            <div class="form-row"><label>排序</label><input v-model.number="form.sort_order" type="number"></div>
            <div class="form-row full"><label>CSV文件名模板</label><input v-model="form.csv_filename_template" placeholder="{vendor_code}_{task_name}_{date}.csv"><div class="hint">可用：{vendor_code} {task_name} {date} {datetime} {yyyy} {mm} {dd} {HH} {MM} {SS}</div></div>
            <div class="form-row full"><label>SQL内容 *</label><textarea v-model="form.sql_content" style="min-height:140px"></textarea><div class="hint">支持常量占位符，如 SELECT * FROM t WHERE d='{{ yesterday }}'</div></div>
            <div class="form-row full"><label>状态</label><select v-model.number="form.enabled"><option :value="1">启用</option><option :value="0">停用</option></select></div>
          </div>
        </div>
        <div class="modal-foot"><button class="btn" @click="showModal=false">取消</button><button class="btn btn-primary" @click="save">保存</button></div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'
export default {
  data() { return { vendors:[], tasks:[], dbs:[], ftps:[], vendorId:0, maxTasks:4, showModal:false, editing:false, form:{ id:0,vendor_id:0,task_name:'',execution_mode:'export_only',db_connection_id:null,ftp_account_id:null,cron_expression:'0 2 * * *',sort_order:0,csv_filename_template:'{vendor_code}_{task_name}_{date}.csv',sql_content:'',enabled:1 } } },
  inject: ['toast'],
  async mounted() {
    const [vr,dr] = await Promise.all([api.get('/vendors'), api.get('/db-connections')])
    this.vendors = vr.data||[]; this.dbs = dr.data||[]
    this.vendorId = Number(this.$route.query.vendor) || (this.vendors[0]?.id||0)
    if (this.vendorId) await this.loadTasks()
  },
  methods: {
    async loadTasks() {
      if (!this.vendorId) return
      const [tr, fr] = await Promise.all([api.get('/vendors/'+this.vendorId+'/tasks'), api.get('/ftp-accounts?vendor_id='+this.vendorId)])
      this.tasks = tr.data||[]; this.ftps = fr.data||[]; this.maxTasks = tr.max||4
    },
    openForm(t) {
      this.editing = !!t
      this.form = t ? { ...t, vendor_id: this.vendorId } : { id:0,vendor_id:this.vendorId,task_name:'',execution_mode:'export_only',db_connection_id:null,ftp_account_id:null,cron_expression:'0 2 * * *',sort_order:0,csv_filename_template:'{vendor_code}_{task_name}_{date}.csv',sql_content:'',enabled:1 }
      this.showModal = true
    },
    async save() {
      if (!this.form.task_name) return this.toast('任务名称不能为空','error')
      const r = await api.post('/tasks', this.form)
      if (r.code===0) { this.showModal=false; this.toast('已保存','success'); this.loadTasks() }
      else this.toast(r.message,'error')
    },
    async delTask(id) { if(!confirm('确认删除？')) return; const r=await api.del('/tasks/'+id); if(r.code===0){this.toast('已删除','success');this.loadTasks()}else this.toast(r.message,'error') },
    async toggleTask(id) { const r=await api.post('/tasks/'+id+'/toggle',{}); if(r.code===0){this.toast('已切换','success');this.loadTasks()}else this.toast(r.message,'error') },
    async execTask(id) { const r=await api.post('/tasks/'+id+'/execute',{}); this.toast(r.message||'已提交', r.code===0?'success':'error') }
  }
}
</script>
