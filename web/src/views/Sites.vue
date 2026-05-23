<template>
  <div class="sites-page">
    <div class="page-toolbar">
      <div class="heading-group">
        <div class="heading-icon green"><el-icon :size="18"><Connection /></el-icon></div>
        <div>
          <div class="page-heading">站点管理</div>
          <div class="page-sub">专业版多站代理，按域名匹配不同回源和防护策略</div>
        </div>
      </div>
      <button class="btn-primary" :disabled="!isPro" @click="openCreate">新增站点</button>
    </div>

    <div class="upgrade-panel" v-if="!isPro">
      <div class="upgrade-copy">
        <span class="upgrade-kicker">专业版能力</span>
        <h3>多站点统一接入与独立防护</h3>
        <p>适合一台服务器承载官网、后台、API、上传服务等多个业务入口，为不同域名配置独立回源、AI 检测和挑战策略。</p>
        <div class="upgrade-features">
          <span>多域名回源</span>
          <span>泛域名匹配</span>
          <span>站点级策略</span>
          <span>按站点审计</span>
        </div>
      </div>
      <div class="upgrade-actions">
        <button class="btn-primary" @click="goSettings">前往授权</button>
        <span>激活后即可配置多站代理</span>
      </div>
    </div>

    <template v-else>
      <div class="stats-grid">
        <div class="stat-card"><span>站点总数</span><strong>{{ sites.length }}</strong></div>
        <div class="stat-card"><span>启用站点</span><strong>{{ enabledCount }}</strong></div>
        <div class="stat-card"><span>AI 防护</span><strong>{{ aiCount }}</strong></div>
        <div class="stat-card"><span>挑战页</span><strong>{{ challengeCount }}</strong></div>
      </div>

      <div class="ops-grid">
        <div class="ops-card">
          <span>匹配逻辑</span>
          <strong>Host 精确匹配优先，支持 *.example.com 泛域名</strong>
        </div>
        <div class="ops-card">
          <span>默认回源</span>
          <strong>未命中站点时回退到系统代理默认后端</strong>
        </div>
        <div class="ops-card">
          <span>站点策略</span>
          <strong>每个站点可独立控制 AI 检测、挑战页和业务上下文</strong>
        </div>
      </div>

      <div class="table-card">
        <div class="table-head">
          <div>
            <div class="table-title">站点列表</div>
            <div class="table-sub">请求进入 WAF 后会按 Host 匹配站点，未匹配时回退到默认回源</div>
          </div>
          <div class="table-tools">
            <input v-model.trim="keyword" placeholder="搜索站点 / 域名 / 回源" />
            <select v-model="typeFilter">
              <option value="">全部类型</option>
              <option value="website">普通网站</option>
              <option value="admin">后台管理</option>
              <option value="api">API 接口</option>
              <option value="upload">上传服务</option>
              <option value="payment">支付订单</option>
            </select>
          </div>
        </div>
        <table class="data-table">
          <thead>
            <tr>
              <th>站点</th>
              <th>域名</th>
              <th>回源地址</th>
              <th>类型</th>
              <th>策略</th>
              <th>状态</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="site in filteredSites" :key="site.id">
              <td><strong>{{ site.name }}</strong></td>
              <td><div class="domains"><span v-for="d in site.domains" :key="d">{{ d }}</span></div></td>
              <td class="mono">{{ site.upstream }}</td>
              <td>{{ typeText(site.site_type) }}</td>
              <td>
                <div class="policy-tags">
                  <span v-if="site.ai_enabled">AI</span>
                  <span v-if="site.challenge_enabled">挑战页</span>
                  <span v-if="!site.ai_enabled && !site.challenge_enabled">基础规则</span>
                </div>
              </td>
              <td><span class="status-pill" :class="{ on: site.enabled }">{{ site.enabled ? '启用' : '停用' }}</span></td>
              <td class="actions">
                <button class="btn-ghost" @click="toggleSite(site)">{{ site.enabled ? '停用' : '启用' }}</button>
                <button class="btn-ghost" @click="openEdit(site)">编辑</button>
                <button class="btn-danger" @click="removeSite(site)">删除</button>
              </td>
            </tr>
            <tr v-if="!filteredSites.length && !loading">
              <td colspan="7" class="empty">{{ isPro ? '暂无匹配站点' : '当前版本未启用站点数据' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <div class="modal-overlay" v-if="showDlg" @click.self="showDlg = false">
      <div class="modal-card">
        <div class="modal-head">
          <div>
            <div class="modal-title">{{ isEdit ? '编辑站点' : '新增站点' }}</div>
            <div class="modal-sub">域名一行一个，回源填写内网服务地址，例如 127.0.0.1:3000</div>
          </div>
          <button class="close-btn" @click="showDlg = false">×</button>
        </div>
        <div class="modal-body">
          <div class="form-grid">
            <label>站点名称<input v-model.trim="form.name" placeholder="官网 / 管理后台 / API 服务" /></label>
            <label>业务类型
              <select v-model="form.site_type">
                <option value="website">普通网站</option>
                <option value="admin">后台管理</option>
                <option value="api">API 接口</option>
                <option value="upload">上传服务</option>
                <option value="payment">支付订单</option>
              </select>
            </label>
          </div>
          <label>域名<textarea v-model="domainsText" rows="3" placeholder="www.example.com&#10;api.example.com"></textarea></label>
          <label>回源地址<input v-model.trim="form.upstream" placeholder="127.0.0.1:3000" /></label>
          <div class="switch-row">
            <label><input type="checkbox" v-model="form.enabled" /> 启用站点</label>
            <label><input type="checkbox" v-model="form.ai_enabled" /> AI 检测</label>
            <label><input type="checkbox" v-model="form.challenge_enabled" /> 挑战页</label>
          </div>
          <div class="modal-actions">
            <button class="btn-ghost" @click="showDlg = false">取消</button>
            <button class="btn-primary" :disabled="saving" @click="saveSite">{{ saving ? '保存中' : '保存站点' }}</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Connection } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const sites = ref([])
const router = useRouter()
const isPro = inject('isPro', ref(false))
const loading = ref(false)
const saving = ref(false)
const showDlg = ref(false)
const isEdit = ref(false)
const domainsText = ref('')
const keyword = ref('')
const typeFilter = ref('')
const form = reactive(defaultForm())

const enabledCount = computed(() => sites.value.filter(s => s.enabled).length)
const aiCount = computed(() => sites.value.filter(s => s.ai_enabled).length)
const challengeCount = computed(() => sites.value.filter(s => s.challenge_enabled).length)
const filteredSites = computed(() => {
  const kw = keyword.value.toLowerCase()
  return sites.value.filter((site) => {
    const matchesType = !typeFilter.value || site.site_type === typeFilter.value
    const haystack = [site.name, site.upstream, typeText(site.site_type), ...(site.domains || [])].join(' ').toLowerCase()
    return matchesType && (!kw || haystack.includes(kw))
  })
})

function defaultForm() {
  return { id: '', name: '', domains: [], upstream: '', enabled: true, ai_enabled: true, challenge_enabled: true, site_type: 'website' }
}

function typeText(t) {
  return { website: '普通网站', admin: '后台管理', api: 'API 接口', upload: '上传服务', payment: '支付订单' }[t] || t
}

async function loadSites() {
  loading.value = true
  try { sites.value = await api.get('/sites') || [] } catch {} finally { loading.value = false }
}

function openCreate() {
  if (!isPro.value) return
  Object.assign(form, defaultForm())
  domainsText.value = ''
  isEdit.value = false
  showDlg.value = true
}

function goSettings() {
  router.push('/settings')
}

function openEdit(site) {
  if (!isPro.value) return
  Object.assign(form, JSON.parse(JSON.stringify(site)))
  domainsText.value = (site.domains || []).join('\n')
  isEdit.value = true
  showDlg.value = true
}

async function saveSite() {
  if (!isPro.value) return
  form.domains = domainsText.value.split(/\n|,/).map(s => s.trim()).filter(Boolean)
  if (!form.name || !form.upstream || !form.domains.length) {
    ElMessage.warning('请填写站点名称、域名和回源地址')
    return
  }
  saving.value = true
  try {
    if (isEdit.value) await api.put(`/sites/${form.id}`, form)
    else await api.post('/sites', form)
    ElMessage.success('站点已保存，代理规则已刷新')
    showDlg.value = false
    loadSites()
  } finally { saving.value = false }
}

async function toggleSite(site) {
  try {
    await api.put(`/sites/${site.id}`, { ...site, enabled: !site.enabled })
    ElMessage.success(site.enabled ? '站点已停用' : '站点已启用')
    loadSites()
  } catch {}
}

async function removeSite(site) {
  try {
    await ElMessageBox.confirm(`确认删除站点「${site.name}」？`, '删除站点', { type: 'warning' })
    await api.delete(`/sites/${site.id}`)
    ElMessage.success('站点已删除')
    loadSites()
  } catch {}
}

watch(isPro, (value) => { if (value && !sites.value.length) loadSites() })
onMounted(() => { if (isPro.value) loadSites() })
</script>

<style scoped>
.page-toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 16px; }
.stat-card { background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card); padding: 16px; }
.stat-card span { color: var(--text-secondary); font-size: 12px; font-weight: 700; }
.stat-card strong { display: block; margin-top: 8px; font-size: 24px; color: var(--text-primary); }
.ops-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 16px; }
.ops-card { background: var(--bg-hover); border: 1px solid var(--border); border-radius: var(--radius-card); padding: 14px 16px; }
.ops-card span { display: block; color: var(--text-secondary); font-size: 12px; font-weight: 800; margin-bottom: 6px; }
.ops-card strong { color: var(--text-primary); font-size: 13px; line-height: 1.55; }
.table-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: var(--card-pad); border-bottom: 1px solid var(--border-light); }
.table-title { font-size: 15px; font-weight: 800; color: var(--text-primary); }
.table-sub { font-size: 12px; color: var(--text-muted); margin-top: 3px; }
.table-tools { display: flex; align-items: center; gap: 10px; }
.table-tools input, .table-tools select { height: 36px; min-width: 180px; }
.domains, .policy-tags { display: flex; gap: 6px; flex-wrap: wrap; }
.domains span, .policy-tags span { padding: 4px 8px; border-radius: 999px; background: var(--primary-light); color: var(--primary-text); font-size: 12px; font-weight: 700; }
.policy-tags span { background: #ecfdf5; color: #15803d; }
.status-pill { padding: 4px 9px; border-radius: 999px; background: #f1f5f9; color: var(--text-secondary); font-size: 12px; font-weight: 800; }
.status-pill.on { background: #ecfdf5; color: #15803d; }
.actions { display: flex; gap: 8px; }
.upgrade-panel {
  display: flex; align-items: center; justify-content: space-between; gap: 24px;
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  padding: 28px; box-shadow: 0 1px 3px rgba(0,0,0,.04);
}
.upgrade-kicker {
  display: inline-flex; margin-bottom: 10px; padding: 5px 10px; border-radius: 999px;
  background: #ecfdf5; color: #15803d; font-size: 12px; font-weight: 800;
}
.upgrade-copy h3 { margin: 0 0 10px; color: var(--text-primary); font-size: 22px; font-weight: 850; }
.upgrade-copy p { max-width: 680px; color: var(--text-secondary); font-size: 14px; line-height: 1.75; }
.upgrade-features { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 18px; }
.upgrade-features span {
  padding: 6px 12px; border-radius: 999px; background: var(--bg-hover); color: var(--text-secondary);
  border: 1px solid var(--border); font-size: 12px; font-weight: 700;
}
.upgrade-actions { display: flex; flex-direction: column; align-items: flex-end; gap: 10px; flex-shrink: 0; }
.upgrade-actions .btn-primary { padding: 10px 20px; }
.upgrade-actions span { color: var(--text-muted); font-size: 12px; }
.empty { text-align: center; color: var(--text-muted); padding: 36px !important; }
.switch-row { display: flex; gap: 16px; flex-wrap: wrap; margin: 6px 0 16px; }
.switch-row label { flex-direction: row; align-items: center; margin: 0; }
label { display: flex; flex-direction: column; gap: 6px; color: var(--text-secondary); font-size: 12px; font-weight: 700; margin-bottom: 12px; }
input, textarea, select { border: 1px solid var(--border); border-radius: var(--radius-input); padding: 9px 11px; font: inherit; color: var(--text-primary); outline: none; }
input:focus, textarea:focus, select:focus { border-color: var(--primary); box-shadow: 0 0 0 3px rgba(99,102,241,.08); }
.modal-sub { font-size: 12px; color: var(--text-muted); margin-top: 3px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; }

@media (max-width: 768px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .ops-grid { grid-template-columns: 1fr; }
  .table-head { align-items: stretch; flex-direction: column; }
  .table-tools { width: 100%; flex-direction: column; }
  .table-tools input, .table-tools select { flex: 1; min-width: 0; width: 100%; }
  .table-card { overflow-x: auto; }
  .data-table { min-width: 860px; }
  .stats-grid, .form-grid { grid-template-columns: 1fr; }
  .page-toolbar { align-items: flex-start; flex-direction: column; }
  .upgrade-panel { align-items: stretch; flex-direction: column; padding: 22px; }
  .upgrade-actions { align-items: stretch; }
}
</style>
