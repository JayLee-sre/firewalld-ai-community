<template>
  <div class="ti-page">
    <!-- 头部 -->
    <div class="ti-header">
      <div class="heading-group">
        <div class="heading-icon rose"><el-icon :size="18"><Warning /></el-icon></div>
        <div>
          <div class="page-heading">威胁情报</div>
          <div class="page-sub">自动同步恶意 IP 情报源，实时更新黑名单防护</div>
        </div>
      </div>
      <div class="header-actions">
        <button class="btn-ghost" :disabled="loading" @click="loadStatus">
          <el-icon :size="14"><Refresh /></el-icon> 刷新
        </button>
        <button class="btn-primary" :disabled="syncing" @click="triggerSync">
          <el-icon :size="14"><RefreshRight /></el-icon>
          {{ syncing ? '同步中...' : '立即同步' }}
        </button>
      </div>
    </div>

    <!-- 状态卡片 -->
    <div class="kpi-row">
      <div class="kpi-card">
        <div class="kpi-icon blue">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
        </div>
        <div>
          <div class="kpi-label">情报源</div>
          <div class="kpi-val">{{ status.provider || 'AbuseIPDB' }}</div>
        </div>
      </div>
      <div class="kpi-card">
        <div class="kpi-icon emerald">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        </div>
        <div>
          <div class="kpi-label">上次同步</div>
          <div class="kpi-val">{{ formatTime(status.last_sync) }}</div>
        </div>
      </div>
      <div class="kpi-card">
        <div class="kpi-icon rose">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        </div>
        <div>
          <div class="kpi-label">已同步恶意 IP</div>
          <div class="kpi-val">{{ threatIPs.length || 0 }}</div>
        </div>
      </div>
      <div class="kpi-card">
        <div class="kpi-icon amber">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        </div>
        <div>
          <div class="kpi-label">自动同步间隔</div>
          <div class="kpi-val">6 小时</div>
        </div>
      </div>
    </div>

    <!-- 双栏 -->
    <div class="two-col">
      <!-- API 配置 -->
      <div class="card">
        <div class="card-h">
          <div class="card-title">
            <span class="dot indigo"></span>API Key 配置
          </div>
        </div>
        <div class="card-b">
          <div class="field">
            <label>AbuseIPDB API Key</label>
            <div class="key-row">
              <input v-model="apiKey" :type="showKey ? 'text' : 'password'" placeholder="输入你的 AbuseIPDB API Key" />
              <button class="eye-btn" type="button" @click="showKey = !showKey">
                <el-icon :size="15"><View v-if="!showKey" /><Hide v-else /></el-icon>
              </button>
            </div>
          </div>
          <div class="hint">
            在 <a href="https://www.abuseipdb.com/register" target="_blank" rel="noopener">AbuseIPDB</a> 注册获取免费 API Key，免费版每日可查询 1000 次。
          </div>
          <button class="btn-primary full" :disabled="saving" @click="saveConfig">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>

      <!-- 同步说明 -->
      <div class="card">
        <div class="card-h">
          <div class="card-title">
            <span class="dot emerald"></span>同步机制
          </div>
        </div>
        <div class="card-b">
          <div class="steps">
            <div class="step">
              <div class="step-num">1</div>
              <div>
                <b>定时拉取</b>
                <p>系统每 6 小时自动从 AbuseIPDB 拉取高置信度（abuseConfidenceScore ≥ 75）恶意 IP。</p>
              </div>
            </div>
            <div class="step">
              <div class="step-num">2</div>
              <div>
                <b>自动封禁</b>
                <p>同步的恶意 IP 自动加入黑名单，WAF 请求管道实时拦截来自这些 IP 的流量。</p>
              </div>
            </div>
            <div class="step">
              <div class="step-num">3</div>
              <div>
                <b>定期清理</b>
                <p>超过 30 天未被报告的 IP 自动移出黑名单，保持情报的时效性和准确性。</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- IP 列表 -->
    <div class="card ip-card" v-if="threatIPs.length > 0">
      <div class="card-h">
        <div class="card-title">
          <span class="dot rose"></span>已同步恶意 IP
        </div>
        <span class="badge">{{ threatIPs.length }}</span>
      </div>
      <div class="card-b no-pad">
        <table class="ip-table">
          <thead>
            <tr>
              <th>#</th>
              <th>IP 地址</th>
              <th>同步时间</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, i) in pagedIPs" :key="item.ip">
              <td class="idx">{{ (ipPage - 1) * ipPageSize + i + 1 }}</td>
              <td class="ip-mono">{{ item.ip }}</td>
              <td class="ip-time">{{ formatTime(item.created_at) }}</td>
              <td><span class="st-blocked">已封禁</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="card-footer" v-if="threatIPs.length > ipPageSize">
        <span class="page-info">共 {{ threatIPs.length }} 条，第 {{ ipPage }} / {{ ipTotalPages }} 页</span>
        <div class="page-btns">
          <button :disabled="ipPage <= 1" @click="ipPage--">上一页</button>
          <button :disabled="ipPage >= ipTotalPages" @click="ipPage++">下一页</button>
        </div>
      </div>
    </div>

    <div class="empty-card" v-else>
      <svg viewBox="0 0 24 24" fill="none" stroke="#cbd5e1" stroke-width="1.5" width="40" height="40"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
      <p>暂无同步数据</p>
      <span>配置 API Key 后点击"立即同步"开始获取恶意 IP 情报</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Warning, RefreshRight, Refresh, View, Hide } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const status = ref({ provider: 'abuseipdb', last_sync: null, ip_count: 0 })
