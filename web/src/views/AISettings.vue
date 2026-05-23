<template>
  <!-- 专业版引导 -->
  <div class="pro-gate" v-if="!isPro">
    <div class="gate-card">
      <div class="gate-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="48" height="48"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      </div>
      <h2>专业版功能</h2>
      <p>AI 智能检测引擎是专业版专属功能，支持接入任意 OpenAI 兼容模型实现智能攻击分析，升级后即可使用。</p>
      <div class="gate-features">
        <div class="gate-feat"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="20 6 9 17 4 12"/></svg> AI 驱动的智能攻击检测</div>
        <div class="gate-feat"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="20 6 9 17 4 12"/></svg> 支持任意 OpenAI 兼容模型</div>
        <div class="gate-feat"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="20 6 9 17 4 12"/></svg> 误报学习与规则自动沉淀</div>
      </div>
      <router-link to="/settings" class="gate-btn">升级专业版</router-link>
      <router-link to="/dashboard" class="gate-back">返回管理面板</router-link>
    </div>
  </div>

  <div class="ai-page" v-else>
    <!-- 头部 -->
    <div class="ai-header">
      <div class="heading-group">
        <div class="heading-icon violet"><el-icon :size="18"><Cpu /></el-icon></div>
        <div>
          <div class="page-heading">AI 模型配置</div>
          <div class="page-sub">AI 驱动的智能检测引擎，与规则引擎协同实现双重防护</div>
        </div>
      </div>
      <div class="header-right">
        <span class="edition-badge" :class="isPro ? 'pro' : 'community'">
          {{ isPro ? '专业版' : '社区版' }}
        </span>
      </div>
    </div>

    <!-- 用量卡片 (社区版显示) -->
    <div class="usage-banner" v-if="!isPro">
      <div class="usage-left">
        <div class="usage-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="22" height="22"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
        </div>
        <div>
          <div class="usage-title">今日 AI 调用额度</div>
          <div class="usage-bar-wrap">
            <div class="usage-bar">
              <div class="usage-fill" :style="{ width: usagePercent + '%' }" :class="{ warn: usagePercent > 70 }"></div>
            </div>
            <span class="usage-text">{{ aiUsage.today_used || 0 }} / {{ aiUsage.daily_limit || 50 }}</span>
          </div>
        </div>
      </div>
      <div class="usage-right">
        <div class="usage-remaining">
          剩余 <b>{{ aiUsage.remaining || 0 }}</b> 次
        </div>
        <router-link to="/settings" class="upgrade-link">升级专业版解锁无限调用</router-link>
      </div>
    </div>

    <!-- 引擎开关 -->
    <div class="engine-card">
      <div class="engine-left">
        <div class="engine-icon-wrap">
          <el-icon :size="22" color="#6366f1"><Cpu /></el-icon>
        </div>
        <div>
          <div class="engine-title">AI 智能检测引擎</div>
          <div class="engine-desc">未被规则引擎拦截的请求将由 AI 模型进行二次分析</div>
        </div>
      </div>
      <div class="engine-right">
        <label class="toggle-switch">
          <input type="checkbox" v-model="config.enabled" @change="saveGlobal" />
          <span class="toggle-track"></span>
        </label>
        <span class="toggle-label" :class="config.enabled ? 'on' : 'off'">
          {{ config.enabled ? '已启用' : '已禁用' }}
        </span>
      </div>
    </div>

    <!-- Pro 专属: AI 统计 -->
    <div class="kpi-row" v-if="isPro">
      <div class="kpi-card" v-for="k in kpiCards" :key="k.key">
        <div class="kpi-val" :style="{ color: k.color }">{{ k.value }}</div>
        <div class="kpi-label">{{ k.label }}</div>
      </div>
    </div>

    <!-- 双栏: 全局参数 + Provider -->
    <div class="two-col">
      <!-- 全局参数 -->
      <div class="card">
        <div class="card-h"><div class="card-title"><span class="dot indigo"></span>全局参数</div></div>
        <div class="card-b">
          <div class="form-grid">
            <div class="form-item">
              <label>超时策略</label>
              <div class="pill-group">
                <button class="pill" :class="{ active: config.fail_open === true }" @click="config.fail_open = true; saveGlobal()">放行（推荐）</button>
                <button class="pill" :class="{ active: config.fail_open === false }" @click="config.fail_open = false; saveGlobal()">拦截</button>
              </div>
            </div>
            <div class="form-item">
              <label>分析超时</label>
              <div class="input-unit">
                <input type="number" v-model.number="config.async_timeout" min="1" max="30" class="form-input" @change="saveGlobal" />
                <span class="unit">秒</span>
              </div>
            </div>
            <div class="form-item">
              <label>缓存有效期</label>
              <div class="input-unit">
                <input type="number" v-model.number="config.cache_ttl" min="60" max="3600" step="60" class="form-input" @change="saveGlobal" />
                <span class="unit">秒</span>
              </div>
            </div>
            <div class="form-item">
              <label>请求限速</label>
              <div class="input-unit">
                <input type="number" v-model.number="config.max_requests_per_min" min="10" max="600" step="10" class="form-input" @change="saveGlobal" />
                <span class="unit">次/分</span>
              </div>
            </div>
            <div class="form-item full">
              <label>高风险路径</label>
              <input v-model="highRiskPathsText" class="form-input" placeholder="/admin, /login, /upload, /payment" @change="saveGlobal" />
            </div>
          </div>
        </div>
      </div>

      <!-- Provider 配置 -->
      <div class="card">
        <div class="card-h">
          <div class="card-title"><span class="dot emerald"></span>模型配置</div>
          <span class="active-badge" v-if="config.enabled">活跃</span>
        </div>
        <div class="card-b">
          <div class="form-stack">
            <div class="form-item">
              <label>API 地址</label>
              <input v-model="config.providers.openai.base_url" placeholder="https://api.openai.com/v1" class="form-input" />
              <span class="hint">兼容 OpenAI API 格式的任何服务</span>
            </div>
            <div class="form-item">
              <label>API 密钥</label>
              <div class="password-wrap">
                <input v-model="config.providers.openai.api_key" :type="showKey ? 'text' : 'password'" placeholder="sk-..." class="form-input" />
                <button class="eye-btn" @click="showKey = !showKey">
                  <el-icon :size="14"><View v-if="!showKey" /><Hide v-else /></el-icon>
                </button>
              </div>
            </div>
            <div class="form-item">
              <label>模型 ID</label>
              <input v-model="config.providers.openai.model" placeholder="gpt-4o / deepseek-chat / ..." class="form-input" list="model-list" />
              <datalist id="model-list">
                <option value="gpt-4o-mini" />
                <option value="gpt-4o" />
                <option value="gpt-4o-mini" />
                <option value="deepseek-chat" />
                <option value="qwen-plus" />
                <option value="glm-4" />
              </datalist>
              <span class="hint">支持任意 OpenAI 兼容模型</span>
            </div>
          </div>
        </div>
        <div class="card-f">
          <button class="btn-primary" @click="saveProvider('openai')" :disabled="saving">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
          <button class="btn-ghost" @click="testAI" :disabled="testing">
            <el-icon :size="14"><Connection /></el-icon>
            {{ testing ? '测试中...' : '测试连接' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Pro 专属功能区 -->
    <template v-if="isPro">
      <!-- 业务上下文 -->
      <div class="card context-card">
        <div class="card-h"><div class="card-title"><span class="dot amber"></span>业务上下文策略</div></div>
        <div class="card-b">
          <div class="context-grid">
            <div class="ctx-item high"><span>登录 / 认证</span><strong>账号枚举、撞库、SQL 注入、异常自动化从严判断</strong></div>
            <div class="ctx-item high"><span>后台 / 管理</span><strong>越权、命令执行、路径穿越、敏感操作从严判断</strong></div>
            <div class="ctx-item high"><span>上传 / 导入</span><strong>WebShell、扩展伪装、压缩包滥用、路径穿越重点识别</strong></div>
            <div class="ctx-item high"><span>支付 / API</span><strong>重放、参数篡改、订单归属绕过、签名绕过重点识别</strong></div>
          </div>
        </div>
      </div>

      <!-- AI 建议规则 -->
      <div class="card">
        <div class="card-h">
          <div class="card-title"><span class="dot rose"></span>AI 建议规则</div>
          <button class="btn-sm" @click="loadAIInsight">刷新</button>
        </div>
        <div class="card-b" v-if="suggestions.length">
          <div class="suggestion-row" v-for="item in suggestions" :key="item.key">
            <div>
              <div class="sug-title">{{ item.rule_name || item.rule_id }}</div>
              <div class="sug-meta">{{ item.path }} · {{ item.count }} 次命中 · {{ sevTxt(item.severity) }} · 已复核 {{ item.reviewed || 0 }} 次</div>
            </div>
            <button class="btn-primary small" @click="promoteSuggestion(item)">沉淀为规则</button>
          </div>
        </div>
        <div class="card-b empty" v-else>暂无可沉淀的 AI 重复命中</div>
      </div>

      <!-- AI 命中审计 -->
      <div class="card">
        <div class="card-h">
          <div class="card-title">
            <span class="dot indigo"></span>AI 命中审计
          </div>
          <button class="btn-sm" @click="loadAIHits">刷新</button>
        </div>
        <div class="card-b no-pad" v-if="aiHits.length">
          <div class="hit-row" v-for="item in aiHits" :key="item.id">
            <div class="hit-main">
              <div class="hit-title">
                <span class="sev-pill" :class="item.severity">{{ sevTxt(item.severity) }}</span>
                <strong>{{ item.rule_name || item.rule_id }}</strong>
                <span class="review-badge" :class="{ done: item.reviewed, fp: item.false_positive }">
                  {{ item.false_positive ? '误报' : item.reviewed ? '已复核' : '待复核' }}
                </span>
              </div>
              <div class="hit-meta">
                <span>{{ siteName(item) }}</span>
                <span>{{ fmt(item.timestamp) }}</span>
                <span class="mono">{{ item.client_ip }}</span>
                <span>{{ item.method }} {{ item.path }}</span>
              </div>
              <div class="hit-reason" v-if="item.ai_reasoning">{{ item.ai_reasoning }}</div>
            </div>
            <div class="hit-actions">
              <button class="btn-ghost small" :disabled="item.reviewed && !item.false_positive" @click="markReviewed(item)">确认有效</button>
              <button class="btn-danger small" :disabled="item.false_positive" @click="markFalsePositive(item)">误报学习</button>
            </div>
          </div>
        </div>
        <div class="card-b empty" v-else>暂无 AI 命中审计记录</div>
        <div class="card-f pager" v-if="aiHitTotal > 0">
          <span>共 {{ aiHitTotal }} 条 / {{ aiHitTotalPages }} 页</span>
          <div class="page-btns">
            <button :disabled="aiHitPage <= 1" @click="aiHitPage = 1; loadAIHits()">首页</button>
            <button :disabled="aiHitPage <= 1" @click="aiHitPage--; loadAIHits()">上一页</button>
            <button :disabled="aiHitPage >= aiHitTotalPages" @click="aiHitPage++; loadAIHits()">下一页</button>
            <button :disabled="aiHitPage >= aiHitTotalPages" @click="aiHitPage = aiHitTotalPages; loadAIHits()">末页</button>
          </div>
        </div>
      </div>
    </template>

    <!-- 社区版: Pro 功能预览 -->
    <div class="pro-preview" v-if="!isPro">
      <div class="preview-header">
        <span class="preview-kicker">专业版功能</span>
        <h3>解锁 AI 全部能力</h3>
      </div>
      <div class="preview-grid">
        <div class="preview-card" v-for="f in proFeatures" :key="f.name">
          <div class="preview-icon" :class="f.color">
            <el-icon :size="18"><component :is="f.icon" /></el-icon>
          </div>
          <div>
            <b>{{ f.name }}</b>
            <span>{{ f.desc }}</span>
          </div>
        </div>
      </div>
      <div class="preview-footer">
        <router-link to="/settings" class="btn-primary">前往升级专业版</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, reactive, inject, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Cpu, Connection, View, Hide, SetUp, Warning, DataAnalysis } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const isPro = inject('isPro', ref(false))
const router = useRouter()

const saving = ref(false), testing = ref(false)
const showKey = ref(false)
const aiStats = ref({})
const aiUsage = ref({ today_used: 0, daily_limit: 50, remaining: 50, is_pro: false })
const suggestions = ref([])
const aiHits = ref([])
const aiHitTotal = ref(0)
const aiHitPage = ref(1)
const highRiskPathsText = ref('')
const aiHitTotalPages = computed(() => Math.max(1, Math.ceil(aiHitTotal.value / 5)))

const usagePercent = computed(() => {
  const limit = aiUsage.value.daily_limit || 50
  return Math.min(100, Math.round((aiUsage.value.today_used || 0) / limit * 100))
})

const config = reactive({
  enabled: false, provider: 'openai', async_timeout: 5, cache_ttl: 300,
  max_requests_per_min: 60, fail_open: true,
  providers: {
    openai: { api_key: '', model: '', base_url: '' }
  }
})

const kpiCards = computed(() => [
  { key: 'ai', label: 'AI 拦截', value: aiStats.value.ai_count || 0, color: '#6366f1' },
  { key: 'eff', label: '有效拦截', value: aiStats.value.ai_effective_blocked || 0, color: '#10b981' },
  { key: 'fp', label: '误报标记', value: aiStats.value.ai_false_positive || 0, color: '#ef4444' },
  { key: 'rev', label: '人工复核', value: aiStats.value.ai_reviewed || 0, color: '#f59e0b' },
])

const proFeatures = [
  { name: '无限 AI 调用', desc: '大流量站无限制使用 AI 检测', icon: DataAnalysis, color: 'indigo' },
  { name: 'AI 详细分析报告', desc: '攻击原理、危害等级、攻击者意图解读', icon: Warning, color: 'rose' },
  { name: '误报学习', desc: '标记误报自动调优，越用越准', icon: Cpu, color: 'violet' },
  { name: 'AI 规则生成', desc: '自然语言描述需求，自动生成检测规则', icon: SetUp, color: 'amber' },
  { name: '威胁画像分析', desc: 'AI 分析攻击者行为模式与趋势', icon: DataAnalysis, color: 'emerald' },
  { name: '多模型支持', desc: 'Claude / 自定义 / 本地模型，数据不出境', icon: Connection, color: 'cyan' },
]

async function load() {
  try {
    const r = await api.get('/ai/providers')
    config.enabled = r.enabled
    config.provider = 'openai'
    config.async_timeout = r.async_timeout || config.async_timeout
    config.cache_ttl = r.cache_ttl || config.cache_ttl
    config.max_requests_per_min = r.max_requests_per_min || config.max_requests_per_min
    config.fail_open = r.fail_open
    highRiskPathsText.value = (r.high_risk_paths || []).join(', ')
    if (r.providers?.openai) Object.assign(config.providers.openai, r.providers.openai)
  } catch {}
}

async function loadUsage() {
  try {
    aiUsage.value = await api.get('/ai/usage') || aiUsage.value
  } catch {}
}

async function saveGlobal() {
  try {
    await api.put('/ai/global', {
      enabled: config.enabled, provider: config.provider,
      async_timeout: config.async_timeout, cache_ttl: config.cache_ttl,
      max_requests_per_min: config.max_requests_per_min, fail_open: config.fail_open,
      high_risk_paths: highRiskPathsText.value.split(',').map(s => s.trim()).filter(Boolean),
    })
    ElMessage.success('全局配置已更新')
  } catch {}
}

async function saveProvider(name) {
  saving.value = true
  try {
    await api.put(`/ai/providers/${name}`, config.providers[name])
    ElMessage.success('模型配置已保存')
  } catch {} finally { saving.value = false }
}

async function loadAIInsight() {
  if (!isPro.value) return
  try {
    const stats = await api.get('/ai/stats', { params: { hours: 24 } })
    aiStats.value = stats || {}
    const list = await api.get('/ai/suggestions', { params: { hours: 168, min_count: 2, limit: 10 } })
    suggestions.value = list.data || []
  } catch {}
}

async function loadAIHits() {
  if (!isPro.value) return
  try {
    const res = await api.get('/logs', { params: { source: 'ai', page: aiHitPage.value, limit: 5 } })
    aiHits.value = res.data || []
    aiHitTotal.value = res.total || 0
  } catch {}
}

async function promoteSuggestion(item) {
  if (!isPro.value) return
  try {
    await api.post('/ai/suggestions/promote', {
      name: `AI 建议规则：${item.path}`,
      description: `${item.rule_name || item.rule_id} 重复命中 ${item.count} 次，建议人工复核后启用`,
      severity: item.severity || 'medium',
      patterns: [item.pattern],
      enabled: false,
    })
    ElMessage.success('已生成禁用状态的建议规则，请到规则管理复核启用')
    loadAIInsight()
  } catch {}
}

async function markReviewed(item) {
  if (!isPro.value) return
  try {
    await api.post(`/logs/${item.id}/reviewed`)
    ElMessage.success('已确认 AI 命中有效')
    item.reviewed = true
    await Promise.all([loadAIInsight(), loadAIHits()])
  } catch {}
}

async function markFalsePositive(item) {
  if (!isPro.value) return
  try {
    await ElMessageBox.confirm('确认这是 AI 误报，并将来源 IP 加入白名单学习？', 'AI 误报学习', { type: 'warning' })
    await api.post(`/logs/${item.id}/false-positive`, { add_whitelist: true, note: `AI 误报学习：${item.path}` })
    ElMessage.success('已标记误报，来源 IP 已加入白名单')
    item.reviewed = true
    item.false_positive = true
    await Promise.all([loadAIInsight(), loadAIHits()])
  } catch {}
}

async function testAI() {
  testing.value = true
  try {
    const r = await api.post('/ai/test')
    r.status === 'ok' ? ElMessage.success(r.message) : ElMessage.warning(r.message || '测试完成')
  } catch {} finally { testing.value = false }
}

function sevTxt(s) {
  return { critical: '严重', high: '高危', medium: '中危', low: '低危' }[s] || s || '-'
}
function siteName(item) { return item?.site_name || item?.domain || '默认站点' }
function fmt(ts) { return ts ? new Date(ts).toLocaleString('zh-CN') : '-' }

function loadAll() {
  load()
  loadUsage()
  if (isPro.value) {
    loadAIInsight()
    loadAIHits()
  }
}

watch(isPro, () => loadAll(), { immediate: true })
</script>

<style scoped>
.ai-page { }

/* Pro Gate */
.pro-gate { display: flex; align-items: center; justify-content: center; min-height: 60vh; }
.gate-card {
  text-align: center; max-width: 420px; padding: 48px 40px;
  background: var(--bg-card); border: 1px solid var(--border); border-radius: 20px;
  box-shadow: 0 4px 24px rgba(0,0,0,.06);
}
.gate-icon { margin-bottom: 20px; color: #d97706; }
.gate-card h2 { font-size: 22px; font-weight: 800; color: var(--text-primary); margin-bottom: 10px; }
.gate-card p { font-size: 14px; color: var(--text-muted); line-height: 1.6; margin-bottom: 20px; }
.gate-features { text-align: left; margin-bottom: 24px; }
.gate-feat {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 0; font-size: 13px; color: var(--text-secondary); font-weight: 500;
}
.gate-feat svg { color: #059669; flex-shrink: 0; }
.gate-btn {
  display: inline-block; padding: 10px 28px; border-radius: 10px;
  background: var(--primary); color: #fff; font-size: 14px; font-weight: 700;
  text-decoration: none; transition: all 0.2s;
}
.gate-btn:hover { background: var(--primary-hover); }
.gate-back {
  display: block; margin-top: 12px; font-size: 13px; color: var(--text-muted);
  text-decoration: none;
}
.gate-back:hover { color: var(--primary); }

/* Header */
.ai-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 20px;
}
.edition-badge {
  padding: 4px 12px; border-radius: 999px; font-size: 11px; font-weight: 700;
}
.edition-badge.pro { background: #f5f3ff; color: #7c3aed; }
.edition-badge.community { background: var(--border-light); color: var(--text-secondary); }

/* Usage Banner */
.usage-banner {
  display: flex; align-items: center; justify-content: space-between; gap: 20px;
  background: linear-gradient(135deg, var(--primary) 0%, #7c3aed 100%);
  border-radius: var(--radius-card); padding: 20px 24px; margin-bottom: 16px; color: #fff;
}
.usage-left { display: flex; align-items: center; gap: 16px; }
.usage-icon {
  width: 44px; height: 44px; border-radius: 12px;
  background: rgba(255,255,255,0.15); display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.usage-title { font-size: 13px; font-weight: 600; opacity: 0.9; margin-bottom: 8px; }
.usage-bar-wrap { display: flex; align-items: center; gap: 10px; }
.usage-bar {
  width: 200px; height: 6px; background: rgba(255,255,255,0.2);
  border-radius: 999px; overflow: hidden;
}
.usage-fill { height: 100%; border-radius: inherit; background: #fff; transition: width 0.5s ease; }
.usage-fill.warn { background: #fbbf24; }
.usage-text { font-size: 12px; font-weight: 700; opacity: 0.9; white-space: nowrap; }
.usage-right { text-align: right; flex-shrink: 0; }
.usage-remaining { font-size: 14px; font-weight: 700; margin-bottom: 4px; }
.usage-remaining b { font-size: 20px; }
.upgrade-link {
  font-size: 12px; color: rgba(255,255,255,0.8); text-decoration: none;
  border-bottom: 1px solid rgba(255,255,255,0.3);
}
.upgrade-link:hover { color: #fff; }

/* Engine Card */
.engine-card {
  display: flex; justify-content: space-between; align-items: center;
  background: var(--bg-card); border-radius: var(--radius-card); border: 1px solid var(--border);
  padding: 18px 22px; margin-bottom: 16px;
}
.engine-left { display: flex; align-items: center; gap: 14px; }
.engine-icon-wrap {
  width: 44px; height: 44px; border-radius: 12px; background: var(--primary-light);
  display: flex; align-items: center; justify-content: center;
}
.engine-title { font-size: 15px; font-weight: 700; color: var(--text-primary); }
.engine-desc { font-size: 12.5px; color: var(--text-muted); margin-top: 2px; }
.engine-right { display: flex; align-items: center; gap: 10px; }
.toggle-label { font-size: 13px; font-weight: 600; }
.toggle-label.on { color: #059669; }
.toggle-label.off { color: var(--text-muted); }

/* KPI */
.kpi-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 16px; }
.kpi-card {
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  padding: 16px 18px; text-align: center;
}
.kpi-val { font-size: 26px; font-weight: 800; line-height: 1; }
.kpi-label { font-size: 12px; color: var(--text-muted); font-weight: 600; margin-top: 4px; }

/* Two column */
.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }

/* Card internal layout */
.card-h {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 18px; border-bottom: 1px solid var(--border-light);
}
.card-title { display: flex; align-items: center; gap: 8px; font-size: 14px; font-weight: 700; color: var(--text-primary); }
.dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.dot.indigo { background: var(--primary); }
.dot.emerald { background: #10b981; }
.dot.rose { background: #e11d48; }
.dot.amber { background: #d97706; }
.card-b { padding: 18px; }
.card-b.no-pad { padding: 0; }
.card-b.empty { color: var(--text-muted); font-size: 13px; }
.card-f {
  display: flex; align-items: center; gap: 8px;
  padding: 12px 18px; border-top: 1px solid var(--border-light);
}
.card-f.pager { justify-content: space-between; }

.active-badge {
  padding: 3px 10px; border-radius: 6px; font-size: 11px; font-weight: 700;
  background: #ecfdf5; color: #059669;
}

/* Form */
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.form-grid .form-item.full { grid-column: 1 / -1; }
.form-stack { display: flex; flex-direction: column; gap: 14px; }
.form-item label { display: block; font-size: 12px; font-weight: 600; color: var(--text-secondary); margin-bottom: 5px; }
.hint { display: block; font-size: 11px; color: var(--text-muted); margin-top: 3px; }
.input-unit { display: flex; align-items: center; gap: 6px; }
.unit { font-size: 12px; color: var(--text-muted); }
.password-wrap { position: relative; }
.password-wrap .form-input { padding-right: 36px; }
.eye-btn {
  position: absolute; right: 8px; top: 50%; transform: translateY(-50%);
  background: none; border: none; color: var(--text-muted); cursor: pointer; padding: 4px;
}
.eye-btn:hover { color: var(--text-secondary); }

.pill-group { display: flex; gap: 6px; }
.pill {
  padding: 6px 14px; border-radius: 8px; border: 1px solid var(--border);
  background: #fff; color: var(--text-secondary); font-size: 12.5px; font-weight: 500;
  cursor: pointer; transition: all 0.2s;
}
.pill:hover { border-color: #cbd5e1; }
.pill.active { background: var(--primary); border-color: var(--primary); color: #fff; }

.btn-primary.small { padding: 6px 12px; font-size: 12px; }
.btn-ghost.small { padding: 6px 10px; font-size: 12px; }
.btn-danger.small {
  border-radius: 8px; padding: 6px 10px; font-size: 12px; font-weight: 700; cursor: pointer;
  border: 1px solid #fecaca; background: #fef2f2; color: var(--danger);
}
.btn-danger.small:disabled { opacity: 0.45; cursor: not-allowed; }

/* Context */
.context-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; }
.ctx-item {
  min-height: 78px; padding: 13px 14px; border-radius: 10px;
  background: var(--bg-hover); border: 1px solid var(--border);
}
.ctx-item span, .ctx-item strong { display: block; }
.ctx-item span { color: var(--primary-text); font-size: 12px; font-weight: 800; }
.ctx-item strong { margin-top: 8px; color: #334155; font-size: 12.5px; line-height: 1.55; }

/* Suggestions */
.suggestion-row {
  display: flex; align-items: center; justify-content: space-between; gap: 14px;
  padding: 12px 0; border-bottom: 1px solid var(--border-light);
}
.suggestion-row:last-child { border-bottom: none; }
.sug-title { font-size: 13px; font-weight: 700; color: var(--text-primary); }
.sug-meta { font-size: 12px; color: var(--text-secondary); margin-top: 3px; }

/* AI Hits */
.hit-row {
  display: flex; justify-content: space-between; gap: 14px;
  padding: 13px 18px; border-bottom: 1px solid var(--border-light);
}
.hit-row:last-child { border-bottom: none; }
.hit-main { min-width: 0; flex: 1; }
.hit-title { display: flex; align-items: center; gap: 8px; min-width: 0; }
.hit-title strong { color: var(--text-primary); font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.hit-meta { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 7px; color: var(--text-secondary); font-size: 12px; }
.hit-reason { margin-top: 8px; color: var(--text-secondary); font-size: 12.5px; line-height: 1.55; }
.hit-actions { display: flex; align-items: flex-start; gap: 8px; flex-shrink: 0; }

.review-badge { border-radius: 999px; padding: 3px 8px; background: var(--border-light); color: var(--text-secondary); font-size: 11px; font-weight: 800; }
.review-badge.done { background: #ecfdf5; color: #15803d; }
.review-badge.fp { background: #fef2f2; color: var(--danger); }

/* Pager */
.pager { font-size: 12px; color: var(--text-muted); }
.page-btns { display: flex; gap: 6px; }
.page-btns button {
  padding: 5px 10px; border: 1px solid var(--border); border-radius: 7px;
  background: #fff; color: var(--text-secondary); font-size: 12px; cursor: pointer;
}
.page-btns button:disabled { opacity: 0.45; cursor: not-allowed; }

/* Pro Preview */
.pro-preview {
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  padding: 28px; margin-top: 8px;
}
.preview-header { margin-bottom: 20px; }
.preview-kicker {
  display: inline-flex; margin-bottom: 8px; padding: 4px 10px; border-radius: 999px;
  background: #f5f3ff; color: #7c3aed; font-size: 11px; font-weight: 800;
}
.preview-header h3 { margin: 0; font-size: 20px; font-weight: 800; color: var(--text-primary); }
.preview-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 24px; }
.preview-card {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 14px; border-radius: 10px; background: var(--bg-hover); border: 1px solid var(--border);
}
.preview-icon {
  width: 36px; height: 36px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.preview-icon.indigo { background: var(--primary-light); color: var(--primary); }
.preview-icon.rose { background: #fff1f2; color: #e11d48; }
.preview-icon.violet { background: #f5f3ff; color: #7c3aed; }
.preview-icon.amber { background: #fffbeb; color: #d97706; }
.preview-icon.emerald { background: #ecfdf5; color: #10b981; }
.preview-icon.cyan { background: #ecfeff; color: #0891b2; }
.preview-card b { display: block; font-size: 13px; color: var(--text-primary); margin-bottom: 2px; }
.preview-card span { font-size: 11.5px; color: var(--text-secondary); line-height: 1.5; }
.preview-footer { text-align: center; }

/* Responsive */
@media (max-width: 768px) {
  .ai-header { flex-direction: column; align-items: flex-start; gap: 8px; }
  .usage-banner { flex-direction: column; align-items: stretch; }
  .usage-right { text-align: left; }
  .usage-bar { width: 100%; }
  .engine-card { flex-direction: column; align-items: flex-start; gap: 12px; }
  .kpi-row { grid-template-columns: 1fr 1fr; }
  .two-col { grid-template-columns: 1fr; }
  .form-grid { grid-template-columns: 1fr; }
  .context-grid { grid-template-columns: 1fr; }
  .hit-row { flex-direction: column; }
  .hit-actions { width: 100%; }
  .preview-grid { grid-template-columns: 1fr; }
}
</style>
