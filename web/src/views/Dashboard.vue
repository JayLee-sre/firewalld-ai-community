<template>
  <div class="dashboard">
    <div class="scope-toolbar">
      <div>
        <div class="scope-title">安全态势</div>
        <div class="scope-desc">按站点查看攻击趋势、AI 命中和实时事件</div>
      </div>
      <div class="scope-control">
        <label>站点范围</label>
        <select v-model="selectedSite" class="scope-select" @change="reloadBySite">
          <option value="">全部站点</option>
          <option v-for="site in sites" :key="site.id" :value="site.id">
            {{ site.name }} · {{ site.domains?.[0] || site.upstream }}
          </option>
        </select>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card" v-for="(s, i) in statCards" :key="s.key"
        :style="{ '--accent': s.accent, animationDelay: `${i * 80}ms` }">
        <div class="stat-header">
          <span class="stat-label">{{ s.label }}</span>
          <div class="stat-badge" :class="s.color">
            <el-icon :size="16"><component :is="s.icon" /></el-icon>
          </div>
        </div>
        <div class="stat-value" :class="s.textColor">{{ s.displayValue }}</div>
        <div class="stat-footer">{{ s.desc }}</div>
      </div>
    </div>

    <!-- 左右两栏 -->
    <div class="mid-grid">

      <!-- 左：威胁分布 -->
      <section class="card">
        <div class="card-head">
          <div>
            <div class="card-title">威胁等级分布</div>
            <div class="card-sub">按严重程度聚合统计</div>
          </div>
          <span class="live-badge" :class="{ on: isConnected }">
            <span class="live-dot"></span>
            {{ isConnected ? '实时' : '离线' }}
          </span>
        </div>
        <div class="chart-wrap">
          <div ref="chartRef" class="echart" v-show="chartReady"></div>
          <div class="chart-loading" v-if="!chartReady">
            <div class="spinner"></div>
          </div>
        </div>
        <div class="sev-bar">
          <div class="sev-item" v-for="item in severityRows" :key="item.key">
            <span class="sev-dot" :class="item.key"></span>
            <span>{{ item.label }}</span>
            <strong>{{ item.count }}</strong>
          </div>
        </div>
      </section>

      <!-- 右：来源排行 -->
      <section class="card">
        <div class="card-head">
          <div>
            <div class="card-title">来源排行</div>
            <div class="card-sub">地区与最近活跃 IP</div>
          </div>
        </div>
        <div class="rank-body" v-if="regionRank.length">
          <div class="rank-row" v-for="(item, i) in pagedRegionRank" :key="item.region">
            <span class="rank-idx" :class="{ hot: regionPageOffset + i < 3 }">{{ regionPageOffset + i + 1 }}</span>
            <div class="rank-info">
              <span class="rank-name">{{ item.region || '未知地区' }}</span>
              <div class="rank-bar"><span :style="{ width: rankWidth(item.count) }"></span></div>
            </div>
            <strong class="rank-cnt">{{ item.count }}</strong>
          </div>
        </div>
        <div class="empty" v-else>暂无地区聚合数据</div>
        <div class="mini-pager" v-if="regionTotalPages > 1">
          <button :disabled="regionPage <= 1" @click="regionPage--">上一页</button>
          <span>{{ regionPage }} / {{ regionTotalPages }}</span>
          <button :disabled="regionPage >= regionTotalPages" @click="regionPage++">下一页</button>
        </div>

        <div class="ip-bar" v-if="activeIPs.length">
          <span class="ip-label">活跃 IP</span>
          <span v-for="ip in activeIPs" :key="ip" class="ip-tag">{{ ip }}</span>
        </div>
      </section>

    </div>

    <!-- 高频攻击路径 -->
    <section class="card">
      <div class="card-head">
        <div>
          <div class="card-title">高频攻击路径</div>
          <div class="card-sub">被高频访问的恶意接口路径</div>
        </div>
      </div>
      <table class="path-table" v-if="topPaths.length">
        <thead>
          <tr>
            <th>方法</th>
            <th>路径</th>
            <th class="num">命中次数</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="path in pagedTopPaths" :key="path.path">
            <td><span class="method-tag" :class="getMethodClass(path.path)">{{ getMethod(path.path) }}</span></td>
            <td class="path-cell" :title="path.path">{{ path.path }}</td>
            <td class="num"><strong>{{ path.count }}</strong></td>
          </tr>
        </tbody>
      </table>
      <div class="empty" v-else>暂无路径数据</div>
      <div class="mini-pager bottom" v-if="pathTotalPages > 1">
        <button :disabled="pathPage <= 1" @click="pathPage--">上一页</button>
        <span>{{ pathPage }} / {{ pathTotalPages }}</span>
        <button :disabled="pathPage >= pathTotalPages" @click="pathPage++">下一页</button>
      </div>
    </section>

    <!-- 最近攻击事件 -->
    <section class="card">
      <div class="card-head">
        <div>
          <div class="card-title">最近攻击事件</div>
          <div class="card-sub">最新拦截的安全威胁记录</div>
        </div>
      </div>
      <template v-if="recentLogs.length">
        <table class="event-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>站点</th>
              <th>来源 IP</th>
              <th>方法</th>
              <th>路径</th>
              <th>检测引擎</th>
              <th>
                <span class="th-help">
                  检测结果
                  <button class="help-btn" type="button" @click.stop="showResultHelp = !showResultHelp" aria-label="检测结果说明">?</button>
                </span>
              </th>
              <th>危险度</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in pagedRecentLogs" :key="log.id">
              <td class="mono time-cell">{{ fmt(log.timestamp) }}</td>
              <td class="site-cell">{{ siteName(log) }}</td>
              <td class="mono">{{ log.client_ip }}</td>
              <td><span class="method-tag" :class="getMethodClass(log.path)">{{ getMethod(log.path) }}</span></td>
              <td class="path-cell" :title="log.path">{{ log.path }}</td>
              <td>
                <span class="engine-badge" :class="log.source === 'ai' ? 'ai' : 'rule'">
                  {{ log.source === 'ai' ? 'AI' : '规则' }}
                </span>
              </td>
              <td class="result-cell" :title="log.rule_name">{{ log.rule_name }}</td>
              <td><span class="sev-pill" :class="log.severity">{{ sevTxt(log.severity) }}</span></td>
            </tr>
          </tbody>
        </table>
        <div class="help-box" v-if="showResultHelp">
          <button class="help-close" @click="showResultHelp = false">×</button>
          <strong>检测结果怎么看？</strong>
          <p>这里展示系统判断异常的主要原因。普通客户可以把它理解成"为什么被拦截"，再结合来源 IP、访问路径和危险度判断是否需要进一步处理。</p>
        </div>
      </template>
      <div class="empty" v-else-if="recentLoading">加载中...</div>
      <div class="empty" v-else>暂无攻击记录</div>
      <div class="mini-pager bottom" v-if="recentTotalPages > 1">
        <button :disabled="recentPage <= 1" @click="recentPage--">上一页</button>
        <span>{{ recentPage }} / {{ recentTotalPages }}</span>
        <button :disabled="recentPage >= recentTotalPages" @click="recentPage++">下一页</button>
      </div>
    </section>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import * as echarts from 'echarts'
