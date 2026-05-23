<template>
  <div class="certs-page">
    <div class="page-toolbar">
      <div class="heading-group">
        <div class="heading-icon green"><el-icon :size="18"><Lock /></el-icon></div>
        <div>
          <div class="page-heading">SSL 证书</div>
          <div class="page-sub">管理 TLS 证书，监控到期时间</div>
        </div>
      </div>
      <button class="btn-reload" :disabled="reloading" @click="reloadCerts">
        <el-icon :size="14"><RefreshRight /></el-icon>
        {{ reloading ? '重载中' : '重载证书' }}
      </button>
    </div>

    <div class="certs-grid" v-if="certs.length">
      <div class="cert-card" v-for="cert in certs" :key="cert.file_path"
        :class="{ expired: cert.is_expired, warning: !cert.is_expired && cert.days_left <= 30 }">
        <div class="cert-header">
          <div class="cert-domain">
            <el-icon :size="20" class="cert-icon"><Lock /></el-icon>
            <span>{{ cert.domain || '未知域名' }}</span>
          </div>
          <span class="cert-status" :class="statusClass(cert)">
            {{ statusText(cert) }}
          </span>
        </div>

        <div class="cert-details">
          <div class="cert-row">
            <span class="cert-label">颁发者</span>
            <span class="cert-value">{{ cert.issuer || '-' }}</span>
          </div>
          <div class="cert-row">
            <span class="cert-label">主体</span>
            <span class="cert-value">{{ cert.subject || '-' }}</span>
          </div>
          <div class="cert-row">
            <span class="cert-label">生效时间</span>
            <span class="cert-value mono">{{ fmtDate(cert.not_before) }}</span>
          </div>
          <div class="cert-row">
            <span class="cert-label">到期时间</span>
            <span class="cert-value mono" :class="{ 'text-danger': cert.is_expired, 'text-warn': !cert.is_expired && cert.days_left <= 30 }">
              {{ fmtDate(cert.not_after) }}
            </span>
          </div>
          <div class="cert-row">
            <span class="cert-label">剩余天数</span>
            <span class="cert-value" :class="{ 'text-danger': cert.is_expired, 'text-warn': !cert.is_expired && cert.days_left <= 30 }">
              <strong>{{ cert.is_expired ? '已过期' : cert.days_left + ' 天' }}</strong>
            </span>
          </div>
          <div class="cert-row">
            <span class="cert-label">文件路径</span>
            <span class="cert-value mono cert-path">{{ cert.file_path }}</span>
          </div>
        </div>

        <div class="cert-progress">
          <div class="progress-bar">
            <div class="progress-fill" :class="progressClass(cert)" :style="{ width: progressWidth(cert) }"></div>
          </div>
          <div class="progress-labels">
            <span>{{ fmtDate(cert.not_before) }}</span>
            <span>{{ fmtDate(cert.not_after) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="empty-state" v-else-if="!loading">
      <el-icon :size="40"><Lock /></el-icon>
      <div class="empty-title">暂无证书</div>
      <div class="empty-desc">配置 TLS 证书文件或在 ./certs 目录放置证书后，证书信息将在此展示</div>
    </div>

    <div class="empty-state" v-else>
      <div class="spinner"></div>
      <div class="empty-title">加载中...</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Lock, RefreshRight } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const certs = ref([])
const loading = ref(true)
const reloading = ref(false)

async function loadCerts() {
  loading.value = true
  try {
    certs.value = await api.get('/certs') || []
  } catch {
    certs.value = []
  } finally {
    loading.value = false
  }
}

async function reloadCerts() {
  reloading.value = true
  try {
    await api.post('/certs/reload')
    ElMessage.success('证书已重载')
    await loadCerts()
  } finally {
    reloading.value = false
  }
}

function statusClass(cert) {
  if (cert.is_expired) return 'expired'
  if (cert.days_left <= 30) return 'warning'
  return 'valid'
}

function statusText(cert) {
  if (cert.is_expired) return '已过期'
  if (cert.days_left <= 30) return '即将过期'
  return '有效'
}

function progressClass(cert) {
  if (cert.is_expired) return 'expired'
  if (cert.days_left <= 30) return 'warning'
  return 'valid'
}

function progressWidth(cert) {
  const start = new Date(cert.not_before).getTime()
  const end = new Date(cert.not_after).getTime()
  const now = Date.now()
  if (end <= start) return '100%'
  const pct = Math.min(100, Math.max(0, ((now - start) / (end - start)) * 100))
  return pct + '%'
}

function fmtDate(ts) {
  if (!ts) return '-'
  return new Date(ts).toLocaleDateString('zh-CN')
}

onMounted(loadCerts)
</script>

<style scoped>
.certs-page { display: flex; flex-direction: column; gap: 16px; }

.page-toolbar {
  display: flex; justify-content: space-between; align-items: center;
}
.page-heading { font-size: 18px; font-weight: 800; color: var(--text-primary); }
.page-sub { margin-top: 2px; font-size: 12.5px; color: var(--text-muted); }

.btn-reload {
  display: flex; align-items: center; gap: 6px;
  height: 36px; padding: 0 14px; border-radius: var(--radius-btn);
  border: 1px solid var(--border); background: var(--bg-card);
  color: var(--text-secondary); font-size: 13px; font-weight: 600; cursor: pointer;
}
.btn-reload:hover { background: var(--bg-hover); border-color: var(--primary); color: var(--primary); }
.btn-reload:disabled { opacity: .5; cursor: not-allowed; }

.certs-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(420px, 1fr)); gap: 16px; }

