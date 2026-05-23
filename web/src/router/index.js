import { createRouter, createWebHistory } from 'vue-router'
import { getAuthToken } from '../api'

const routes = [
  { path: '/login', component: () => import('../views/Login.vue') },
  { path: '/setup', component: () => import('../views/SetupWizard.vue') },
  { path: '/welcome', component: () => import('../views/WelcomeLetter.vue') },
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', component: () => import('../views/Dashboard.vue'), meta: { requiresAuth: true } },
  { path: '/logs', component: () => import('../views/AttackLogs.vue'), meta: { requiresAuth: true } },
  { path: '/ssh-logs', component: () => import('../views/SSHLogs.vue'), meta: { requiresAuth: true } },
  { path: '/rules', component: () => import('../views/Rules.vue'), meta: { requiresAuth: true } },
  { path: '/sites', component: () => import('../views/Sites.vue'), meta: { requiresAuth: true } },
  { path: '/iplist', component: () => import('../views/IPList.vue'), meta: { requiresAuth: true } },
  { path: '/geo', component: () => import('../views/GeoBlock.vue'), meta: { requiresAuth: true } },
  { path: '/threatintel', component: () => import('../views/ThreatIntel.vue'), meta: { requiresAuth: true } },
  { path: '/audit', component: () => import('../views/AuditLogs.vue'), meta: { requiresAuth: true } },
  { path: '/ai', component: () => import('../views/AISettings.vue'), meta: { requiresAuth: true } },
  { path: '/settings', component: () => import('../views/Settings.vue'), meta: { requiresAuth: true } },
  { path: '/certs', component: () => import('../views/Certificates.vue'), meta: { requiresAuth: true } },
  { path: '/soc-dashboard', component: () => import('../views/SocDashboard.vue'), meta: { requiresAuth: true } },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  const hasToken = !!getAuthToken()
  if (to.meta.requiresAuth && !hasToken) {
    next('/login')
  } else {
    next()
  }
})

export default router