import { DataAnalysis, WarningFilled, Location, TrendCharts } from '@element-plus/icons-vue'
import api from '../api'
import { useWebSocket } from '../composables/useWebSocket'

const stats = ref({})
const sites = ref([])
const selectedSite = ref('')
const regionPage = ref(1)
const pathPage = ref(1)
const recentPage = ref(1)
const showResultHelp = ref(false)
const pageSizeSmall = 5
const { messages: realtimeLogs, isConnected } = useWebSocket('/api/v1/logs/stream')

const chartRef = ref(null)
const chartReady = ref(false)
let chartInstance = null
let resizeObserver = null

// ── 统计卡片 ──
const blockRate = computed(() => {
  const total = stats.value.total_requests || 0
  if (!total) return '0%'
  return `${Math.round(((stats.value.blocked_count || 0) / total) * 100)}%`
})

const regionRank = computed(() => stats.value.top_regions || [])
const topPaths = computed(() => stats.value.top_attack_paths || [])
const recentLogs = ref([])
const recentLoading = ref(false)
const regionTotalPages = computed(() => Math.max(1, Math.ceil(regionRank.value.length / pageSizeSmall)))
const pathTotalPages = computed(() => Math.max(1, Math.ceil(topPaths.value.length / pageSizeSmall)))
const recentTotalPages = computed(() => Math.max(1, Math.ceil(recentLogs.value.length / pageSizeSmall)))
const regionPageOffset = computed(() => (regionPage.value - 1) * pageSizeSmall)
const pagedRegionRank = computed(() => regionRank.value.slice(regionPageOffset.value, regionPageOffset.value + pageSizeSmall))
const pagedTopPaths = computed(() => topPaths.value.slice((pathPage.value - 1) * pageSizeSmall, pathPage.value * pageSizeSmall))
const pagedRecentLogs = computed(() => recentLogs.value.slice((recentPage.value - 1) * pageSizeSmall, recentPage.value * pageSizeSmall))

