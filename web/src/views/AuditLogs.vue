<template>
  <div class="audit-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <div class="page-icon cyan">
          <el-icon :size="20"><List /></el-icon>
        </div>
        <div>
          <h1 class="page-title">审计日志</h1>
          <p class="page-desc">系统操作记录追踪，保障运维安全可溯</p>
        </div>
      </div>
      <div class="header-actions">
        <button class="btn-outline" @click="loadLogs" :disabled="loading">
          <el-icon :size="14"><RefreshRight /></el-icon>
          {{ loading ? '加载中' : '刷新' }}
        </button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon indigo">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
        </div>
        <div class="stat-body">
          <span class="stat-value">{{ total }}</span>
          <span class="stat-label">总记录</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon emerald">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        </div>
        <div class="stat-body">
          <span class="stat-value">{{ successTotal }}</span>
          <span class="stat-label">成功操作</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon rose">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        </div>
        <div class="stat-body">
          <span class="stat-value">{{ failureTotal }}</span>
          <span class="stat-label">失败操作</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon amber">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
        </div>
        <div class="stat-body">
          <span class="stat-value">{{ uniqueActors }}</span>
          <span class="stat-label">操作用户</span>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-group">
        <label>操作类型</label>
        <select v-model="filterAction" class="filter-select" @change="page = 1; loadLogs()">
          <option value="">全部操作</option>
          <option value="login">登录</option>
          <option value="logout">登出</option>
          <option value="create_rule">创建规则</option>
          <option value="update_rule">更新规则</option>
          <option value="delete_rule">删除规则</option>
          <option value="add_ip">添加 IP</option>
          <option value="remove_ip">移除 IP</option>
          <option value="add_geo">添加地理规则</option>
          <option value="remove_geo">移除地理规则</option>
          <option value="update_settings">更新设置</option>
          <option value="activate_license">激活授权</option>
          <option value="create_user">创建用户</option>
          <option value="delete_user">删除用户</option>
          <option value="backup_export">备份导出</option>
          <option value="backup_import">备份导入</option>
          <option value="config_reload">配置重载</option>
        </select>
      </div>
      <div class="filter-group">
        <label>操作结果</label>
        <select v-model="filterStatus" class="filter-select" @change="page = 1; loadLogs()">
          <option value="">全部状态</option>
          <option value="success">成功</option>
          <option value="failure">失败</option>
        </select>
      </div>
      <div class="filter-group">
        <label>操作者</label>
        <input v-model="filterActor" class="filter-input" placeholder="搜索用户名..." @keyup.enter="page = 1; loadLogs()" />
      </div>
      <div class="filter-group filter-actions">
        <button class="filter-btn" @click="resetFilters">
          <el-icon :size="14"><RefreshRight /></el-icon>
          重置
        </button>
      </div>
    </div>

    <!-- 日志列表 -->
    <div class="log-list">
      <div class="log-card" v-for="log in logs" :key="log.id">
        <div class="log-timeline">
          <div class="timeline-dot" :class="log.status"></div>
          <div class="timeline-line"></div>
        </div>
        <div class="log-content">
          <div class="log-header">
            <div class="log-action-info">
              <span class="action-badge" :class="getActionClass(log.action)">
                {{ actionLabel(log.action) }}
              </span>
              <span class="status-badge" :class="log.status">
                {{ log.status === 'success' ? '成功' : '失败' }}
              </span>
            </div>
            <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          </div>
          <div class="log-meta">
            <span class="meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="13" height="13"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              {{ log.actor }}
            </span>
            <span class="meta-item ip mono">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="13" height="13"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
              {{ log.client_ip }}
            </span>
          </div>
          <div class="log-detail" v-if="log.detail">{{ log.detail }}</div>
        </div>
      </div>

      <div class="empty-state" v-if="logs.length === 0 && !loading">
        <div class="empty-visual">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" width="48" height="48"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
        </div>
        <div class="empty-text">暂无审计记录</div>
        <div class="empty-desc">系统操作将自动记录在此，支持按操作类型、结果和操作者筛选</div>
      </div>
    </div>

    <!-- 分页 -->
    <div class="pagination-bar" v-if="total > 0">
      <span class="page-info">共 {{ total }} 条记录，第 {{ page }} / {{ totalPages }} 页</span>
      <div class="page-controls">
        <button class="page-btn" :disabled="page <= 1" @click="page = 1; loadLogs()">
          首页
        </button>
        <button class="page-btn" :disabled="page <= 1" @click="page--; loadLogs()">
          上一页
        </button>
        <span class="page-current">{{ page }}</span>
        <button class="page-btn" :disabled="page >= totalPages" @click="page++; loadLogs()">
          下一页
        </button>
        <button class="page-btn" :disabled="page >= totalPages" @click="page = totalPages; loadLogs()">
          末页
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { List, RefreshRight } from '@element-plus/icons-vue'
import api from '../api'

