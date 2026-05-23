<template>
  <div class="iplist-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <div class="page-icon cyan">
          <el-icon :size="20"><Filter /></el-icon>
        </div>
        <div>
          <h1 class="page-title">访问控制</h1>
          <p class="page-desc">管理 IP 黑白名单，精确控制网络访问权限</p>
        </div>
      </div>
      <button class="btn-outline" @click="loadIPs">
        <el-icon :size="14"><RefreshRight /></el-icon>
        刷新
      </button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card" @click="activeTab = 'blacklist'; loadIPs()" :class="{ active: activeTab === 'blacklist' }">
        <div class="stat-icon rose">
          <el-icon :size="20"><CircleClose /></el-icon>
        </div>
        <div class="stat-body">
          <span class="stat-value">{{ blacklistCount }}</span>
          <span class="stat-label">黑名单 IP</span>
        </div>
        <div class="stat-arrow" v-if="activeTab === 'blacklist'">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="16" height="16"><polyline points="6 9 12 15 18 9"/></svg>
        </div>
      </div>
      <div class="stat-card" @click="activeTab = 'whitelist'; loadIPs()" :class="{ active: activeTab === 'whitelist' }">
        <div class="stat-icon emerald">
          <el-icon :size="20"><CircleCheck /></el-icon>
        </div>
        <div class="stat-body">
          <span class="stat-value">{{ whitelistCount }}</span>
          <span class="stat-label">白名单 IP</span>
        </div>
        <div class="stat-arrow" v-if="activeTab === 'whitelist'">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="16" height="16"><polyline points="6 9 12 15 18 9"/></svg>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon indigo">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/></svg>
        </div>
        <div class="stat-body">
          <span class="stat-value">{{ entries.length }}</span>
          <span class="stat-label">当前列表</span>
        </div>
      </div>
    </div>

    <!-- 添加表单 -->
    <div class="add-panel">
      <div class="add-header">
        <h3>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
          添加到{{ activeTab === 'blacklist' ? '黑名单' : '白名单' }}
        </h3>
      </div>
      <div class="add-form">
        <div class="add-field">
          <label>IP 地址</label>
          <input v-model="newIP" placeholder="例如: 192.168.1.100 或 10.0.0.0/24" class="add-input" @keyup.enter="addIP" />
        </div>
        <div class="add-field">
          <label>备注</label>
          <input v-model="newNote" placeholder="可选，描述此 IP 的来源或用途" class="add-input" @keyup.enter="addIP" />
        </div>
        <button class="btn-primary add-btn" @click="addIP">
          <el-icon :size="14"><Plus /></el-icon>
          添加
        </button>
      </div>
    </div>

    <!-- 列表 -->
    <div class="list-panel">
      <div class="list-header">
        <h3>{{ activeTab === 'blacklist' ? '黑名单' : '白名单' }}列表</h3>
        <span class="list-count">共 {{ entries.length }} 条</span>
      </div>

      <!-- 卡片式列表 -->
      <div class="ip-cards">
        <div class="ip-card" v-for="entry in pagedEntries" :key="entry.id">
          <div class="ip-card-left">
            <div class="ip-type-dot" :class="entry.list_type"></div>
            <div class="ip-info">
              <span class="ip-addr mono">{{ entry.ip_address }}</span>
              <div class="ip-meta">
                <span class="type-tag" :class="entry.list_type">{{ listTypeText(entry.list_type) }}</span>
                <span class="ip-note" v-if="entry.note">{{ entry.note }}</span>
                <span class="ip-time">{{ formatTime(entry.created_at) }}</span>
              </div>
            </div>
          </div>
          <button class="btn-remove" @click="removeIP(entry.id)" title="移除">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <div class="empty-state" v-if="entries.length === 0 && !loading">
          <div class="empty-visual">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" width="48" height="48"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/></svg>
          </div>
          <div class="empty-text">{{ activeTab === 'blacklist' ? '黑名单为空' : '白名单为空' }}</div>
          <div class="empty-desc">{{ activeTab === 'blacklist' ? '暂无被拦截的 IP 地址，添加 IP 到黑名单以阻止其访问' : '暂无信任的 IP 地址，白名单中的 IP 将跳过所有检测规则' }}</div>
        </div>
      </div>

      <!-- 分页 -->
      <div class="pagination-bar" v-if="entries.length > 0">
        <span class="page-info">共 {{ entries.length }} 条，第 {{ page }} / {{ totalPages }} 页</span>
        <div class="page-controls">
          <button class="page-btn" :disabled="page <= 1" @click="page = 1">首页</button>
          <button class="page-btn" :disabled="page <= 1" @click="page--">上一页</button>
          <span class="page-current">{{ page }}</span>
          <button class="page-btn" :disabled="page >= totalPages" @click="page++">下一页</button>
          <button class="page-btn" :disabled="page >= totalPages" @click="page = totalPages">末页</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { Plus, Filter, CircleClose, CircleCheck, RefreshRight } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const activeTab = ref('blacklist'), entries = ref([]), loading = ref(false)
const newIP = ref(''), newNote = ref('')
const page = ref(1)
let requestSeq = 0

const blacklistCount = ref(0)
const whitelistCount = ref(0)

function formatTime(ts) { return ts ? new Date(ts).toLocaleString('zh-CN') : '-' }
function listTypeText(type) { return type === 'whitelist' ? '白名单' : '黑名单' }
const totalPages = computed(() => Math.max(1, Math.ceil(entries.value.length / 10)))
const pagedEntries = computed(() => entries.value.slice((page.value - 1) * 10, page.value * 10))

