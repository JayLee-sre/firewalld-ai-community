<template>
  <div class="settings-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <div class="page-icon slate">
          <el-icon :size="20"><Setting /></el-icon>
        </div>
        <div>
          <h1 class="page-title">系统设置</h1>
          <p class="page-desc">管理授权、密码和系统运维配置</p>
        </div>
      </div>
      <button class="btn-outline" :disabled="loadingHealth" @click="loadHealth">
        <el-icon :size="14"><RefreshRight /></el-icon>
        {{ loadingHealth ? '加载中' : '刷新' }}
      </button>
    </div>

    <!-- 授权面板 (全宽) -->
    <div class="panel license-panel">
      <div class="panel-head">
        <div class="panel-title-group">
          <div class="panel-icon indigo">
            <el-icon :size="16"><Key /></el-icon>
          </div>
          <h2>授权管理</h2>
        </div>
        <span class="edition-badge" :class="{ pro: isPro }">
          {{ isPro ? '专业版' : '社区版' }}
        </span>
      </div>

      <div class="license-body">
        <div class="license-status-card" :class="{ active: isPro }">
          <div class="license-status-icon">
            <el-icon :size="28"><CircleCheck v-if="isPro" /><Key v-else /></el-icon>
          </div>
          <div class="license-status-info">
            <strong>{{ isPro ? (licenseInfo.customer || '专业版已激活') : '当前为社区版' }}</strong>
            <span>{{ licenseSummary }}</span>
          </div>
          <div class="license-metrics-row">
            <div class="license-metric">
              <span class="metric-label">授权状态</span>
              <strong class="metric-value">{{ licenseStatusText }}</strong>
            </div>
            <div class="license-metric">
              <span class="metric-label">剩余天数</span>
              <strong class="metric-value">{{ remainingDaysText }}</strong>
            </div>
            <div class="license-metric">
              <span class="metric-label">版本类型</span>
              <strong class="metric-value">{{ isPro ? '专业版' : '社区版' }}</strong>
            </div>
            <div class="license-metric">
              <span class="metric-label">到期时间</span>
              <strong class="metric-value">{{ isPro ? formatDateTime(licenseInfo.expires_at) : '-' }}</strong>
            </div>
          </div>
        </div>

        <div v-if="!isPro" class="activate-section">
          <div class="activate-hint">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
            <span>升级专业版解锁 AI 检测、地理封锁、站点管理等高级功能</span>
          </div>
          <div class="activate-form">
            <input v-model.trim="licenseKey" placeholder="请输入授权码" autocomplete="off" class="activate-input" />
            <button class="activate-btn" :disabled="activating || !licenseKey" @click="activateLicense">
              {{ activating ? '校验中...' : '激活授权' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 两列布局 -->
    <div class="settings-grid">
      <!-- 管理员密码 -->
      <div class="panel">
        <div class="panel-head">
          <div class="panel-title-group">
            <div class="panel-icon amber">
              <el-icon :size="16"><Key /></el-icon>
            </div>
            <h2>管理员密码</h2>
          </div>
        </div>
        <div class="panel-body">
          <div class="field">
            <label>当前密码</label>
            <div class="password-input">
              <input v-model="pwdForm.old_password" :type="showOld ? 'text' : 'password'" autocomplete="current-password" placeholder="输入当前密码" />
              <button type="button" class="toggle-vis" @click="showOld = !showOld">
                <el-icon :size="14"><View v-if="!showOld" /><Hide v-else /></el-icon>
              </button>
            </div>
          </div>
          <div class="field">
            <label>新密码</label>
            <div class="password-input">
              <input v-model="pwdForm.new_password" :type="showNew ? 'text' : 'password'" autocomplete="new-password" placeholder="至少 12 位字符" />
              <button type="button" class="toggle-vis" @click="showNew = !showNew">
                <el-icon :size="14"><View v-if="!showNew" /><Hide v-else /></el-icon>
              </button>
            </div>
          </div>
          <div class="field">
            <label>确认密码</label>
            <div class="password-input">
              <input v-model="pwdConfirm" :type="showConfirm ? 'text' : 'password'" autocomplete="new-password" placeholder="再次输入新密码" />
              <button type="button" class="toggle-vis" @click="showConfirm = !showConfirm">
                <el-icon :size="14"><View v-if="!showConfirm" /><Hide v-else /></el-icon>
              </button>
            </div>
          </div>
          <button class="btn-primary" :disabled="changingPwd" @click="changePassword">
            {{ changingPwd ? '更新中...' : '更新密码' }}
          </button>
        </div>
      </div>

      <!-- 协议与防护 -->
      <div class="panel">
        <div class="panel-head">
          <div class="panel-title-group">
            <div class="panel-icon emerald">
              <el-icon :size="16"><Setting /></el-icon>
            </div>
            <h2>协议与防护</h2>
          </div>
        </div>
        <div class="panel-body">
          <div class="toggle-card">
            <div class="toggle-info">
              <strong>HTTP/2 协议</strong>
              <span>启用后通过 TLS ALPN 自动协商 HTTP/2，提升传输性能</span>
            </div>
            <label class="switch">
              <input type="checkbox" v-model="settingsForm.http2" />
              <span class="slider"></span>
            </label>
          </div>
          <div class="toggle-card">
            <div class="toggle-info">
              <strong>动态防护</strong>
              <span>每次请求注入随机化脚本，防止爬虫和自动化工具分析</span>
            </div>
            <label class="switch">
              <input type="checkbox" v-model="settingsForm.dynamicProtect" />
              <span class="slider"></span>
            </label>
          </div>
          <button class="btn-primary" :disabled="savingSettings" @click="saveSettings">
            {{ savingSettings ? '保存中...' : '保存设置' }}
          </button>
        </div>
      </div>

      <!-- 运维操作 -->
      <div class="panel">
        <div class="panel-head">
          <div class="panel-title-group">
            <div class="panel-icon rose">
              <el-icon :size="16"><RefreshRight /></el-icon>
            </div>
            <h2>运维操作</h2>
          </div>
        </div>
        <div class="panel-body">
          <div class="ops-info">
            <p>重载配置会重新读取本地配置、规则和授权状态，不会清空数据或重启服务。</p>
          </div>
          <div class="ops-grid">
            <div class="ops-card">
              <div class="ops-card-icon">
                <el-icon :size="18"><RefreshRight /></el-icon>
              </div>
              <div class="ops-card-info">
                <strong>重载配置</strong>
                <span>修改配置后手动生效</span>
              </div>
              <button class="btn-secondary" :disabled="reloading" @click="reloadConfig">
                {{ reloading ? '重载中' : '重载' }}
              </button>
            </div>
            <div class="ops-card">
              <div class="ops-card-icon dark">
                <el-icon :size="18"><RefreshRight /></el-icon>
              </div>
              <div class="ops-card-info">
                <strong>版本更新</strong>
                <span>检查并安装新版本</span>
              </div>
              <button class="btn-dark" :disabled="checkingUpdate" @click="checkVersionUpdate">
                {{ checkingUpdate ? '检查中' : '检查' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 备份与恢复 -->
      <div class="panel">
        <div class="panel-head">
          <div class="panel-title-group">
            <div class="panel-icon violet">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
            </div>
            <h2>备份与恢复</h2>
          </div>
        </div>
        <div class="panel-body">
          <p class="section-desc">导出或导入系统配置（规则、IP 列表、站点、地理围栏和设置）</p>
          <div class="backup-row">
            <button class="btn-primary" :disabled="exporting" @click="exportBackup">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="15" height="15"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
              {{ exporting ? '导出中...' : '导出配置' }}
            </button>
            <label class="btn-outline-primary">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="15" height="15"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
              {{ importing ? '导入中...' : '导入配置' }}
              <input type="file" accept=".json" @change="importBackup" :disabled="importing" hidden />
            </label>
          </div>
          <div class="import-result" v-if="importResult">
            <div class="import-summary">
              <span v-for="(count, key) in importResult.imported" :key="key">
                {{ importLabel(key) }}: {{ count }}
              </span>
            </div>
            <div class="import-errors" v-if="importResult.errors?.length">
              <div v-for="err in importResult.errors" :key="err" class="error-line">{{ err }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 用户管理 -->
      <div class="panel full-width">
        <div class="panel-head">
          <div class="panel-title-group">
            <div class="panel-icon cyan">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
            </div>
            <h2>用户管理</h2>
          </div>
          <button class="btn-sm" @click="showCreateUser = !showCreateUser">
            <el-icon :size="12"><Plus /></el-icon>
            {{ showCreateUser ? '收起' : '新建用户' }}
          </button>
        </div>
        <div class="panel-body">
          <!-- 创建用户表单 -->
          <div class="create-user-form" v-if="showCreateUser">
            <div class="form-row-3">
              <input v-model="newUser.username" class="form-input" placeholder="用户名" />
              <input v-model="newUser.password" type="password" class="form-input" placeholder="密码（至少 12 位）" />
              <select v-model="newUser.role" class="form-select">
                <option value="operator">操作员</option>
                <option value="viewer">只读用户</option>
                <option value="admin">管理员</option>
              </select>
            </div>
            <button class="btn-primary" :disabled="creatingUser || !newUser.username || !newUser.password" @click="createUser">
              {{ creatingUser ? '创建中...' : '创建用户' }}
            </button>
          </div>

          <!-- 用户列表 -->
          <div class="user-grid">
            <div class="user-card" v-for="u in users" :key="u.id">
              <div class="user-avatar" :class="u.role">
                {{ u.username.charAt(0).toUpperCase() }}
              </div>
              <div class="user-info">
                <strong>{{ u.username }}</strong>
                <span class="role-badge" :class="u.role">{{ roleLabel(u.role) }}</span>
              </div>
              <button class="btn-text-danger" @click="deleteUser(u)" :disabled="u.role === 'admin' && adminCount <= 1">
                删除
              </button>
            </div>
            <div class="empty-mini" v-if="!users.length">
              <span>暂无用户数据</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, reactive, inject, onMounted } from 'vue'
import { CircleCheck, Hide, Key, RefreshRight, View, Setting, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const edition = inject('edition', ref('community'))
const isPro = computed(() => edition.value === 'pro')

const health = ref({})
const loadingHealth = ref(false)
const showOld = ref(false)
const showNew = ref(false)
const showConfirm = ref(false)
const changingPwd = ref(false)
const reloading = ref(false)
const checkingUpdate = ref(false)
const activating = ref(false)
const savingSettings = ref(false)
const licenseKey = ref('')
const pwdForm = reactive({ old_password: '', new_password: '' })
const pwdConfirm = ref('')
const settingsForm = reactive({ http2: false, dynamicProtect: false })

const exporting = ref(false)
const importing = ref(false)
const importResult = ref(null)

const users = ref([])
const showCreateUser = ref(false)
const creatingUser = ref(false)
const newUser = reactive({ username: '', password: '', role: 'operator' })
const adminCount = computed(() => users.value.filter(u => u.role === 'admin').length)

const licenseInfo = computed(() => health.value.license || {})
const licenseStatusText = computed(() => {
  if (isPro.value) return '已激活'
  if (licenseInfo.value.status && licenseInfo.value.status !== 'community') return '授权异常'
  return '未激活'
})
const licenseSummary = computed(() => {
  if (!isPro.value) return '输入授权码后自动绑定当前机器并升级专业版'
  const expires = formatDateTime(licenseInfo.value.expires_at)
  return remainingDays.value === null ? `到期：${expires}` : `到期：${expires}，剩余 ${remainingDays.value} 天`
})
const remainingDays = computed(() => {
  const value = licenseInfo.value.expires_at
  if (!value) return null
  const expires = new Date(value).getTime()
  if (Number.isNaN(expires)) return null
  return Math.max(0, Math.ceil((expires - Date.now()) / 86400000))
})
const remainingDaysText = computed(() => {
  if (!isPro.value) return '-'
  if (remainingDays.value === null) return '未知'
  return `${remainingDays.value} 天`
})

async function loadHealth() {
  loadingHealth.value = true
  try {
    const [h, settings] = await Promise.all([
      api.get('/health', { suppressError: true }),
      api.get('/settings', { suppressError: true }),
    ])
    health.value = h || {}
    edition.value = h?.edition === 'pro' ? 'pro' : 'community'
    if (settings) {
      settingsForm.dynamicProtect = settings.dynamic_protect === 'true'
    }
  } catch {
    edition.value = 'community'
    health.value = {}
  } finally {
    loadingHealth.value = false
  }
}

async function activateLicense() {
  if (!licenseKey.value) return
  activating.value = true
  try {
    const r = await api.post('/license/activate', { license_key: licenseKey.value })
    ElMessage.success(r.message || '授权已激活')
    edition.value = r.edition === 'pro' ? 'pro' : 'community'
    licenseKey.value = ''
    await loadHealth()
  } finally {
    activating.value = false
  }
}

async function changePassword() {
  if (!pwdForm.old_password || !pwdForm.new_password || !pwdConfirm.value) {
    ElMessage.warning('请填写完整密码')
    return
  }
  if (pwdForm.new_password.length < 12) {
    ElMessage.warning('新密码至少 12 位字符')
    return
  }
  if (pwdForm.new_password !== pwdConfirm.value) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  changingPwd.value = true
  try {
    await api.post('/auth/password', pwdForm)
    ElMessage.success('密码已更新，请重新登录')
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdConfirm.value = ''
  } finally {
    changingPwd.value = false
  }
}

async function reloadConfig() {
  if (reloading.value) return
  reloading.value = true
  try {
    const r = await api.post('/config/reload')
    ElMessage.success(r.message || '配置已重载')
    await loadHealth()
  } finally {
    reloading.value = false
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    await api.put('/settings', {
      dynamic_protect: String(settingsForm.dynamicProtect),
    })
    ElMessage.success('设置已保存')
    await reloadConfig()
  } finally {
    savingSettings.value = false
  }
}

async function checkVersionUpdate() {
  if (checkingUpdate.value) return
  checkingUpdate.value = true
  try {
    await api.post('/system/update/check', {}, { suppressError: true })
    ElMessage.success('已提交版本检查请求')
  } catch {
    ElMessage.info('版本更新接口已预留，当前版本暂未接入在线更新服务')
  } finally {
    checkingUpdate.value = false
  }
}

function formatDateTime(value) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleDateString('zh-CN')
}

async function exportBackup() {
  exporting.value = true
  try {
    const resp = await fetch('/api/v1/backup/export', {
      headers: { Authorization: `Bearer ${localStorage.getItem('zhiyu_waf_token')}` },
    })
    const blob = await resp.json()
    const url = URL.createObjectURL(new Blob([JSON.stringify(blob, null, 2)], { type: 'application/json' }))
    const a = document.createElement('a')
    a.href = url
    a.download = `zhiyu-waf-backup-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('配置已导出')
  } finally {
    exporting.value = false
  }
}

async function importBackup(e) {
  const file = e.target.files?.[0]
  if (!file) return
  importing.value = true
  importResult.value = null
  try {
    const text = await file.text()
    const data = JSON.parse(text)
    const result = await api.post('/backup/import', data)
    importResult.value = result
    if (result.errors?.length) {
      ElMessage.warning(`导入完成，${result.errors.length} 个错误`)
    } else {
      ElMessage.success('配置已导入')
    }
  } catch (err) {
    ElMessage.error('导入失败: ' + (err.message || '未知错误'))
  } finally {
    importing.value = false
    e.target.value = ''
  }
}

function importLabel(key) {
  return { rules: '规则', ip_entries: 'IP', sites: '站点', geo_rules: '地理围栏', settings: '设置' }[key] || key
}

async function loadUsers() {
  try { users.value = await api.get('/users') || [] } catch { users.value = [] }
}

async function createUser() {
  if (!newUser.username || !newUser.password) return
  creatingUser.value = true
  try {
    await api.post('/users', { ...newUser })
    ElMessage.success('用户已创建')
    newUser.username = ''
    newUser.password = ''
    newUser.role = 'operator'
    showCreateUser.value = false
    await loadUsers()
  } finally {
    creatingUser.value = false
  }
}

async function deleteUser(u) {
  try {
    await ElMessageBox.confirm(`确定删除用户 "${u.username}"？`, '删除确认', { type: 'warning' })
  } catch { return }
  try {
    await api.delete(`/users/${u.id}`)
    ElMessage.success('用户已删除')
    await loadUsers()
  } catch {}
}

function roleLabel(role) {
  return { admin: '管理员', operator: '操作员', viewer: '只读' }[role] || role
}

onMounted(() => { loadHealth(); loadUsers() })
</script>

<style scoped>
.settings-page {
  max-width: 1200px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

/* ===== 页面头部 ===== */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
.page-icon.slate { background: #f1f5f9; color: #475569; }
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
.btn-outline:disabled { opacity: 0.5; cursor: not-allowed; }

/* ===== 面板通用 ===== */
.panel {
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0,0,0,.03);
}
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #f1f5f9;
}
.panel-title-group {
  display: flex;
  align-items: center;
  gap: 10px;
}
.panel-icon {
  width: 32px; height: 32px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.panel-icon.indigo { background: #eef2ff; color: #6366f1; }
.panel-icon.amber { background: #fffbeb; color: #d97706; }
.panel-icon.emerald { background: #ecfdf5; color: #10b981; }
.panel-icon.rose { background: #fff1f2; color: #ef4444; }
.panel-icon.violet { background: #f5f3ff; color: #7c3aed; }
.panel-icon.cyan { background: #ecfeff; color: #0891b2; }
.panel-head h2 {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}
.panel-body {
  padding: 20px;
}
.edition-badge {
  padding: 4px 12px;
  border-radius: 20px;
  background: #f1f5f9;
  color: #94a3b8;
  font-size: 12px;
  font-weight: 800;
}
.edition-badge.pro { background: #ecfdf5; color: #10b981; }

/* ===== 授权面板 ===== */
.license-panel { }
.license-body { padding: 20px; }
.license-status-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  border-radius: 12px;
  background: #f8f9fc;
  border: 1px solid #eef0f4;
  flex-wrap: wrap;
}
.license-status-card.active {
  background: #f0fdf4;
  border-color: #bbf7d0;
}
.license-status-icon {
  width: 52px; height: 52px;
  border-radius: 14px;
  background: #e2e8f0;
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.license-status-card.active .license-status-icon {
  background: #dcfce7;
  color: #16a34a;
}
.license-status-info {
  flex: 1;
  min-width: 200px;
}
.license-status-info strong {
  display: block;
  font-size: 16px;
  color: #0f172a;
}
.license-status-info span {
  display: block;
  font-size: 12.5px;
  color: #64748b;
  margin-top: 3px;
}
.license-metrics-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  width: 100%;
  margin-top: 8px;
}
.license-metric {
  padding: 12px 14px;
  background: #fff;
  border-radius: 10px;
  border: 1px solid #eef0f4;
}
.metric-label {
  display: block;
  font-size: 11px;
  color: #94a3b8;
  font-weight: 600;
  margin-bottom: 4px;
}
.metric-value {
  display: block;
  font-size: 14px;
  color: #0f172a;
  font-weight: 700;
}

.activate-section {
  margin-top: 16px;
}
.activate-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: #eef2ff;
  border-radius: 10px;
  color: #4338ca;
  font-size: 12.5px;
  font-weight: 500;
  margin-bottom: 12px;
}
.activate-form {
  display: flex;
  gap: 10px;
}
.activate-input {
  flex: 1;
  height: 40px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 0 14px;
  font-size: 13px;
  outline: none;
  transition: all 0.2s;
  background: #f8f9fc;
}
.activate-input:focus {
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99,102,241,.08);
  background: #fff;
}
.activate-btn {
  height: 40px;
  padding: 0 24px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.activate-btn:hover { box-shadow: 0 4px 12px rgba(99,102,241,.3); }
.activate-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* ===== 设置网格 ===== */
.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 18px;
  align-items: start;
}
.full-width { grid-column: 1 / -1; }

/* ===== 字段 ===== */
.field {
  margin-bottom: 14px;
}
.field:last-of-type { margin-bottom: 18px; }
.field label {
  display: block;
  font-size: 12px;
  font-weight: 700;
  color: #475569;
  margin-bottom: 6px;
}
.password-input {
  position: relative;
}
.password-input input {
  width: 100%;
  height: 40px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 0 40px 0 14px;
  font-size: 13px;
  outline: none;
  transition: all 0.2s;
  background: #f8f9fc;
  color: #0f172a;
}
.password-input input:focus {
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99,102,241,.08);
  background: #fff;
}
.password-input input::placeholder { color: #94a3b8; }
.toggle-vis {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 28px; height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #94a3b8;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.toggle-vis:hover { color: #475569; }

/* ===== 按钮 ===== */
.btn-primary {
  width: 100%;
  height: 40px;
  border: none;
  border-radius: 10px;
  background: #6366f1;
  color: #fff;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}
.btn-primary:hover { background: #4f46e5; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 9px;
  background: #6366f1;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.btn-secondary:hover { background: #4f46e5; }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-dark {
  height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 9px;
  background: #0f172a;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.btn-dark:hover { background: #1e293b; }
.btn-dark:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-sm {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 12px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  background: #fff;
  color: #6366f1;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-sm:hover { background: #eef2ff; }

.btn-outline-primary {
  flex: 1;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
  background: #fff;
  color: #475569;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-outline-primary:hover { border-color: #6366f1; color: #6366f1; }

.btn-text-danger {
  border: none;
  background: none;
  color: #ef4444;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
  transition: all 0.2s;
}
.btn-text-danger:hover { background: #fff1f2; }
.btn-text-danger:disabled { opacity: 0.3; cursor: not-allowed; }

/* ===== Toggle ===== */
.toggle-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  background: #f8f9fc;
  border-radius: 10px;
  border: 1px solid #eef0f4;
  margin-bottom: 12px;
}
.toggle-card:last-of-type { margin-bottom: 18px; }
.toggle-info { flex: 1; min-width: 0; }
.toggle-info strong { display: block; font-size: 13.5px; color: #0f172a; }
.toggle-info span { display: block; font-size: 12px; color: #94a3b8; margin-top: 2px; }

.switch {
  position: relative;
  width: 44px; height: 24px;
  flex-shrink: 0;
}
.switch input { opacity: 0; width: 0; height: 0; }
.slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: #cbd5e1;
  border-radius: 24px;
  transition: 0.3s;
}
.slider:before {
  content: "";
  position: absolute;
  height: 18px; width: 18px;
  left: 3px; bottom: 3px;
  background: #fff;
  border-radius: 50%;
  transition: 0.3s;
}
.switch input:checked + .slider { background: #6366f1; }
.switch input:checked + .slider:before { transform: translateX(20px); }

/* ===== 运维 ===== */
.ops-info p {
  margin: 0 0 14px;
  font-size: 13px;
  color: #64748b;
  line-height: 1.6;
}
.ops-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ops-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #f8f9fc;
  border-radius: 10px;
  border: 1px solid #eef0f4;
}
.ops-card-icon {
  width: 38px; height: 38px;
  border-radius: 10px;
  background: #eef2ff;
  color: #6366f1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.ops-card-icon.dark { background: #f1f5f9; color: #0f172a; }
.ops-card-info {
  flex: 1;
}
.ops-card-info strong { display: block; font-size: 13.5px; color: #0f172a; }
.ops-card-info span { display: block; font-size: 12px; color: #94a3b8; margin-top: 1px; }

/* ===== 备份 ===== */
.section-desc {
  margin: 0 0 14px;
  font-size: 13px;
  color: #64748b;
  line-height: 1.6;
}
.backup-row {
  display: flex;
  gap: 10px;
}
.import-result {
  margin-top: 12px;
  padding: 12px 14px;
  background: #f8f9fc;
  border: 1px solid #eef0f4;
  border-radius: 10px;
  font-size: 12px;
}
.import-summary { display: flex; flex-wrap: wrap; gap: 8px; color: #475569; }
.import-summary span {
  padding: 3px 10px;
  background: #eef2ff;
  border-radius: 6px;
  color: #6366f1;
  font-weight: 600;
}
.import-errors { margin-top: 8px; border-top: 1px solid #eef0f4; padding-top: 8px; }
.error-line { color: #ef4444; font-size: 12px; padding: 2px 0; }

/* ===== 用户管理 ===== */
.create-user-form {
  margin-bottom: 16px;
  padding: 16px;
  background: #f8f9fc;
  border: 1px solid #eef0f4;
  border-radius: 12px;
}
.form-row-3 {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 10px;
  margin-bottom: 10px;
}
.form-input, .form-select {
  height: 40px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 0 12px;
  font-size: 13px;
  outline: none;
  background: #fff;
  color: #0f172a;
  transition: all 0.2s;
}
.form-input:focus, .form-select:focus {
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99,102,241,.08);
}
.form-input::placeholder { color: #94a3b8; }

.user-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}
.user-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #f8f9fc;
  border-radius: 10px;
  border: 1px solid #eef0f4;
  transition: all 0.2s;
}
.user-card:hover { border-color: #cbd5e1; }
.user-avatar {
  width: 38px; height: 38px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  font-weight: 800;
  flex-shrink: 0;
}
.user-avatar.admin { background: #eef2ff; color: #6366f1; }
.user-avatar.operator { background: #ecfdf5; color: #10b981; }
.user-avatar.viewer { background: #f1f5f9; color: #64748b; }
.user-info {
  flex: 1;
  min-width: 0;
}
.user-info strong {
  display: block;
  font-size: 13.5px;
  color: #0f172a;
}
.role-badge {
  display: inline-block;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 8px;
  border-radius: 5px;
  margin-top: 3px;
}
.role-badge.admin { background: #eef2ff; color: #6366f1; }
.role-badge.operator { background: #ecfdf5; color: #10b981; }
.role-badge.viewer { background: #f1f5f9; color: #94a3b8; }

.empty-mini {
  grid-column: 1 / -1;
  text-align: center;
  padding: 32px;
  color: #94a3b8;
  font-size: 13px;
}

/* ===== 响应式 ===== */
@media (max-width: 768px) {
  .settings-grid { grid-template-columns: 1fr; }
  .full-width { grid-column: auto; }
  .license-metrics-row { grid-template-columns: repeat(2, 1fr); }
  .activate-form { flex-direction: column; }
  .backup-row { flex-direction: column; }
  .form-row-3 { grid-template-columns: 1fr; }
  .user-grid { grid-template-columns: 1fr; }
}
</style>
