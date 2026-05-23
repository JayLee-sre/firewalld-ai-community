<template>
  <div class="layout" v-if="!isStandalone">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-header">
        <div class="brand">
          <div class="brand-icon">
            <img src="/logo.png" alt="智域 WAF" />
          </div>
          <div class="brand-text">
            <span class="brand-name">智域 WAF</span>
            <span class="brand-tag">{{ editionLabel }}</span>
          </div>
        </div>
      </div>

      <nav class="nav-menu">
        <router-link
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: route.path === item.path }"
        >
          <div class="nav-icon-wrap">
            <div class="nav-icon" :class="item.color">
              <el-icon :size="18"><component :is="item.icon" /></el-icon>
            </div>
          </div>
          <div class="nav-content">
            <span class="nav-label">{{ item.label }}</span>
            <span class="nav-desc">{{ item.desc }}</span>
          </div>
          <div class="nav-lock" v-if="item.pro && !isPro">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          </div>
          <div class="nav-indicator" v-if="route.path === item.path"></div>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="engine-status">
          <div class="status-dot" :class="{ active: true }"></div>
          <span class="status-text">防护引擎运行中</span>
        </div>
        <button class="logout-btn" @click="logout">
          <el-icon :size="16"><SwitchButton /></el-icon>
          <span>退出登录</span>
        </button>
      </div>
    </aside>

    <!-- 主内容区 -->
    <!-- 移动端遮罩 -->
    <div class="sidebar-overlay" v-if="sidebarOpen" @click="sidebarOpen = false"></div>

    <main class="main-area">
      <header class="topbar">
        <div class="topbar-left">
          <button class="hamburger-btn" @click="sidebarOpen = !sidebarOpen">
            <el-icon :size="20"><Fold v-if="sidebarOpen" /><Expand v-else /></el-icon>
          </button>
          <div class="breadcrumb">
            <span class="breadcrumb-current">{{ currentMenu?.label }}</span>
            <span class="breadcrumb-desc">{{ currentMenu?.desc }}</span>
          </div>
        </div>
        <div class="topbar-right">
          <div class="status-chips" v-if="!statusLoading">
            <div class="chip">
              <span class="chip-dot" :class="aiEnabled ? 'indigo' : 'muted'"></span>
              <span>AI {{ aiStatus }}</span>
            </div>
            <div class="chip">
              <span class="chip-dot emerald"></span>
              <span>规则 {{ ruleCount }} 条</span>
            </div>
            <div class="chip">
              <span class="chip-dot" :class="systemOk ? 'emerald' : 'rose'"></span>
              <span>{{ systemOk ? '系统正常' : '系统异常' }}</span>
            </div>
          </div>
        </div>
      </header>
      <div class="content-wrapper">
        <router-view v-slot="{ Component }">
          <transition name="page" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
      <footer class="app-footer">© 2026 小睿科技 版权所有</footer>
    </main>
  </div>
  <router-view v-else />
</template>

<script setup>
import { ref, computed, onMounted, watch, provide } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { Monitor, Document, SetUp, Filter, Cpu, SwitchButton, Setting, Key, Fold, Expand, Connection, Location, Warning, DataAnalysis, Lock, List } from '@element-plus/icons-vue'
import api, { clearAuthToken, getAuthToken } from './api'

const route = useRoute()
const router = useRouter()
const isLoginPage = computed(() => route.path === '/login')
const isStandalone = computed(() => route.path === '/login' || route.path === '/soc-dashboard' || route.path === '/setup' || route.path === '/welcome')
const edition = ref('community')
const isPro = computed(() => edition.value === 'pro')
const editionLabel = computed(() => isPro.value ? '专业版' : '社区版')
provide('edition', edition)
provide('isPro', isPro)
const sidebarOpen = ref(false)

watch(() => route.path, () => {
  document.body.style.background = '#f4f6fb'
}, { immediate: true })

watch(() => route.path, () => { sidebarOpen.value = false })