const activeIPs = computed(() => {
  const seen = new Set()
  return realtimeLogs.value
    .filter((log) => !selectedSite.value || log.site_id === selectedSite.value)
    .map((log) => log.client_ip)
    .filter((ip) => ip && !seen.has(ip) && seen.add(ip))
    .slice(0, 8)
})

const statCards = computed(() => [
  {
    key: 'total', label: '总检测量', value: stats.value.total_requests || 0,
    desc: '近 24 小时请求命中统计', icon: DataAnalysis,
    color: 'blue', textColor: '', accent: '#3b82f6',
  },
  {
    key: 'blocked', label: '拦截次数', value: stats.value.blocked_count || 0,
    desc: '已执行阻断动作的事件', icon: WarningFilled,
    color: 'red', textColor: 'red', accent: '#ef4444',
  },
  {
    key: 'regions', label: '来源地区', value: regionRank.value.length || 0,
    desc: '有归属地记录的攻击来源', icon: Location,
    color: 'green', textColor: 'green', accent: '#22c55e',
  },
  {
    key: 'rate', label: '拦截占比', value: blockRate.value,
    desc: '拦截数 / 检测量', icon: TrendCharts,
    color: 'purple', textColor: 'purple', accent: '#8b5cf6',
  },
].map((c) => ({
  ...c,
  displayValue: typeof c.value === 'number' ? countUpMap.value[c.key] ?? '0' : c.value,
})))

// ── Count-up 动画 ──
const countUpMap = ref({})