async function loadCounts() {
  try {
    const [bl, wl] = await Promise.all([
      api.get('/iplist', { params: { type: 'blacklist' } }).catch(() => []),
      api.get('/iplist', { params: { type: 'whitelist' } }).catch(() => []),
    ])
    blacklistCount.value = Array.isArray(bl) ? bl.length : 0
    whitelistCount.value = Array.isArray(wl) ? wl.length : 0
  } catch {}
}

async function loadIPs() {
  const seq = ++requestSeq
  const type = activeTab.value
  loading.value = true
  try {
    const res = await api.get('/iplist', { params: { type } }) || []
    if (seq === requestSeq) {
      entries.value = res
      page.value = 1
    }
  } catch {} finally {
    if (seq === requestSeq) loading.value = false
  }
  loadCounts()
}

async function addIP() {
  const ip = newIP.value.trim()
  if (!ip) { ElMessage.warning('请输入 IP 地址'); return }
  try {
    await api.post('/iplist', { ip_address: ip, list_type: activeTab.value, note: newNote.value.trim() })
    ElMessage.success('添加成功'); newIP.value = ''; newNote.value = ''; loadIPs()
  } catch {}
}

async function removeIP(id) {
  try {
    await ElMessageBox.confirm('确定移除此 IP？', '确认', { type: 'warning', confirmButtonText: '移除', cancelButtonText: '取消' })
    await api.delete(`/iplist/${id}`); ElMessage.success('已移除'); loadIPs()
  } catch {}
}

onMounted(() => { loadIPs(); loadCounts() })
</script>

<style scoped>
.iplist-page {
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
.btn-outline:hover { border-color: #6366f1; color: #6366f1; }

/* ===== 统计卡片 ===== */
.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
  margin-bottom: 18px;
}
.stat-card {
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  padding: 18px 20px;
  display: flex;
  align-items: center;
  gap: 14px;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
  position: relative;
}
.stat-card:hover { border-color: #cbd5e1; }
.stat-card.active {
  border-color: #6366f1;
  box-shadow: 0 2px 12px rgba(99,102,241,.1);
}
.stat-icon {
  width: 44px; height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.stat-icon.rose { background: #fff1f2; color: #ef4444; }
.stat-icon.emerald { background: #ecfdf5; color: #10b981; }
.stat-icon.indigo { background: #eef2ff; color: #6366f1; }
.stat-body { display: flex; flex-direction: column; }
.stat-value { font-size: 24px; font-weight: 800; color: #0f172a; line-height: 1.2; }
.stat-label { font-size: 12px; color: #94a3b8; margin-top: 2px; }
.stat-arrow {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: #6366f1;
}

/* ===== 添加面板 ===== */
.add-panel {
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  margin-bottom: 18px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
}
.add-header {
  padding: 14px 20px;
  border-bottom: 1px solid #f1f5f9;
}
.add-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
  display: flex;
  align-items: center;
  gap: 8px;
}
.add-header h3 svg { color: #6366f1; }
.add-form {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  align-items: flex-end;
}
.add-field {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.add-field label {
  font-size: 11px;
  font-weight: 700;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}
.add-input {
  height: 40px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 0 14px;
  font-size: 13px;
  outline: none;
  background: #f8f9fc;
  color: #0f172a;
  transition: all 0.2s;
}
.add-input:focus {
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99,102,241,.08);
  background: #fff;
}
.add-input::placeholder { color: #94a3b8; }
.add-btn {
  flex: 0 0 auto;
  width: auto;
  height: 40px;
  padding: 0 24px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.btn-primary {
  border: none;
  border-radius: 10px;
  background: #6366f1;
  color: #fff;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-primary:hover { background: #4f46e5; }

/* ===== 列表面板 ===== */
.list-panel {
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
}
.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid #f1f5f9;
}
.list-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
}
.list-count {
  font-size: 12px;
  color: #94a3b8;
  padding: 3px 10px;
  background: #f1f5f9;
  border-radius: 12px;
  font-weight: 600;
}

/* ===== IP 卡片 ===== */
.ip-cards { padding: 8px; }
.ip-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-radius: 10px;
  transition: background 0.15s;
  margin-bottom: 2px;
}
.ip-card:hover { background: #f8f9fc; }
.ip-card-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}
.ip-type-dot {
  width: 10px; height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.ip-type-dot.blacklist { background: #ef4444; box-shadow: 0 0 6px rgba(239,68,68,.3); }
.ip-type-dot.whitelist { background: #10b981; box-shadow: 0 0 6px rgba(16,185,129,.3); }
.ip-info { flex: 1; min-width: 0; }
.ip-addr {
  display: block;
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 4px;
}
.ip-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.type-tag {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 5px;
}
.type-tag.blacklist { background: #fff1f2; color: #ef4444; }
.type-tag.whitelist { background: #ecfdf5; color: #10b981; }
.ip-note {
  font-size: 12px;
  color: #64748b;
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ip-time {
  font-size: 11.5px;
  color: #94a3b8;
}

.btn-remove {
  width: 32px; height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #cbd5e1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
}
.btn-remove:hover {
  background: #fff1f2;
  color: #ef4444;
}

/* ===== 空状态 ===== */
.empty-state {
  text-align: center;
  padding: 52px 20px;
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
  max-width: 400px;
  margin: 0 auto;
  line-height: 1.6;
}

/* ===== 分页 ===== */
.pagination-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-top: 1px solid #f1f5f9;
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
@media (max-width: 768px) {
  .stats-row { grid-template-columns: 1fr; }
  .add-form { flex-direction: column; }
  .add-btn { width: 100%; justify-content: center; }
  .ip-meta { flex-direction: column; align-items: flex-start; gap: 4px; }
  .pagination-bar { flex-direction: column; gap: 10px; align-items: flex-start; }
}
</style>