const menuItems = [
  { path: '/dashboard', label: '安全态势', desc: '实时监控总览', icon: Monitor, color: 'indigo' },
  { path: '/logs', label: '攻击日志', desc: '威胁事件追踪', icon: Document, color: 'rose' },
  { path: '/ssh-logs', label: 'SSH 监控', desc: '暴力破解防护', icon: Key, color: 'amber' },
  { path: '/rules', label: '规则引擎', desc: '检测规则管理', icon: SetUp, color: 'amber' },
  { path: '/iplist', label: '访问控制', desc: 'IP 黑白名单', icon: Filter, color: 'cyan' },
  { path: '/threatintel', label: '威胁情报', desc: '自动恶意IP同步', icon: Warning, color: 'rose' },
  { path: '/audit', label: '审计日志', desc: '操作记录追踪', icon: List, color: 'cyan' },
  { path: '/certs', label: 'SSL 证书', desc: 'TLS 证书管理', icon: Lock, color: 'green' },
  { path: '/settings', label: '系统设置', desc: '授权与安全配置', icon: Setting, color: 'slate' },
  { path: '/geo', label: '地理封锁', desc: '按国家/地区屏蔽', icon: Location, color: 'indigo', pro: true },
  { path: '/sites', label: '站点管理', desc: '多站代理回源', icon: Connection, color: 'green', pro: true },
  { path: '/ai', label: 'AI 模型', desc: '智能检测配置', icon: Cpu, color: 'violet', pro: true },
  { path: '/soc-dashboard', label: '监控大屏', desc: '安全态势可视化', icon: DataAnalysis, color: 'green', pro: true },
]

const currentMenu = computed(() => menuItems.find(i => i.path === route.path))
const statusLoading = ref(true)
const aiEnabled = ref(false)
const systemOk = ref(true)
const ruleCount = ref(0)
const aiStatus = computed(() => aiEnabled.value ? '已启用' : '未启用')

function updateDocumentTitle() {
  let pageName = currentMenu.value?.label
  if (route.path === '/login') pageName = '登录'
  else if (route.path === '/soc-dashboard') pageName = '监控大屏'
  document.title = pageName
    ? `${pageName} - 智域 WAF ${editionLabel.value}`
    : `智域 WAF ${editionLabel.value}`
}

async function loadHeaderStatus() {
  if (isLoginPage.value || !getAuthToken()) return
  statusLoading.value = true
  try {
    const [health, rules] = await Promise.all([
      api.get('/health', { suppressError: true }),
      api.get('/rules', { suppressError: true }),
    ])
    systemOk.value = health?.status === 'ok'
    aiEnabled.value = !!health?.ai_enabled
    ruleCount.value = Array.isArray(rules) ? rules.length : 0
    edition.value = health?.edition === 'pro' ? 'pro' : 'community'
  } catch {
    edition.value = 'community'
    systemOk.value = false
  } finally {
    statusLoading.value = false
  }
}

watch(isStandalone, (standalone) => {
  if (!standalone) loadHeaderStatus()
})

watch([() => route.path, editionLabel], updateDocumentTitle, { immediate: true })

onMounted(loadHeaderStatus)

function logout() {
  ElMessageBox.confirm('确定退出当前账号？', '退出确认', {
    confirmButtonText: '退出',
    cancelButtonText: '取消',
    type: 'warning',
  }).then(() => {
    clearAuthToken()
    edition.value = 'community'
    router.push('/login')
  }).catch(() => {})
}
</script>

<style>
/* Page transitions */
.page-enter-active, .page-leave-active { transition: opacity 0.2s ease, transform 0.2s ease; }
.page-enter-from { opacity: 0; }
.page-leave-to { opacity: 0; }
</style>

<style scoped>
.layout {
  height: 100vh;
  display: flex;
  overflow: hidden;
}

