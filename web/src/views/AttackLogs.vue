<template>
  <div class="logs-page">
    <!-- 页面标题 -->
    <div class="page-toolbar">
      <div class="heading-group">
        <div class="heading-icon rose"><el-icon :size="18"><Document /></el-icon></div>
        <div>
          <div class="page-heading">攻击日志</div>
          <div class="page-sub">威胁事件追踪与分析</div>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-group">
        <div class="filter-item">
          <label>站点范围</label>
          <select v-model="filters.site_id" class="filter-select site-select" @change="page = 1; loadLogs()">
            <option value="">全部站点</option>
            <option v-for="site in sites" :key="site.id" :value="site.id">
              {{ site.name }} · {{ site.domains?.[0] || site.upstream }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label>来源 IP</label>
          <input v-model="filters.client_ip" placeholder="输入 IP 地址" class="filter-input" @keyup.enter="loadLogs" />
        </div>
        <div class="filter-item">
          <label>威胁等级</label>
          <select v-model="filters.severity" class="filter-select" @change="loadLogs">
            <option value="">全部</option>
            <option value="critical">严重</option>
            <option value="high">高危</option>
            <option value="medium">中危</option>
            <option value="low">低危</option>
          </select>
        </div>
        <div class="filter-item">
          <label>检测引擎</label>
          <select v-model="filters.source" class="filter-select" @change="loadLogs">
            <option value="">全部</option>
            <option value="rule_engine">规则引擎</option>
            <option value="ai">AI 检测</option>
          </select>
        </div>
      </div>
      <div class="filter-actions">
        <button class="btn-primary" @click="loadLogs">
          <el-icon :size="14"><Search /></el-icon> 查询
        </button>
        <button class="btn-ghost" @click="resetFilters">重置</button>
      </div>
    </div>

    <!-- 数据表 -->
    <div class="table-card">
      <div class="table-header">
        <span class="table-title">攻击日志</span>
        <span class="table-count">{{ total }} 条记录</span>
      </div>
      <table class="data-table">
        <thead>
          <tr>
            <th>时间</th>
            <th>站点</th>
            <th>来源 IP</th>
            <th>地区</th>
            <th>方法</th>
            <th>攻击路径</th>
            <th>检测引擎</th>
            <th>
              <span class="th-help">
                检测结果
                <button class="help-btn" type="button" @click.stop="toggleHelp('result')" aria-label="检测结果说明">?</button>
              </span>
            </th>
            <th>危险度</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in logs" :key="log.id" @click="showDetail(log)" class="clickable-row">
            <td class="mono time-cell">{{ fmt(log.timestamp) }}</td>
            <td class="site-cell">{{ siteName(log) }}</td>
            <td class="mono">{{ log.client_ip }}</td>
            <td class="region-cell">{{ log.region || '-' }}</td>
            <td><span class="method-tag" :class="log.method?.toLowerCase()">{{ log.method }}</span></td>
            <td class="path-cell" :title="log.path">{{ log.path }}</td>
            <td>
              <span class="engine-badge" :class="log.source === 'ai' ? 'ai' : 'rule'">
                <el-icon v-if="log.source === 'ai'" :size="12"><Cpu /></el-icon>
                {{ log.source === 'ai' ? 'AI' : '规则' }}
              </span>
            </td>
            <td class="result-cell">{{ log.rule_name }}</td>
            <td><span class="severity-pill" :class="log.severity">{{ sevTxt(log.severity) }}</span></td>
            <td><span class="detail-link">详情</span></td>
          </tr>
          <tr v-if="logs.length === 0 && !loading">
            <td colspan="10" class="empty-state">
              <div class="empty-icon">
                <el-icon :size="32"><Document /></el-icon>
              </div>
              <div class="empty-text">暂无攻击记录</div>
              <div class="empty-desc">系统未检测到任何攻击行为</div>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="help-popover" v-if="helpKey === 'result'">
        <button class="help-close" @click="helpKey = ''">×</button>
        <strong>检测结果怎么看？</strong>
        <p>这里显示系统判断请求异常的主要原因。可以理解为“为什么这次访问被认为有风险”，例如尝试注入数据库、访问敏感路径、提交异常脚本或出现自动化扫描特征。</p>
      </div>

      <div class="pagination-bar" v-if="total > 0">
        <span class="page-info">共 {{ total }} 条 / {{ totalPages }} 页</span>
        <div class="page-controls">
          <select v-model="pageSize" class="page-size-select" @change="page = 1; loadLogs()">
            <option :value="10">10 条/页</option>
            <option :value="20">20 条/页</option>
            <option :value="50">50 条/页</option>
          </select>
          <button class="page-btn" :disabled="page <= 1" @click="page = 1; loadLogs()">首页</button>
          <button class="page-btn" :disabled="page <= 1" @click="page--; loadLogs()">上一页</button>
          <input type="number" v-model.number="jumpPage" class="page-jump" min="1" :max="totalPages" @keyup.enter="doJump" />
          <button class="page-btn" @click="doJump">跳转</button>
          <button class="page-btn" :disabled="page >= totalPages" @click="page++; loadLogs()">下一页</button>
          <button class="page-btn" :disabled="page >= totalPages" @click="page = totalPages; loadLogs()">末页</button>
        </div>
      </div>
    </div>

    <!-- 详情弹窗 -->
    <div class="modal-overlay" v-if="detailVisible" @click.self="detailVisible = false">
      <div class="modal-card">
        <div class="modal-header">
          <div>
            <div class="modal-title">攻击详情</div>
            <div class="modal-subtitle">请求 ID: {{ d?.id }}</div>
          </div>
          <button class="modal-close" @click="detailVisible = false">
            <el-icon :size="18"><Close /></el-icon>
          </button>
        </div>
        <div class="modal-body" v-if="d">
          <div class="detail-grid">
            <div class="detail-row">
              <span class="detail-label">时间</span>
              <span class="detail-value">{{ fmt(d.timestamp) }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">所属站点</span>
              <span class="detail-value">{{ siteName(d) }}</span>
            </div>
            <div class="detail-row" v-if="d.domain">
              <span class="detail-label">访问域名</span>
              <span class="detail-value mono">{{ d.domain }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">来源 IP</span>
              <span class="detail-value mono">{{ d.client_ip }}</span>
            </div>
            <div class="detail-row" v-if="d.region">
              <span class="detail-label">地区</span>
              <span class="detail-value">{{ d.region }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">请求方法</span>
              <span class="detail-value"><span class="method-tag" :class="d.method?.toLowerCase()">{{ d.method }}</span></span>
            </div>
            <div class="detail-row">
              <span class="detail-label">请求路径</span>
              <span class="detail-value mono">{{ d.path }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">检测引擎</span>
              <span class="detail-value">
                <span class="engine-badge" :class="d.source === 'ai' ? 'ai' : 'rule'">
                  {{ d.source === 'ai' ? 'AI 智能检测' : '规则引擎' }}
                </span>
              </span>
            </div>
            <div class="detail-row">
              <span class="detail-label label-with-help">
                检测结果
                <button class="help-btn" type="button" @click.stop="toggleHelp('detail-result')" aria-label="检测结果说明">?</button>
              </span>
              <span class="detail-value">
                {{ d.rule_name }}
                <span class="plain-explain">{{ detectionExplain(d) }}</span>
              </span>
            </div>
            <div class="detail-row">
              <span class="detail-label">危险度</span>
              <span class="detail-value"><span class="severity-pill" :class="d.severity">{{ sevTxt(d.severity) }}</span></span>
            </div>
            <div class="detail-row" v-if="d.ai_reasoning">
              <span class="detail-label label-with-help">
                AI 分析
                <button class="help-btn" type="button" @click.stop="toggleHelp('ai-result')" aria-label="AI 分析说明">?</button>
              </span>
              <span class="detail-value">{{ d.ai_reasoning }}</span>
            </div>
          <div class="detail-row" v-if="d.source === 'ai'">
              <span class="detail-label">人工复核</span>
              <span class="detail-value review-actions">
                <button class="btn-ghost" @click.stop="markReviewed">确认有效</button>
                <button class="btn-primary danger" @click.stop="markFalsePositive">标记误报并学习白名单</button>
              </span>
            </div>
          </div>
          <div class="detail-help" v-if="helpKey === 'detail-result' || helpKey === 'ai-result'">
            <button class="help-close" @click="helpKey = ''">×</button>
            <strong>{{ helpKey === 'ai-result' ? 'AI 分析说明' : '检测结果说明' }}</strong>
            <p>{{ helpKey === 'ai-result' ? 'AI 分析会用更接近业务的语言说明这次访问哪里异常、可能影响什么。它用于辅助判断，仍建议结合路径、IP 和请求内容复核。' : '检测结果是系统命中的规则或 AI 判断名称，用来说明本次访问被拦截的主要原因。下面的解释会把它翻译成普通客户能看懂的话。' }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, Document, Cpu, Close } from '@element-plus/icons-vue'
import api from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const logs = ref([]), total = ref(0), page = ref(1), pageSize = ref(20), jumpPage = ref(1)
const loading = ref(false), detailVisible = ref(false), d = ref(null)
const sites = ref([])
const helpKey = ref('')
const filters = reactive({ site_id: '', client_ip: '', severity: '', source: '' })
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

function sevTxt(s) { return { critical:'严重', high:'高危', medium:'中危', low:'低危' }[s] || s }
function fmt(ts) { return ts ? new Date(ts).toLocaleString('zh-CN') : '-' }
function siteName(log) { return log?.site_name || log?.domain || '默认站点' }
function resetFilters() { Object.assign(filters, { site_id:'', client_ip:'', severity:'', source:'' }); page.value = 1; loadLogs() }
function toggleHelp(key) { helpKey.value = helpKey.value === key ? '' : key }
function detectionExplain(log) {
  if (!log) return ''
  const name = `${log.rule_name || log.rule_id || ''}`.toLowerCase()
  if (log.source === 'ai') return 'AI 判断这次访问行为异常，建议结合来源 IP 和请求路径复核。'
  if (name.includes('sql') || name.includes('sqli')) return '请求里像是在尝试拼接数据库语句，可能用于读取或修改数据。'
  if (name.includes('xss')) return '请求里包含可疑脚本，可能影响访问者浏览器安全。'
  if (name.includes('cmd') || name.includes('command')) return '请求疑似尝试执行服务器命令，风险较高。'
  if (name.includes('traversal') || name.includes('path')) return '请求疑似尝试访问非公开文件或目录。'
  if (name.includes('sensitive')) return '请求访问了敏感路径或敏感文件，建议确认是否为正常业务。'
  if (log.severity === 'critical' || log.severity === 'high') return '系统判断这次访问风险较高，建议优先查看来源和请求内容。'
  return '系统发现异常访问特征，建议结合业务场景确认是否为正常请求。'
}

async function loadSites() {
  try { sites.value = await api.get('/sites') || [] } catch {}
}

async function loadLogs() {
  loading.value = true
  try {
    const params = { page: page.value, limit: pageSize.value }
    if (filters.site_id) params.site_id = filters.site_id
    if (filters.client_ip) params.client_ip = filters.client_ip
    if (filters.severity) params.severity = filters.severity
    if (filters.source) params.source = filters.source
    const res = await api.get('/logs', { params })
    logs.value = res.data || []; total.value = res.total || 0
    jumpPage.value = page.value
  } catch {} finally { loading.value = false }
}
function doJump() {
  const p = Math.min(Math.max(Number(jumpPage.value) || 1, 1), totalPages.value)
  page.value = p
  jumpPage.value = p
  loadLogs()
}
function showDetail(row) { d.value = row; detailVisible.value = true; helpKey.value = '' }
async function markReviewed() {
  if (!d.value) return
  try {
    await api.post(`/logs/${d.value.id}/reviewed`)
    ElMessage.success('已标记为有效拦截')
    d.value.reviewed = true
    loadLogs()
  } catch {}
}
async function markFalsePositive() {
  if (!d.value) return
  try {
    await ElMessageBox.confirm('确认这是误报，并将来源 IP 加入白名单学习？', '误报学习', { type: 'warning' })
    await api.post(`/logs/${d.value.id}/false-positive`, { add_whitelist: true })
    ElMessage.success('已标记误报，来源 IP 已加入白名单')
    detailVisible.value = false
    loadLogs()
  } catch {}
}
onMounted(() => { loadSites(); loadLogs() })
</script>

<style scoped>
.logs-page { }

/* Filter Bar */
.filter-bar {
  background: var(--bg-card); border-radius: var(--radius-card); border: 1px solid var(--border);
  padding: 18px 20px; margin-bottom: 16px;
  display: flex; justify-content: space-between; align-items: flex-end;
}
.filter-group { display: flex; gap: 16px; }
.filter-item { display: flex; flex-direction: column; gap: 5px; }
.filter-item label { font-size: 11.5px; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.3px; }
.filter-input { width: 160px; }
.filter-select { width: 110px; cursor: pointer; }
.filter-select.site-select { width: 240px; }
.filter-actions { display: flex; gap: 8px; }
.btn-primary.danger { background: var(--danger); }
.btn-primary.danger:hover { background: #b91c1c; }

/* Table header override */
.table-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--card-pad); border-bottom: 1px solid var(--border-light);
}
.table-title { font-size: 14.5px; font-weight: 700; color: var(--text-primary); }
.table-count { font-size: 12px; color: var(--text-muted); background: var(--border-light); padding: 3px 10px; border-radius: 10px; }

.clickable-row { cursor: pointer; transition: background 0.15s; }
.clickable-row:hover { background: var(--bg-hover); }
.th-help, .label-with-help { display: inline-flex; align-items: center; gap: 6px; }
.help-btn {
  width: 18px; height: 18px; border-radius: 999px; border: 1px solid #cbd5e1;
  background: #fff; color: var(--text-secondary); font-size: 12px; line-height: 1; font-weight: 800;
  cursor: pointer;
}
.help-btn:hover { border-color: var(--primary); color: var(--primary-hover); background: var(--primary-light); }
.help-popover, .detail-help {
  background: #fff; border: 1px solid #dbe3ef; border-radius: var(--radius-card);
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.12);
  color: var(--text-secondary); font-size: 13px; line-height: 1.7;
}
.help-popover {
  position: absolute; right: 18px; top: 54px; z-index: 5;
  width: min(420px, calc(100% - 36px)); padding: 16px 18px;
}
.detail-help { position: relative; padding: 16px 18px; margin-top: 4px; }
.help-popover strong, .detail-help strong { display: block; color: var(--text-primary); margin-bottom: 6px; font-size: 14px; }
.help-close {
  position: absolute; top: 8px; right: 10px; border: none; background: transparent;
  color: var(--text-muted); font-size: 18px; cursor: pointer;
}

.time-cell { white-space: nowrap; }
.region-cell { max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-secondary); font-size: 12.5px; }
.site-cell {
  max-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  color: var(--text-secondary); font-weight: 700;
}
.path-cell { max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.result-cell { max-width: 170px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-link { font-size: 12px; color: var(--primary); font-weight: 600; }

/* Override method-tag to global colors */
.method-tag.get { background: #eff6ff; color: #2563eb; }
.method-tag.post { background: #fff7ed; color: #ea580c; }
.method-tag.put { background: #f5f3ff; color: #7c3aed; }
.method-tag.delete { background: #fef2f2; color: #dc2626; }

/* Pagination override */
.pagination-bar {
  display: flex; justify-content: space-between; align-items: center;
  padding: 12px 20px; border-top: 1px solid var(--border-light);
}

/* Modal detail */
.modal-subtitle { font-size: 12px; color: var(--text-muted); margin-top: 2px; font-family: var(--font-mono); }
.detail-grid { display: flex; flex-direction: column; gap: 14px; }
.detail-row { display: flex; gap: 16px; }
.detail-label { width: 80px; font-size: 12.5px; color: var(--text-muted); font-weight: 500; flex-shrink: 0; }
.detail-value { font-size: 13px; color: #1e293b; flex: 1; }
.plain-explain {
  display: block; margin-top: 7px; color: var(--text-secondary); line-height: 1.65;
  white-space: normal; font-size: 13px;
}
.review-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.review-actions .btn-primary,
.review-actions .btn-ghost { padding: 7px 12px; }

@media (max-width: 768px) {
  .filter-bar { flex-direction: column; gap: 12px; }
  .filter-group { flex-wrap: wrap; }
  .filter-input, .filter-select, .filter-select.site-select { width: 100%; min-width: 0; }
  .filter-actions { width: 100%; }
  .filter-actions .btn-primary { flex: 1; }
  .help-popover { left: 12px; right: 12px; width: auto; }
  .table-card { overflow-x: auto; }
  .data-table { min-width: 1040px; }
  .modal-card { width: 92vw; max-height: 90vh; }
  .pagination-bar { flex-direction: column; gap: 8px; }
}
</style>
