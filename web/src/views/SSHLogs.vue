<template>
  <div class="ssh-page">
    <!-- 页面标题 -->
    <div class="page-toolbar">
      <div class="heading-group">
        <div class="heading-icon amber"><el-icon :size="18"><Key /></el-icon></div>
        <div>
          <div class="page-heading">SSH 监控</div>
          <div class="page-sub">暴力破解检测与防护</div>
        </div>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon blue">
          <el-icon :size="20"><List /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.total || 0 }}</div>
          <div class="stat-label">总事件数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon amber">
          <el-icon :size="20"><Warning /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.failed || 0 }}</div>
          <div class="stat-label">登录失败</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon rose">
          <el-icon :size="20"><CircleClose /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.blocked || 0 }}</div>
          <div class="stat-label">已封禁</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon emerald">
          <el-icon :size="20"><CircleCheck /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ successCount }}</div>
          <div class="stat-label">登录成功</div>
        </div>
      </div>
    </div>

    <!-- 高频攻击者 -->
    <div class="attackers-card" v-if="stats.top_attackers?.length">
      <div class="card-header">
        <span class="card-title">高频攻击 IP</span>
      </div>
      <div class="attacker-list">
        <div class="attacker-row" v-for="(a, i) in stats.top_attackers" :key="a.ip">
          <span class="attacker-rank" :class="{ top: i < 3 }">{{ i + 1 }}</span>
          <span class="attacker-ip mono">{{ a.ip }}</span>
          <span class="attacker-region">{{ a.region || '未知' }}</span>
          <div class="attacker-bar-wrap">
            <div class="attacker-bar" :style="{ width: barWidth(a.count) }"></div>
          </div>
          <span class="attacker-count">{{ a.count }} 次</span>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-group">
        <div class="filter-item">
          <label>来源 IP</label>
          <input v-model="filterIP" placeholder="输入 IP 地址" class="filter-input" @keyup.enter="doSearch" />
        </div>
        <div class="filter-item">
          <label>事件类型</label>
          <select v-model="filterType" class="filter-input" @change="doSearch">
            <option value="">全部</option>
            <option value="failed">登录失败</option>
            <option value="blocked">已封禁</option>
            <option value="success">登录成功</option>
          </select>
        </div>
        <div class="filter-item">
          <label>用户名</label>
          <input v-model="filterUser" placeholder="输入用户名" class="filter-input" @keyup.enter="doSearch" />
        </div>
      </div>
      <div class="filter-actions">
        <button class="btn-primary" @click="doSearch">
          <el-icon :size="14"><Search /></el-icon> 查询
        </button>
        <button class="btn-ghost" @click="resetFilter">重置</button>
      </div>
    </div>

    <!-- 事件表 -->
    <div class="table-card">
      <div class="table-header">
        <span class="table-title">SSH 事件日志</span>
        <span class="table-count">{{ total }} 条记录</span>
      </div>
      <table class="data-table">
        <thead>
          <tr>
            <th>时间</th>
            <th>来源 IP</th>
            <th>地区</th>
            <th>用户名</th>
            <th>事件类型</th>
            <th>详情</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ev in events" :key="ev.id">
            <td class="mono time-cell">{{ fmt(ev.timestamp) }}</td>
            <td class="mono">{{ ev.client_ip }}</td>
            <td class="region-cell">{{ ev.region || '-' }}</td>
            <td class="mono">{{ ev.username || '-' }}</td>
            <td>
              <span class="event-badge" :class="ev.event_type">
                {{ eventTypeText(ev.event_type) }}
              </span>
            </td>
            <td class="msg-cell" :title="ev.message">{{ ev.message }}</td>
          </tr>
          <tr v-if="events.length === 0 && !loading">
            <td colspan="6" class="empty-state">
              <div class="empty-icon">
                <el-icon :size="32"><Monitor /></el-icon>
              </div>
              <div class="empty-text">暂无 SSH 事件</div>
              <div class="empty-desc">SSH 监控未启用或暂无记录</div>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="pagination-bar" v-if="total > 0">
        <span class="page-info">共 {{ total }} 条 / {{ totalPages }} 页</span>
        <div class="page-controls">
          <select v-model="pageSize" class="page-size-select" @change="page=1; loadEvents()">
            <option :value="20">20 条/页</option>
            <option :value="50">50 条/页</option>
            <option :value="100">100 条/页</option>
          </select>
          <button class="page-btn" :disabled="page <= 1" @click="page=1; loadEvents()">首页</button>
          <button class="page-btn" :disabled="page <= 1" @click="page--; loadEvents()">上一页</button>
          <input type="number" v-model.number="jumpPage" class="page-jump" min="1" :max="totalPages" @keyup.enter="doJump" />
          <button class="page-btn" @click="doJump">跳转</button>
          <button class="page-btn" :disabled="page >= totalPages" @click="page++; loadEvents()">下一页</button>
          <button class="page-btn" :disabled="page >= totalPages" @click="page=totalPages; loadEvents()">末页</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Search, List, Warning, CircleClose, CircleCheck, Monitor, Key } from '@element-plus/icons-vue'
import api from '../api'

const events = ref([]), total = ref(0), page = ref(1), pageSize = ref(20)
const loading = ref(false), filterIP = ref(''), filterType = ref(''), filterUser = ref('')
const jumpPage = ref(1)
const stats = ref({ total: 0, failed: 0, blocked: 0, top_attackers: [] })
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

const successCount = computed(() => Math.max(0, (stats.value.total || 0) - (stats.value.failed || 0) - (stats.value.blocked || 0)))