const logs = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const filterAction = ref('')
const filterStatus = ref('')
const filterActor = ref('')

const pageSize = 20
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const successTotal = computed(() => logs.value.filter(l => l.status === 'success').length)
const failureTotal = computed(() => logs.value.filter(l => l.status !== 'success').length)
const uniqueActors = computed(() => new Set(logs.value.map(l => l.actor)).size)

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.toLocaleString('zh-CN')
}

function actionLabel(action) {
  const map = {
    login: '用户登录', logout: '用户登出',
    create_rule: '创建检测规则', update_rule: '更新检测规则', delete_rule: '删除检测规则',
    add_ip: '添加 IP 记录', remove_ip: '移除 IP 记录',
    add_geo: '添加地理封锁规则', remove_geo: '移除地理封锁规则',
    update_settings: '更新系统设置', activate_license: '激活授权',
    create_user: '创建用户', delete_user: '删除用户',
    backup_export: '备份导出', backup_import: '备份导入',
    config_reload: '配置重载',
  }
  return map[action] || action
}

function getActionClass(action) {
  if (['login', 'logout'].includes(action)) return 'auth'
  if (['create_rule', 'update_rule', 'delete_rule'].includes(action)) return 'rule'
  if (['add_ip', 'remove_ip'].includes(action)) return 'ip'
  if (['add_geo', 'remove_geo'].includes(action)) return 'geo'
  if (['create_user', 'delete_user'].includes(action)) return 'user'
  if (['backup_export', 'backup_import'].includes(action)) return 'backup'
  return 'system'
}

function resetFilters() {
  filterAction.value = ''
  filterStatus.value = ''
  filterActor.value = ''
  page.value = 1
  loadLogs()
}