/* ===== Sidebar ===== */
.sidebar {
  width: 260px;
  background: #fff;
  border-right: 1px solid #eef0f4;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 20px 20px 16px;
  border-bottom: 1px solid #eef0f4;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-icon {
  width: 38px;
  height: 38px;
  flex-shrink: 0;
  border-radius: 10px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 0 0 1px #eef2f7;
}
.brand-icon img { width: 100%; height: 100%; object-fit: cover; display: block; }
.brand-text { display: flex; flex-direction: column; }
.brand-name { font-size: 18px; font-weight: 800; color: #0f172a; letter-spacing: -0.5px; }
.brand-tag { font-size: 10px; color: #94a3b8; letter-spacing: 1px; text-transform: uppercase; margin-top: 1px; }

.nav-menu {
  flex: 1;
  padding: 12px 12px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  margin-bottom: 2px;
  text-decoration: none;
  color: #64748b;
  transition: all 0.2s ease;
  position: relative;
}
.nav-item:hover {
  background: #f8f9fc;
  color: #334155;
}
.nav-item.active {
  background: linear-gradient(135deg, #eef2ff 0%, #f0f0ff 100%);
  color: #6366f1;
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.1);
}

.nav-icon-wrap { flex-shrink: 0; }
.nav-icon {
  width: 34px; height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}
.nav-icon.indigo { background: #eef2ff; color: #6366f1; }
.nav-icon.rose { background: #fff1f2; color: #e11d48; }
.nav-icon.amber { background: #fffbeb; color: #d97706; }
.nav-icon.cyan { background: #ecfeff; color: #0891b2; }
.nav-icon.violet { background: #f5f3ff; color: #7c3aed; }
.nav-icon.slate { background: #f1f5f9; color: #475569; }
.nav-icon.green { background: #f0fdf4; color: #16a34a; }

.nav-item.active .nav-icon.indigo { background: #6366f1; color: #fff; }
.nav-item.active .nav-icon.rose { background: #e11d48; color: #fff; }
.nav-item.active .nav-icon.amber { background: #d97706; color: #fff; }
.nav-item.active .nav-icon.cyan { background: #0891b2; color: #fff; }
.nav-item.active .nav-icon.violet { background: #7c3aed; color: #fff; }
.nav-item.active .nav-icon.slate { background: #334155; color: #fff; }
.nav-item.active .nav-icon.green { background: #16a34a; color: #fff; }

.nav-content { display: flex; flex-direction: column; min-width: 0; }
.nav-label { font-size: 13.5px; font-weight: 600; }
.nav-desc { font-size: 11px; color: #94a3b8; margin-top: 1px; }
.nav-item.active .nav-desc { color: #818cf8; }

.nav-indicator {
  position: absolute;
  left: 0; top: 50%;
  transform: translateY(-50%);
  width: 3px; height: 20px;
  border-radius: 3px;
  background: #6366f1;
  box-shadow: 0 0 6px rgba(99, 102, 241, 0.4);
}

.nav-lock {
  flex-shrink: 0;
  margin-left: auto;
  color: #d97706;
  opacity: 0.6;
}

.sidebar-footer {
  padding: 16px 20px;
  border-top: 1px solid #eef0f4;
}

.engine-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.status-dot {
  width: 7px; height: 7px;
  border-radius: 50%;
  background: #94a3b8;
}
.status-dot.active {
  background: #22c55e;
  box-shadow: 0 0 6px rgba(34,197,94,0.5);
  animation: pulse 2s infinite;
}
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
.status-text { font-size: 12px; color: #64748b; }

.logout-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  color: #64748b;
  font-size: 12.5px;
  cursor: pointer;
  transition: all 0.2s;
}
.logout-btn:hover {
  background: #fef2f2;
  border-color: #fecaca;
  color: #dc2626;
}

/* ===== Main Area ===== */
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.topbar {
  height: 56px;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid #eef0f4;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 28px;
  flex-shrink: 0;
  position: sticky;
  top: 0;
  z-index: 100;
}

.topbar-left { display: flex; align-items: center; }
.breadcrumb { display: flex; align-items: baseline; gap: 10px; }
.breadcrumb-current { font-size: 16px; font-weight: 700; color: #0f172a; }
.breadcrumb-desc { font-size: 12.5px; color: #94a3b8; }

.topbar-right { display: flex; align-items: center; }
.status-chips { display: flex; gap: 8px; }
.chip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  background: #f8f9fc;
  border: 1px solid #eef0f4;
  border-radius: 20px;
  font-size: 12px;
  color: #475569;
  font-weight: 500;
  transition: background 0.2s ease;
}
.chip:hover {
  background: #f1f5f9;
}
.chip-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
}
.chip-dot.indigo { background: #6366f1; }
.chip-dot.emerald {
  background: #22c55e;
  box-shadow: 0 0 4px rgba(34, 197, 94, 0.4);
}
.chip-dot.rose { background: #ef4444; }
.chip-dot.muted { background: #94a3b8; }

.content-wrapper {
  flex: 1;
  padding: 24px 28px;
  overflow-y: auto;
  background: linear-gradient(180deg, #f4f6fb 0%, #f8f9fc 100%);
}

.app-footer {
  flex-shrink: 0;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-top: 1px solid #eef0f4;
  background: #fff;
  color: #94a3b8;
  font-size: 12px;
}

/* ===== Mobile ===== */
.hamburger-btn {
  display: none;
  background: none; border: none; color: #475569; cursor: pointer;
  padding: 6px; border-radius: 8px;
}
.hamburger-btn:hover { background: #f1f5f9; }

.sidebar-overlay {
  display: none;
  position: fixed; inset: 0; background: rgba(0,0,0,0.35); z-index: 1000;
}

@media (max-width: 768px) {
  .sidebar {
    position: fixed; left: -260px; top: 0; bottom: 0;
    z-index: 1001; transition: left 0.3s ease;
    box-shadow: 4px 0 20px rgba(0,0,0,0.1);
  }
  .sidebar.open { left: 0; }
  .sidebar-overlay { display: block; }
  .hamburger-btn { display: flex; }
  .brand-tag { display: none; }
  .status-chips { display: none; }
  .topbar { padding: 0 16px; }
  .breadcrumb-desc { display: none; }
  .content-wrapper { padding: 16px; }
  .app-footer { height: 30px; font-size: 11px; }
}
</style>