function eventTypeText(t) {
  return { failed: '登录失败', blocked: '已封禁', success: '登录成功' }[t] || t
}
function fmt(ts) { return ts ? new Date(ts).toLocaleString('zh-CN') : '-' }
function barWidth(count) {
  const max = stats.value.top_attackers?.[0]?.count || 1
  return Math.max(8, (count / max) * 100) + '%'
}

async function loadStats() {
  try {
    stats.value = await api.get('/ssh/stats')
  } catch {}
}

async function loadEvents() {
  loading.value = true
  try {
    const params = { page: page.value, limit: pageSize.value }
    if (filterIP.value) params.client_ip = filterIP.value
    if (filterType.value) params.event_type = filterType.value
    if (filterUser.value) params.username = filterUser.value
    const res = await api.get('/ssh/events', { params })
    events.value = res.data || []
    total.value = res.total || 0
  } catch {} finally { loading.value = false }
}

function doSearch() {
  page.value = 1
  loadEvents()
}

function resetFilter() {
  filterIP.value = ''
  filterType.value = ''
  filterUser.value = ''
  page.value = 1
  loadEvents()
}

function doJump() {
  const p = Math.max(1, Math.min(jumpPage.value, totalPages.value))
  page.value = p
  jumpPage.value = p
  loadEvents()
}

onMounted(() => { loadStats(); loadEvents() })
</script>

<style scoped>
.ssh-page { display: flex; flex-direction: column; gap: 16px; }
.page-toolbar { display: flex; justify-content: space-between; align-items: center; }

/* Stats Grid */
.stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
.stat-card {
  background: var(--bg-card); border-radius: var(--radius-card); border: 1px solid var(--border);
  padding: 18px 20px; display: flex; align-items: center; gap: 14px;
}
.stat-icon {
  width: 44px; height: 44px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-icon.blue { background: var(--primary-light); color: var(--primary); }
.stat-icon.amber { background: #fffbeb; color: #d97706; }
.stat-icon.rose { background: #fff1f2; color: #e11d48; }
.stat-icon.emerald { background: #ecfdf5; color: #059669; }
.stat-value { font-size: 22px; font-weight: 800; color: var(--text-primary); }
.stat-label { font-size: 12px; color: var(--text-muted); margin-top: 2px; }

/* Attackers Card */
.attackers-card {
  background: var(--bg-card); border-radius: var(--radius-card); border: 1px solid var(--border);
  overflow: hidden;
}
.card-header { padding: var(--card-pad); border-bottom: 1px solid var(--border-light); }
.card-title { font-size: 14.5px; font-weight: 700; color: var(--text-primary); }
.attacker-list { padding: 8px 20px; }
.attacker-row {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 0; border-bottom: 1px solid var(--border-light);
}
.attacker-row:last-child { border-bottom: none; }
.attacker-rank {
  width: 24px; height: 24px; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 700; color: var(--text-muted); background: var(--border-light);
}
.attacker-rank.top { background: #fef2f2; color: #dc2626; }
.attacker-ip { font-size: 13px; color: #1e293b; min-width: 130px; }
.attacker-region { font-size: 12px; color: var(--text-secondary); min-width: 100px; }
.attacker-bar-wrap {
  flex: 1; height: 6px; background: var(--border-light); border-radius: 3px; overflow: hidden;
}
.attacker-bar { height: 100%; background: linear-gradient(90deg, #e11d48, #f43f5e); border-radius: 3px; transition: width 0.5s ease; }
.attacker-count { font-size: 12px; font-weight: 600; color: var(--danger); min-width: 60px; text-align: right; }

/* Filter Bar */
.filter-bar {
  background: var(--bg-card); border-radius: var(--radius-card); border: 1px solid var(--border);
  padding: 18px 20px;
  display: flex; justify-content: space-between; align-items: flex-end;
}
.filter-group { display: flex; gap: 16px; }
.filter-item { display: flex; flex-direction: column; gap: 5px; }
.filter-item label { font-size: 11.5px; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.3px; }
.filter-input { width: 180px; }
.filter-actions { display: flex; gap: 8px; }

/* Table overrides */
.table-header { display: flex; justify-content: space-between; align-items: center; padding: var(--card-pad); border-bottom: 1px solid var(--border-light); }
.table-title { font-size: 14.5px; font-weight: 700; color: var(--text-primary); }
.table-count { font-size: 12px; color: var(--text-muted); background: var(--border-light); padding: 3px 10px; border-radius: 10px; }
.time-cell { white-space: nowrap; }
.region-cell { max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-secondary); font-size: 12.5px; }
.msg-cell { max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12.5px; color: var(--text-secondary); }

.event-badge {
  display: inline-block; padding: 3px 10px; border-radius: 6px;
  font-size: 11.5px; font-weight: 600;
}
.event-badge.failed { background: #fffbeb; color: #d97706; }
.event-badge.blocked { background: #fef2f2; color: var(--danger); }
.event-badge.success { background: #ecfdf5; color: #059669; }

/* Pagination override */
.pagination-bar { display: flex; justify-content: space-between; align-items: center; padding: 12px 20px; border-top: 1px solid var(--border-light); }

@media (max-width: 768px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .attacker-row { flex-wrap: wrap; }
  .attacker-ip { min-width: auto; flex: 1; }
  .attacker-region { min-width: auto; }
  .filter-bar { flex-direction: column; }
  .filter-input { width: 100%; }
  .filter-actions { width: 100%; }
  .filter-actions .btn-primary { flex: 1; }
  .table-card { overflow-x: auto; }
  .data-table { min-width: 700px; }
  .pagination-bar { flex-direction: column; gap: 8px; }
}
</style>