const threatIPs = ref([])
const loading = ref(false), syncing = ref(false), saving = ref(false)
const apiKey = ref(''), showKey = ref(false)
const ipPage = ref(1)
const ipPageSize = 20

const ipTotalPages = computed(() => Math.max(1, Math.ceil(threatIPs.value.length / ipPageSize)))
const pagedIPs = computed(() => threatIPs.value.slice((ipPage.value - 1) * ipPageSize, ipPage.value * ipPageSize))

function formatTime(ts) {
  if (!ts) return '从未同步'
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return '从未同步'
  return d.toLocaleString('zh-CN')
}

async function loadStatus() {
  loading.value = true
  try {
    const res = await api.get('/threatintel/status')
    status.value = res || { provider: 'abuseipdb', last_sync: null, ip_count: 0 }
    threatIPs.value = res?.threat_ips || []
    ipPage.value = 1
  } catch {} finally { loading.value = false }
}

async function triggerSync() {
  syncing.value = true
  try {
    const res = await api.post('/threatintel/sync')
    ElMessage.success(res?.message || '同步已触发，请稍后刷新状态')
    setTimeout(loadStatus, 5000)
  } catch {} finally { syncing.value = false }
}

async function saveConfig() {
  if (!apiKey.value.trim()) { ElMessage.warning('请输入 API Key'); return }
  saving.value = true
  try {
    await api.put('/threatintel/config', { api_key: apiKey.value.trim() })
    ElMessage.success('配置已保存')
    apiKey.value = ''
  } catch {} finally { saving.value = false }
}

onMounted(loadStatus)
</script>

<style scoped>
.ti-page { }

/* Header */
.ti-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 20px;
}
.header-actions { display: flex; gap: 8px; }