function animateCount(key, target, duration = 400) {
  const start = parseInt(countUpMap.value[key]) || 0
  if (start === target) return
  const startTime = performance.now()
  function tick(now) {
    const elapsed = now - startTime
    const progress = Math.min(elapsed / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    countUpMap.value[key] = Math.round(start + (target - start) * eased).toLocaleString()
    if (progress < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

watch(
  () => [stats.value.total_requests, stats.value.blocked_count, regionRank.value.length, blockRate.value],
  () => {
    animateCount('total', stats.value.total_requests || 0)
    animateCount('blocked', stats.value.blocked_count || 0)
    animateCount('regions', regionRank.value.length || 0)
    animateCount('rate', parseInt(blockRate.value) || 0)
  },
  { immediate: true }
)

// ── 威胁等级 ──
const severityRows = computed(() => {
  const source = stats.value.by_severity || {}
  const rows = [
    { key: 'critical', label: '严重', icon: '🔴' },
    { key: 'high', label: '高危', icon: '🟠' },
    { key: 'medium', label: '中危', icon: '🟡' },
    { key: 'low', label: '低危', icon: '🟢' },
  ].map((row) => ({ ...row, count: source[row.key] || 0 }))
  const max = Math.max(...rows.map((r) => r.count), 1)
  return rows.map((row) => ({
    ...row,
    width: `${Math.max(4, Math.round((row.count / max) * 100))}%`,
  }))
})

// ── 攻击路径 ──
function getMethod(path) { return path?.method || 'GET' }
function getMethodClass(path) {
  const m = getMethod(path)
  if (m === 'POST') return 'post'
  if (m === 'PUT') return 'put'
  if (m === 'DELETE') return 'delete'
  return 'get'
}
function rankWidth(count) {
  const max = regionRank.value[0]?.count || 1
  return `${Math.max(6, Math.round((count / max) * 100))}%`
}

// ── ECharts 饼图 ──
const SEVERITY_COLORS = {
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#22c55e',
}

async function initChart() {
  await nextTick()
  if (!chartRef.value) return
  chartInstance = echarts.init(chartRef.value, null, { renderer: 'canvas' })
  updateChart()
  chartReady.value = true
  const ro = new ResizeObserver(() => chartInstance?.resize())
  ro.observe(chartRef.value)
  resizeObserver = ro
}

function updateChart() {
  if (!chartInstance) return
  const data = severityRows.value
    .filter(r => r.count > 0)
    .map(r => ({ name: r.label, value: r.count, itemStyle: { color: SEVERITY_COLORS[r.key] } }))
  const total = data.reduce((s, d) => s + d.value, 0)

  chartInstance.setOption({
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(15,23,42,0.92)',
      borderColor: 'rgba(99,102,241,0.2)',
      borderRadius: 10,
      padding: [10, 14],
      textStyle: { color: '#e2e8f0', fontSize: 13 },
      formatter: (p) => `<b>${p.name}</b><br/>${p.value} 次 (${p.percent}%)`,
    },
    series: [{
      type: 'pie',
      radius: ['46%', '72%'],
      center: ['50%', '48%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: {
        show: true, position: 'outside',
        formatter: '{b}\n{d}%', fontSize: 12, color: '#475569', lineHeight: 18,
      },
      labelLine: { length: 14, length2: 10, smooth: 0.3 },
      emphasis: { scaleSize: 6, label: { fontWeight: 700, fontSize: 13 } },
      data: data.length ? data : [{ name: '暂无数据', value: 1, itemStyle: { color: '#e2e8f0' }, label: { color: '#94a3b8' } }],
    }],
    graphic: total > 0 ? [{
      type: 'group', left: 'center', top: '46%',
      children: [
        { type: 'text', style: { text: String(total), fontSize: 26, fontWeight: 800, fill: '#0f172a', textAlign: 'center' }, left: 'center', top: -12 },
        { type: 'text', style: { text: '威胁事件', fontSize: 12, fill: '#94a3b8', textAlign: 'center' }, left: 'center', top: 14 },
      ],
    }] : [],
  }, true)
}

watch(severityRows, () => updateChart(), { deep: true })

// ── 数据加载 ──
async function loadStats() {
  try { stats.value = await api.get('/stats', { params: siteParams() }) || {} } catch {}
}

async function loadRecent() {
  recentLoading.value = true
  try {
    const res = await api.get('/logs', { params: { page: 1, limit: 30, ...siteParams() } })
    recentLogs.value = res.data || []
  } catch {} finally { recentLoading.value = false }
}

async function loadSites() {
  try { sites.value = await api.get('/sites') || [] } catch {}
}

function siteParams() {
  return selectedSite.value ? { site_id: selectedSite.value } : {}
}

function reloadBySite() {
  regionPage.value = 1
  pathPage.value = 1
  recentPage.value = 1
  loadStats()
  loadRecent()
}

function siteName(log) {
  return log?.site_name || log?.domain || '默认站点'
}

function sevTxt(s) { return { critical:'严重', high:'高危', medium:'中危', low:'低危' }[s] || s }
function fmt(ts) { return ts ? new Date(ts).toLocaleString('zh-CN') : '-' }

onMounted(() => { loadSites(); loadStats(); initChart(); loadRecent() })
onBeforeUnmount(() => { resizeObserver?.disconnect(); chartInstance?.dispose(); chartInstance = null })
</script>

<style scoped>
.dashboard { display: flex; flex-direction: column; gap: 16px; }
.scope-toolbar {
  display: flex; justify-content: space-between; align-items: center; gap: 16px;
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  padding: 16px 18px; box-shadow: 0 1px 3px rgba(0,0,0,.04);
}
.scope-title { font-size: 18px; font-weight: 800; color: var(--text-primary); }
.scope-desc { margin-top: 3px; font-size: 12.5px; color: var(--text-secondary); }
.scope-control { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.scope-control label { font-size: 12px; color: var(--text-secondary); font-weight: 700; }
.scope-select {
  min-width: 240px; height: 38px; padding: 0 12px; border-radius: var(--radius-input);
  border: 1px solid #cbd5e1; background: #fff; color: var(--text-primary);
  font-size: 13px; font-weight: 600; outline: none;
}
.scope-select:focus { border-color: var(--primary); box-shadow: 0 0 0 3px rgba(99,102,241,.12); }

/* Stats Cards */
.stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
.stat-card {
  background: var(--bg-card); border-radius: var(--radius-card); padding: 18px 18px 14px;
  border: 1px solid var(--border); box-shadow: 0 1px 3px rgba(0,0,0,.04);
  position: relative; overflow: hidden;
  animation: cardIn .5s ease both;
  transition: transform .25s, box-shadow .25s;
}
.stat-card::before {
  content: ''; position: absolute; top: 0; left: 0; right: 0; height: 3px;
  background: var(--accent); opacity: .7; border-radius: var(--radius-card) var(--radius-card) 0 0;
}
.stat-card:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0,0,0,.07); }
@keyframes cardIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }

.stat-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.stat-label { font-size: 13px; color: var(--text-secondary); font-weight: 600; }
.stat-badge {
  width: 34px; height: 34px; border-radius: 9px;
  display: flex; align-items: center; justify-content: center;
}
.stat-badge.blue { background: #eff6ff; color: #2563eb; }
.stat-badge.red { background: #fef2f2; color: #dc2626; }
.stat-badge.green { background: #f0fdf4; color: #16a34a; }
.stat-badge.purple { background: #f5f3ff; color: #7c3aed; }

.stat-value { font-size: 28px; font-weight: 800; color: var(--text-primary); line-height: 1; letter-spacing: -.5px; }
.stat-value.red { color: #dc2626; }
.stat-value.green { color: #16a34a; }
.stat-value.purple { color: #7c3aed; }
.stat-footer { margin-top: 8px; font-size: 12px; color: var(--text-muted); }

/* Two column */
.mid-grid { display: grid; grid-template-columns: minmax(0, .95fr) minmax(0, 1.05fr); gap: 16px; }

/* Live badge */
.live-badge {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 12px; border-radius: 999px;
  background: var(--border-light); color: var(--text-secondary); font-size: 12px; font-weight: 700;
}
.live-badge.on { background: #dcfce7; color: #15803d; }
.live-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.live-badge.on .live-dot { animation: pulse 1.5s ease-in-out infinite; }
@keyframes pulse { 0%,100% { opacity:1; transform:scale(1); } 50% { opacity:.5; transform:scale(1.3); } }

/* Chart */
.chart-wrap { position: relative; height: 220px; }
.echart { width: 100%; height: 100%; }
.chart-loading {
  position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;
  background: #fafbfd;
}
.spinner {
  width: 26px; height: 26px; border: 2.5px solid var(--border);
  border-top-color: var(--primary); border-radius: 50%; animation: spin .8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* Severity bar */
.sev-bar {
  display: flex; gap: 2px; padding: 12px 16px; border-top: 1px solid var(--border-light);
}
.sev-item {
  flex: 1; display: flex; align-items: center; gap: 6px;
  padding: 8px 10px; border-radius: 8px; background: #fafbfd;
  border: 1px solid var(--border-light); font-size: 12px; color: var(--text-secondary);
}
.sev-item:hover { background: var(--border-light); }
.sev-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.sev-dot.critical { background: #ef4444; }
.sev-dot.high { background: #f97316; }
.sev-dot.medium { background: #eab308; }
.sev-dot.low { background: #22c55e; }
.sev-item strong { margin-left: auto; color: var(--text-primary); font-weight: 700; }

/* Ranking */
.rank-body { padding: 6px 16px; }
.rank-row {
  display: grid; grid-template-columns: 26px minmax(0, 1fr) 44px; align-items: center;
  gap: 8px; padding: 9px 4px; border-bottom: 1px solid var(--border-light);
  border-radius: 6px; transition: background .15s;
}
.rank-row:last-child { border-bottom: none; }
.rank-row:hover { background: var(--bg-hover); }
.rank-idx {
  width: 22px; height: 22px; border-radius: 6px;
  display: inline-flex; align-items: center; justify-content: center;
  background: var(--border-light); color: var(--text-secondary); font-size: 11px; font-weight: 800;
}
.rank-idx.hot { background: #fef2f2; color: #dc2626; }
.rank-info { min-width: 0; }
.rank-name {
  display: block; font-size: 13px; color: var(--text-primary); font-weight: 600;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.rank-bar { height: 5px; margin-top: 4px; border-radius: 999px; background: var(--border-light); overflow: hidden; }
.rank-bar span { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg, #2563eb, #ef4444); transition: width .6s; }
.rank-cnt { font-size: 13px; color: var(--text-primary); text-align: right; }
.mini-pager {
  display: flex; justify-content: flex-end; align-items: center; gap: 8px;
  padding: 10px 16px; border-top: 1px solid var(--border-light); color: var(--text-secondary); font-size: 12px;
}
.mini-pager.bottom { border-top: 1px solid var(--border-light); }
.mini-pager button {
  padding: 4px 9px; border-radius: 6px; border: 1px solid var(--border);
  background: #fff; color: var(--text-secondary); font-size: 12px; cursor: pointer;
}
.mini-pager button:disabled { opacity: .45; cursor: not-allowed; }

/* Active IPs */
.ip-bar {
  display: flex; flex-wrap: wrap; align-items: center; gap: 6px;
  padding: 12px 16px; border-top: 1px solid var(--border-light);
}
.ip-label { font-size: 11px; color: var(--text-muted); font-weight: 600; margin-right: 4px; }
.ip-tag {
  padding: 4px 9px; border-radius: 7px; background: var(--bg-hover); border: 1px solid var(--border);
  color: #334155; font-family: var(--font-mono); font-size: 11.5px;
}
.ip-tag:hover { border-color: #93c5fd; }

/* Path table */
.path-table { width: 100%; border-collapse: collapse; }
.path-table th {
  text-align: left; padding: 9px 16px; font-size: 11px; font-weight: 600;
  color: var(--text-muted); text-transform: uppercase; letter-spacing: .5px;
  border-bottom: 1px solid var(--border-light); background: var(--bg-subtle);
}
.path-table td {
  padding: 10px 16px; font-size: 13px; color: #334155;
  border-bottom: 1px solid var(--border-light);
}
.path-table tr:last-child td { border-bottom: none; }
.path-table tr:hover td { background: var(--bg-hover); }
.path-table .num { text-align: right; }
.path-table .num strong { color: var(--text-primary); }

.empty { color: var(--text-muted); font-size: 13px; text-align: center; padding: 32px 16px; }

/* Event table overrides */
.event-table { width: 100%; border-collapse: collapse; }
.event-table th {
  text-align: left; padding: 9px 14px; font-size: 11px; font-weight: 600;
  color: var(--text-muted); text-transform: uppercase; letter-spacing: .5px;
  border-bottom: 1px solid var(--border-light); background: var(--bg-subtle);
}
.event-table td {
  padding: 10px 14px; font-size: 13px; color: #334155;
  border-bottom: 1px solid var(--border-light);
}
.event-table tr:last-child td { border-bottom: none; }
.event-table tr:hover td { background: var(--bg-hover); }
.th-help { display: inline-flex; align-items: center; gap: 6px; }
.help-btn {
  width: 18px; height: 18px; border-radius: 999px; border: 1px solid #cbd5e1;
  background: #fff; color: var(--text-secondary); font-size: 12px; line-height: 1; font-weight: 800;
  cursor: pointer;
}
.help-btn:hover { border-color: var(--primary); color: var(--primary-hover); background: var(--primary-light); }
.help-box {
  position: relative; margin: 12px 16px 16px; padding: 16px 18px;
  background: #fff; border: 1px solid #dbe3ef; border-radius: var(--radius-card);
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.10);
  color: var(--text-secondary); font-size: 13px; line-height: 1.7;
}
.help-box strong { display: block; color: var(--text-primary); margin-bottom: 6px; font-size: 14px; }
.help-close {
  position: absolute; top: 8px; right: 10px; border: none; background: transparent;
  color: var(--text-muted); font-size: 18px; cursor: pointer;
}
.time-cell { white-space: nowrap; }
.site-cell {
  max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  color: var(--text-secondary); font-weight: 700;
}
.result-cell { max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12.5px; color: var(--text-secondary); }

/* Responsive */
@media (max-width: 1024px) {
  .mid-grid { grid-template-columns: 1fr; }
  .chart-wrap { height: 240px; }
}
@media (max-width: 768px) {
  .scope-toolbar { flex-direction: column; align-items: stretch; }
  .scope-control { align-items: stretch; flex-direction: column; gap: 6px; }
  .scope-select { min-width: 0; width: 100%; }
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .sev-bar { flex-wrap: wrap; }
  .sev-item { flex: none; width: calc(50% - 2px); }
  .path-table, .event-table { font-size: 12px; }
  .card { overflow-x: auto; }
  .path-cell { max-width: 200px; }
  .event-table { min-width: 700px; }
}
</style>
