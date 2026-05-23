<template>
  <div class="rules-page">
    <div class="page-toolbar">
      <div class="heading-group">
        <div class="heading-icon indigo"><el-icon :size="18"><SetUp /></el-icon></div>
        <div>
          <div class="page-heading">规则引擎</div>
          <div class="page-sub">{{ rules.length }} 条检测规则已加载</div>
        </div>
      </div>
      <button class="btn-primary" @click="openCreate">
        <el-icon :size="14"><Plus /></el-icon> 新建规则
      </button>
    </div>

    <div class="table-card">
      <table class="data-table">
        <thead>
          <tr>
            <th>规则 ID</th>
            <th>规则名称</th>
            <th>描述</th>
            <th>等级</th>
            <th>状态</th>
            <th>匹配位置</th>
            <th style="width:120px"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rules" :key="r.id">
            <td class="mono id-cell">{{ r.id }}</td>
            <td class="name-cell">{{ r.name }}</td>
            <td class="desc-cell">{{ r.description || '-' }}</td>
            <td><span class="severity-pill" :class="r.severity">{{ sevTxt(r.severity) }}</span></td>
            <td>
              <span class="status-badge" :class="r.enabled ? 'on' : 'off'">
                {{ r.enabled ? '启用' : '禁用' }}
              </span>
            </td>
            <td>
              <span class="loc-tag" v-for="l in r.match_locations" :key="l">{{ locLabel(l) }}</span>
            </td>
            <td class="action-cell">
              <button class="action-btn edit" @click="openEdit(r)">编辑</button>
              <button class="action-btn delete" @click="del(r.id)">删除</button>
            </td>
          </tr>
          <tr v-if="rules.length === 0 && !loading">
            <td colspan="7" class="empty-state">
              <div class="empty-icon"><el-icon :size="32"><SetUp /></el-icon></div>
              <div class="empty-text">暂无自定义规则</div>
              <div class="empty-desc">系统已加载内置规则，可在此添加自定义检测规则</div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 弹窗 -->
    <div class="modal-overlay" v-if="showDlg" @click.self="showDlg = false">
      <div class="modal-card">
        <div class="modal-header">
          <div class="modal-title">{{ isEdit ? '编辑规则' : '新建规则' }}</div>
          <button class="modal-close" @click="showDlg = false"><el-icon :size="18"><Close /></el-icon></button>
        </div>
        <div class="modal-body">
          <div class="form-grid">
            <div class="form-row" v-if="isEdit">
              <label>规则 ID</label>
              <input v-model="form.id" disabled class="form-input" />
            </div>
            <div class="form-row">
              <label>规则名称</label>
              <input v-model="form.name" class="form-input" placeholder="自定义 SQL 注入检测" />
            </div>
            <div class="form-row">
              <label>描述</label>
              <textarea v-model="form.description" class="form-textarea" rows="2" placeholder="检测描述..."></textarea>
            </div>
            <div class="form-row">
              <label>威胁等级</label>
              <select v-model="form.severity" class="form-select">
                <option value="critical">严重</option>
                <option value="high">高危</option>
                <option value="medium">中危</option>
                <option value="low">低危</option>
              </select>
            </div>
            <div class="form-row">
              <label>匹配位置</label>
              <div class="checkbox-group">
                <label class="checkbox-item" v-for="loc in locOptions" :key="loc.value">
                  <input type="checkbox" :value="loc.value" v-model="form.match_locations" />
                  <span>{{ loc.label }}</span>
                </label>
              </div>
            </div>
            <div class="form-row">
              <label>正则模式</label>
              <textarea v-model="patStr" class="form-textarea code" rows="5" placeholder="每行一个 Go regexp 正则&#10;(?i)(UNION\s+SELECT)&#10;(?i)(SLEEP\s*\()"></textarea>
              <div class="form-hint">每行一个 Go 正则表达式，(?i) 表示不区分大小写</div>
            </div>
            <div class="form-row">
              <label>启用</label>
              <label class="toggle-switch">
                <input type="checkbox" v-model="form.enabled" />
                <span class="toggle-track"></span>
              </label>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="showDlg = false">取消</button>
          <button class="btn-primary" @click="save" :disabled="saving">{{ saving ? '保存中...' : '保存规则' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Plus, SetUp, Close } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const rules = ref([]), loading = ref(false), saving = ref(false)
const showDlg = ref(false), isEdit = ref(false), form = ref({}), patStr = ref('')

const locOptions = [
  { value: 'url', label: 'URL' },
  { value: 'path', label: '路径' },
  { value: 'query', label: '参数' },
  { value: 'body', label: 'Body' },
  { value: 'headers', label: 'Headers' },
]

function sevTxt(s) { return { critical:'严重', high:'高危', medium:'中危', low:'低危' }[s] || s }
function locLabel(l) { return { url:'URL', path:'路径', query:'参数', body:'Body', headers:'Headers', user_agent:'UA' }[l] || l }

async function load() { loading.value = true; try { rules.value = await api.get('/rules') || [] } catch {} finally { loading.value = false } }
function openCreate() { isEdit.value = false; form.value = { id:'', name:'', description:'', severity:'high', enabled:true, match_locations:['url','query','body'] }; patStr.value = ''; showDlg.value = true }
function openEdit(r) { isEdit.value = true; form.value = { ...r }; patStr.value = (r.patterns||[]).join('\n'); showDlg.value = true }

async function save() {
  if (!form.value.name?.trim()) {
    ElMessage.warning('请输入规则名称'); return
  }
  if (!form.value.match_locations?.length) {
    ElMessage.warning('请选择至少一个匹配位置'); return
  }
  const patterns = patStr.value.split('\n').map(s=>s.trim()).filter(Boolean)
  if (!patterns.length) {
    ElMessage.warning('请填写至少一个正则模式'); return
  }
  saving.value = true
  try {
    const d = { ...form.value, name: form.value.name.trim(), patterns }
    isEdit.value ? await api.put(`/rules/${form.value.id}`, d) : await api.post('/rules', d)
    ElMessage.success('规则已保存'); showDlg.value = false; load()
  } catch {} finally { saving.value = false }
}

async function del(id) {
  try {
    await ElMessageBox.confirm('确定删除此规则？删除后不可恢复。', '确认删除', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
    await api.delete(`/rules/${id}`); ElMessage.success('已删除'); load()
  } catch {}
}

onMounted(load)
</script>

<style scoped>
.rules-page { }
.page-toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }

.id-cell { color: var(--primary); font-weight: 600; word-break: break-all; }
.name-cell { font-weight: 600; color: var(--text-primary); }
.desc-cell { color: var(--text-secondary); max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.loc-tag {
  display: inline-block; padding: 2px 7px; border-radius: 4px;
  font-size: 11px; font-weight: 500; background: var(--border-light); color: var(--text-secondary);
  margin: 1px 2px;
}

.action-cell { display: flex; gap: 6px; }
.action-btn {
  padding: 4px 10px; border-radius: 6px; border: none;
  font-size: 12px; font-weight: 600; cursor: pointer; transition: all 0.2s;
}
.action-btn.edit { background: var(--primary-light); color: var(--primary); }
.action-btn.edit:hover { background: var(--primary); color: #fff; }
.action-btn.delete { background: #fff1f2; color: var(--danger); }
.action-btn.delete:hover { background: var(--danger); color: #fff; }

.form-grid { display: flex; flex-direction: column; gap: 16px; }
.form-row { display: flex; flex-direction: column; gap: 5px; }
.form-row label:first-child { font-size: 12.5px; font-weight: 600; color: var(--text-secondary); }
.form-input:disabled { background: var(--bg-hover); color: var(--text-muted); }
.form-textarea.code { font-family: var(--font-mono); font-size: 12.5px; }
.form-hint { font-size: 11px; color: var(--text-muted); margin-top: 2px; }

.checkbox-group { display: flex; gap: 12px; flex-wrap: wrap; }
.checkbox-item {
  display: flex; align-items: center; gap: 5px; cursor: pointer;
  font-size: 13px; color: var(--text-secondary);
}
.checkbox-item input { accent-color: var(--primary); }

.empty-state { text-align: center; padding: 48px 16px !important; }
.empty-icon { color: #cbd5e1; margin-bottom: 12px; }
.empty-text { font-size: 14px; color: var(--text-secondary); font-weight: 600; }
.empty-desc { font-size: 12px; color: var(--text-muted); margin-top: 4px; }

@media (max-width: 768px) {
  .rules-page { max-width: 100%; overflow-x: hidden; }
  .page-toolbar { align-items: stretch; flex-direction: column; gap: 12px; }
  .btn-primary { width: 100%; justify-content: center; }
  .table-card { overflow: visible; background: transparent; border: 0; }
  .data-table, .data-table tbody, .data-table tr, .data-table td { display: block; width: 100%; }
  .data-table thead { display: none; }
  .data-table tr {
    background: #fff;
    border: 1px solid var(--border);
    border-radius: var(--radius-card);
    margin-bottom: 12px;
    overflow: hidden;
    box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  }
  .data-table td {
    display: grid;
    grid-template-columns: 82px minmax(0, 1fr);
    gap: 10px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border-light);
    min-width: 0;
    word-break: break-word;
  }
  .data-table td::before {
    color: var(--text-muted);
    font-size: 12px;
    font-weight: 700;
  }
  .data-table td:nth-child(1)::before { content: "规则 ID"; }
  .data-table td:nth-child(2)::before { content: "规则名称"; }
  .data-table td:nth-child(3)::before { content: "描述"; }
  .data-table td:nth-child(4)::before { content: "等级"; }
  .data-table td:nth-child(5)::before { content: "状态"; }
  .data-table td:nth-child(6)::before { content: "匹配位置"; }
  .data-table td:nth-child(7)::before { content: "操作"; }
  .desc-cell { max-width: none; white-space: normal; }
  .action-cell { display: flex !important; grid-template-columns: none !important; gap: 8px; }
  .action-cell::before { flex: 0 0 72px; }
  .action-btn { flex: 1; padding: 8px 10px; }
  .empty-state { display: block !important; }
  .empty-state::before { content: none !important; }
  .modal-overlay { align-items: flex-end; padding: 0; }
  .modal-card { width: 100vw; max-width: 100vw; max-height: 92vh; border-radius: 16px 16px 0 0; }
  .modal-header, .modal-body, .modal-footer { padding-left: 16px; padding-right: 16px; }
  .modal-footer { flex-direction: column-reverse; }
  .modal-footer .btn-primary, .modal-footer .btn-ghost { width: 100%; justify-content: center; }
  .checkbox-group { gap: 8px; }
  .checkbox-item { min-width: calc(50% - 4px); }
}
</style>