/* KPI */
.kpi-row {
  display: grid; grid-template-columns: repeat(4, 1fr);
  gap: 12px; margin-bottom: 16px;
}
.kpi-card {
  display: flex; align-items: center; gap: 14px;
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: var(--radius-card); padding: 16px 18px;
}
.kpi-icon {
  width: 42px; height: 42px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.kpi-icon svg { width: 20px; height: 20px; }
.kpi-icon.blue { background: #eff6ff; color: #3b82f6; }
.kpi-icon.emerald { background: #ecfdf5; color: #10b981; }
.kpi-icon.rose { background: #fff1f2; color: #e11d48; }
.kpi-icon.amber { background: #fffbeb; color: #d97706; }
.kpi-label { font-size: 11.5px; color: var(--text-muted); font-weight: 600; margin-bottom: 2px; }
.kpi-val { font-size: 15px; font-weight: 700; color: var(--text-primary); }

/* Two column */
.two-col {
  display: grid; grid-template-columns: 1fr 1fr;
  gap: 16px; margin-bottom: 16px;
}

/* Card (reuse global) */
.card-h {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 18px; border-bottom: 1px solid var(--border-light);
}
.card-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 700; color: var(--text-primary);
}
.dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.dot.indigo { background: var(--primary); }
.dot.emerald { background: #10b981; }
.dot.rose { background: #e11d48; }
.card-b { padding: 18px; }
.card-b.no-pad { padding: 0; }

.badge {
  font-size: 11px; font-weight: 700; color: var(--primary);
  background: var(--primary-light); padding: 2px 10px; border-radius: 999px;
}

/* Field */
.field { margin-bottom: 14px; }
.field label {
  display: block; margin-bottom: 6px; font-size: 12px;
  color: var(--text-secondary); font-weight: 700;
}
.key-row { position: relative; }
.key-row input {
  width: 100%; height: 40px; padding: 0 40px 0 12px;
  border: 1px solid var(--border); border-radius: var(--radius-input);
  font-size: 13px; color: #1e293b; outline: none;
  background: var(--bg-hover); transition: all 0.2s;
}
.key-row input:focus {
  border-color: #a5b4fc;
  box-shadow: 0 0 0 3px rgba(99,102,241,0.1);
  background: #fff;
}
.eye-btn {
  position: absolute; right: 6px; top: 50%; transform: translateY(-50%);
  width: 28px; height: 28px; border: none; border-radius: 6px;
  background: transparent; color: var(--text-muted); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
}
.eye-btn:hover { background: var(--border-light); color: var(--primary); }
.hint {
  font-size: 12px; color: var(--text-muted); margin-bottom: 16px; line-height: 1.6;
}
.hint a { color: var(--primary); text-decoration: none; font-weight: 600; }
.hint a:hover { text-decoration: underline; }
.btn-primary.full { width: 100%; justify-content: center; }

/* Steps */
.steps { display: flex; flex-direction: column; gap: 16px; }
.step { display: flex; gap: 14px; }
.step-num {
  width: 28px; height: 28px; border-radius: 9px;
  background: var(--primary-light); color: var(--primary);
  font-size: 13px; font-weight: 800;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; margin-top: 2px;
}
.step b { display: block; font-size: 13px; font-weight: 700; color: var(--text-primary); margin-bottom: 3px; }
.step p { font-size: 12px; color: var(--text-secondary); line-height: 1.6; margin: 0; }

/* IP Table (reuse global .data-table) */
.ip-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.ip-table thead th {
  padding: 10px 16px; text-align: left; font-size: 11px; font-weight: 700;
  color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px;
  background: var(--bg-hover); border-bottom: 1px solid var(--border);
}
.ip-table tbody tr {
  border-bottom: 1px solid var(--border-light); transition: background 0.15s;
}
.ip-table tbody tr:hover { background: var(--bg-hover); }
.ip-table tbody tr:last-child { border-bottom: none; }
.ip-table td { padding: 10px 16px; }
.idx { color: #cbd5e1; font-weight: 700; font-size: 12px; width: 40px; }
.ip-mono { font-family: var(--font-mono); font-size: 12.5px; color: var(--text-primary); font-weight: 600; }
.ip-time { font-size: 12px; color: var(--text-muted); }
.st-blocked {
  font-size: 11px; font-weight: 700; padding: 2px 8px;
  border-radius: 4px; background: #fef2f2; color: var(--danger);
}

/* Footer pager */
.card-footer {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 18px; border-top: 1px solid var(--border-light);
}
.page-info { font-size: 12px; color: var(--text-muted); }
.page-btns { display: flex; gap: 6px; }
.page-btns button {
  padding: 5px 12px; border: 1px solid var(--border); border-radius: 7px;
  background: #fff; color: var(--text-secondary); font-size: 12px; font-weight: 600;
  cursor: pointer; transition: all 0.2s;
}
.page-btns button:hover:not(:disabled) { background: var(--bg-hover); border-color: #cbd5e1; }
.page-btns button:disabled { opacity: .4; cursor: not-allowed; }

/* Empty */
.empty-card {
  text-align: center; padding: 48px 24px;
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: var(--radius-card);
}
.empty-card p { font-size: 15px; font-weight: 700; color: var(--text-secondary); margin: 14px 0 4px; }
.empty-card span { font-size: 12px; color: var(--text-muted); }

/* Responsive */
@media (max-width: 768px) {
  .ti-header { flex-direction: column; align-items: flex-start; gap: 12px; }
  .header-actions { width: 100%; }
  .header-actions button { flex: 1; justify-content: center; }
  .kpi-row { grid-template-columns: 1fr 1fr; }
  .two-col { grid-template-columns: 1fr; }
}
</style>