.cert-card {
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  overflow: hidden; transition: box-shadow .2s;
}
.cert-card:hover { box-shadow: 0 4px 14px rgba(0,0,0,.06); }
.cert-card.expired { border-color: #fecaca; }
.cert-card.warning { border-color: #fde68a; }

.cert-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 16px 18px; border-bottom: 1px solid var(--border-light);
}
.cert-domain {
  display: flex; align-items: center; gap: 8px;
  font-size: 15px; font-weight: 700; color: var(--text-primary);
}
.cert-icon { color: var(--primary); }
.cert-status {
  padding: 3px 10px; border-radius: 6px; font-size: 12px; font-weight: 700;
}
.cert-status.valid { background: #ecfdf5; color: #059669; }
.cert-status.warning { background: #fffbeb; color: #d97706; }
.cert-status.expired { background: #fef2f2; color: var(--danger); }

.cert-details { padding: 14px 18px; }
.cert-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 0; font-size: 13px;
}
.cert-row + .cert-row { border-top: 1px solid var(--border-light); }
.cert-label { color: var(--text-secondary); font-weight: 600; }
.cert-value { color: var(--text-primary); text-align: right; max-width: 60%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cert-path { font-size: 12px; color: var(--text-secondary); }
.text-danger { color: var(--danger) !important; font-weight: 700; }
.text-warn { color: #d97706 !important; font-weight: 700; }

.cert-progress { padding: 0 18px 16px; }
.progress-bar { height: 6px; background: var(--border-light); border-radius: 999px; overflow: hidden; }
.progress-fill { height: 100%; border-radius: 999px; transition: width .6s; }
.progress-fill.valid { background: linear-gradient(90deg, #22c55e, #16a34a); }
.progress-fill.warning { background: linear-gradient(90deg, #f59e0b, #d97706); }
.progress-fill.expired { background: linear-gradient(90deg, #ef4444, var(--danger)); }
.progress-labels {
  display: flex; justify-content: space-between; margin-top: 4px;
  font-size: 11px; color: var(--text-muted);
}

.empty-state {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 60px 20px; color: var(--text-muted); gap: 10px;
}
.empty-title { font-size: 16px; font-weight: 700; color: var(--text-secondary); }
.empty-desc { font-size: 13px; text-align: center; max-width: 400px; }

.spinner {
  width: 28px; height: 28px; border: 3px solid var(--border);
  border-top-color: var(--primary); border-radius: 50%; animation: spin .8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 768px) {
  .certs-grid { grid-template-columns: 1fr; }
  .page-toolbar { flex-direction: column; align-items: stretch; gap: 12px; }
}
</style>