async function loadLogs() {
  loading.value = true
  try {
    const params = { page: page.value, limit: pageSize }
    if (filterAction.value) params.action = filterAction.value
    if (filterStatus.value) params.status = filterStatus.value
    if (filterActor.value) params.actor = filterActor.value
    const res = await api.get('/audit/events', { params })
    logs.value = res?.data || []
    total.value = res?.total || 0
  } catch {
    logs.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

onMounted(loadLogs)
</script>

<style scoped>
.audit-page {
  max-width: 1200px;
}

/* ===== 页面头部 ===== */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}
.page-icon {
  width: 42px; height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.page-icon.cyan { background: #ecfeff; color: #0891b2; }
.page-title {
  font-size: 20px;
  font-weight: 800;
  color: #0f172a;
  margin: 0;
  letter-spacing: -0.3px;
}
.page-desc {
  font-size: 13px;
  color: #94a3b8;
  margin: 2px 0 0;
}
.btn-outline {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
  background: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  color: #475569;
  transition: all 0.2s;
}
.btn-outline:hover { border-color: #6366f1; color: #6366f1; background: #fafaff; }
.btn-outline:disabled { opacity: 0.5; cursor: not-allowed; }

/* ===== 统计卡片 ===== */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-bottom: 20px;
}
.stat-card {
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  padding: 18px 20px;
  display: flex;
  align-items: center;
  gap: 14px;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
}
.stat-icon {
  width: 42px; height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.stat-icon.indigo { background: #eef2ff; color: #6366f1; }
.stat-icon.emerald { background: #ecfdf5; color: #10b981; }
.stat-icon.rose { background: #fff1f2; color: #ef4444; }
.stat-icon.amber { background: #fffbeb; color: #d97706; }
.stat-body { display: flex; flex-direction: column; }
.stat-value { font-size: 22px; font-weight: 800; color: #0f172a; line-height: 1.2; }
.stat-label { font-size: 12px; color: #94a3b8; margin-top: 2px; }

/* ===== 筛选栏 ===== */
.filter-bar {
  display: flex;
  gap: 12px;
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  padding: 16px 20px;
  margin-bottom: 16px;
  align-items: flex-end;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
}
.filter-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}
.filter-group label {
  font-size: 11px;
  font-weight: 700;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}
.filter-select, .filter-input {
  padding: 9px 12px;
  border-radius: 9px;
  border: 1px solid #e2e8f0;
  background: #f8f9fc;
  font-size: 13px;
  color: #0f172a;
  outline: none;
  transition: all 0.2s;
}
.filter-select:focus, .filter-input:focus {
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99,102,241,0.08);
  background: #fff;
}
.filter-input::placeholder { color: #94a3b8; }
.filter-actions { flex: 0 0 auto; }
.filter-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 9px 16px;
  border-radius: 9px;
  border: 1px solid #e2e8f0;
  background: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  color: #64748b;
  transition: all 0.2s;
}
.filter-btn:hover { border-color: #6366f1; color: #6366f1; }

/* ===== 日志列表 ===== */
.log-list {
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
}
.log-card {
  display: flex;
  padding: 0;
  transition: background 0.15s;
}
.log-card:hover { background: #f8f9fc; }
.log-card:last-child .timeline-line { display: none; }

.log-timeline {
  width: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 18px;
  flex-shrink: 0;
}
.timeline-dot {
  width: 10px; height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
  z-index: 1;
}
.timeline-dot.success { background: #22c55e; box-shadow: 0 0 6px rgba(34,197,94,.4); }
.timeline-dot.failure { background: #ef4444; box-shadow: 0 0 6px rgba(239,68,68,.4); }
.timeline-line {
  width: 2px;
  flex: 1;
  background: #eef0f4;
  margin-top: 6px;
}

.log-content {
  flex: 1;
  padding: 16px 20px 16px 0;
  border-bottom: 1px solid #f1f5f9;
  min-width: 0;
}
.log-card:last-child .log-content { border-bottom: none; }

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.log-action-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.action-badge {
  font-size: 13px;
  font-weight: 700;
  color: #0f172a;
  padding: 3px 10px;
  border-radius: 7px;
}
.action-badge.auth { background: #eef2ff; color: #6366f1; }
.action-badge.rule { background: #fffbeb; color: #d97706; }
.action-badge.ip { background: #ecfeff; color: #0891b2; }
.action-badge.geo { background: #f5f3ff; color: #7c3aed; }
.action-badge.user { background: #fdf2f8; color: #db2777; }
.action-badge.backup { background: #ecfdf5; color: #10b981; }
.action-badge.system { background: #f1f5f9; color: #475569; }

.status-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 5px;
}
.status-badge.success { background: #ecfdf5; color: #059669; }
.status-badge.failure { background: #fef2f2; color: #dc2626; }

.log-time {
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
}

.log-meta {
  display: flex;
  gap: 16px;
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}
.meta-item.ip {
  font-family: 'SF Mono', 'Menlo', monospace;
  font-size: 11.5px;
}

.log-detail {
  margin-top: 8px;
  font-size: 12.5px;
  color: #64748b;
  line-height: 1.5;
  word-break: break-all;
  padding: 8px 12px;
  background: #f8f9fc;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
}

/* ===== 空状态 ===== */
.empty-state {
  text-align: center;
  padding: 60px 20px;
}
.empty-visual {
  color: #cbd5e1;
  margin-bottom: 16px;
}
.empty-text {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 6px;
}
.empty-desc {
  font-size: 13px;
  color: #94a3b8;
  max-width: 360px;
  margin: 0 auto;
  line-height: 1.6;
}

/* ===== 分页 ===== */
.pagination-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding: 14px 20px;
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
}
.page-info {
  font-size: 13px;
  color: #64748b;
}
.page-controls {
  display: flex;
  align-items: center;
  gap: 6px;
}
.page-btn {
  padding: 6px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  color: #475569;
  transition: all 0.2s;
}
.page-btn:hover:not(:disabled) { border-color: #6366f1; color: #6366f1; }
.page-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.page-current {
  padding: 6px 14px;
  background: #6366f1;
  color: #fff;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
}

/* ===== 响应式 ===== */
@media (max-width: 1024px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 768px) {
  .stats-row { grid-template-columns: 1fr 1fr; }
  .filter-bar { flex-direction: column; gap: 10px; }
  .filter-actions { align-self: flex-start; }
  .log-timeline { width: 36px; }
  .log-header { flex-direction: column; align-items: flex-start; gap: 6px; }
  .pagination-bar { flex-direction: column; gap: 10px; align-items: flex-start; }
}
</style>
