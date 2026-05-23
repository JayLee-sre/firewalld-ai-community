<template>
  <div class="setup-page">
    <div class="setup-card">
      <div class="setup-header">
        <img src="/logo.png" alt="智域 WAF" class="setup-logo" />
        <h1>智域 WAF 初始配置向导</h1>
        <p class="setup-subtitle">几步完成核心配置，即刻开启智能防护</p>
      </div>

      <el-steps :active="step" finish-status="success" class="steps-bar" align-center>
        <el-step title="管理员密码" />
        <el-step title="代理配置" />
        <el-step title="AI 引擎" />
        <el-step title="安全策略" />
        <el-step title="完成" />
      </el-steps>

      <!-- Step 0: Password -->
      <div class="step-body" v-if="step === 0">
        <h2>设置管理员密码</h2>
        <p class="step-desc">首次使用，请设置管理员密码（至少12位）</p>
        <div class="form-group">
          <label>新密码</label>
          <el-input v-model="form.password" type="password" show-password placeholder="请输入管理员密码" size="large" />
        </div>
        <div class="form-group">
          <label>确认密码</label>
          <el-input v-model="form.confirmPassword" type="password" show-password placeholder="请再次输入密码" size="large" />
        </div>
      </div>

      <!-- Step 1: Proxy -->
      <div class="step-body" v-if="step === 1">
        <h2>代理配置</h2>
        <p class="step-desc">配置 WAF 代理，将流量转发到您的后端服务</p>
        <div class="form-group">
          <label>后端地址 <span class="required">*</span></label>
          <el-input v-model="form.backendAddr" placeholder="例如: 127.0.0.1:8080" size="large" />
          <span class="field-hint">您的 Web 应用实际监听的地址和端口</span>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>WAF 监听端口</label>
            <el-input-number v-model="form.listenPort" :min="1" :max="65535" size="large" style="width: 100%" />
            <span class="field-hint">WAF 代理监听的端口，默认 8080</span>
          </div>
          <div class="form-group">
            <label>iptables 端口</label>
            <el-input-number v-model="form.iptablesPort" :min="1" :max="65535" size="large" style="width: 100%" />
            <span class="field-hint">将此端口流量重定向到 WAF，默认 80</span>
          </div>
        </div>
        <div class="form-group">
          <el-switch v-model="form.iptablesEnable" />
          <span class="switch-label">启用 iptables 流量重定向（需要 root 权限）</span>
        </div>
      </div>

      <!-- Step 2: AI -->
      <div class="step-body" v-if="step === 2">
        <h2>AI 检测引擎</h2>
        <p class="step-desc">配置 AI 模型实现智能攻击检测（可选）</p>
        <div class="form-group">
          <el-switch v-model="form.aiEnabled" />
          <span class="switch-label">启用 AI 检测引擎</span>
        </div>
        <template v-if="form.aiEnabled">
          <div class="form-group">
            <label>API Key <span class="required">*</span></label>
            <el-input v-model="form.apiKey" placeholder="sk-xxxxxxxx" size="large" show-password />
            <span class="field-hint">兼容 OpenAI API 格式的密钥</span>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label>模型名称</label>
              <el-input v-model="form.aiModel" placeholder="gpt-4o / deepseek-chat" size="large" />
            </div>
            <div class="form-group">
              <label>API Base URL</label>
              <el-input v-model="form.aiBaseURL" placeholder="https://api.openai.com/v1" size="large" />
            </div>
          </div>
        </template>
      </div>

      <!-- Step 3: Security -->
      <div class="step-body" v-if="step === 3">
        <h2>安全策略</h2>
        <p class="step-desc">配置基础防护策略</p>
        <div class="form-row">
          <div class="form-group">
            <label>每分钟请求限制</label>
            <el-input-number v-model="form.rpm" :min="1" :max="10000" size="large" style="width: 100%" />
            <span class="field-hint">单 IP 每分钟最大请求数</span>
          </div>
          <div class="form-group">
            <label>突发容量</label>
            <el-input-number v-model="form.burstSize" :min="1" :max="1000" size="large" style="width: 100%" />
            <span class="field-hint">允许的瞬时突发请求数</span>
          </div>
        </div>
        <div class="section-divider"></div>
        <div class="form-group">
          <el-switch v-model="form.sshEnabled" />
          <span class="switch-label">启用 SSH 暴力破解防护</span>
        </div>
        <template v-if="form.sshEnabled">
          <div class="form-row">
            <div class="form-group">
              <label>最大失败次数</label>
              <el-input-number v-model="form.sshMaxFails" :min="1" :max="20" size="large" style="width: 100%" />
            </div>
            <div class="form-group">
              <label>封禁时长（分钟）</label>
              <el-input-number v-model="form.sshBanMinutes" :min="1" :max="1440" size="large" style="width: 100%" />
            </div>
          </div>
        </template>
      </div>

      <!-- Step 4: Summary -->
      <div class="step-body" v-if="step === 4">
        <h2>配置完成</h2>
        <p class="step-desc">请确认以下配置信息</p>
        <div class="summary-grid">
          <div class="summary-item">
            <span class="summary-label">代理后端</span>
            <strong>{{ form.backendAddr || '未设置' }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">监听端口</span>
            <strong>:{{ form.listenPort }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">AI 引擎</span>
            <strong :class="form.aiEnabled ? 'text-green' : 'text-muted'">{{ form.aiEnabled ? '已启用' : '未启用' }}</strong>
          </div>
          <div class="summary-item" v-if="form.aiEnabled">
            <span class="summary-label">AI 模型</span>
            <strong>{{ form.aiModel || '默认' }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">速率限制</span>
            <strong>{{ form.rpm }} req/min</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">SSH 防护</span>
            <strong :class="form.sshEnabled ? 'text-green' : 'text-muted'">{{ form.sshEnabled ? '已启用' : '未启用' }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">iptables</span>
            <strong :class="form.iptablesEnable ? 'text-green' : 'text-muted'">{{ form.iptablesEnable ? '已启用' : '未启用' }}</strong>
          </div>
        </div>
      </div>

      <!-- Navigation -->
      <div class="step-nav">
        <el-button v-if="step > 0" @click="step--" size="large">上一步</el-button>
        <div class="nav-spacer"></div>
        <el-button v-if="step < 4" type="primary" @click="nextStep" size="large" :disabled="!canNext">
          下一步
        </el-button>
        <el-button v-if="step === 4" type="primary" @click="applySetup" size="large" :loading="applying">
          立即启动
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api, { setAuthToken } from '../api'

const router = useRouter()
const step = ref(0)
const applying = ref(false)

const form = ref({
  password: '',
  confirmPassword: '',
  backendAddr: '127.0.0.1:7788',
  listenPort: 8080,
  iptablesEnable: true,
  iptablesPort: 80,
  aiEnabled: true,
  apiKey: '',
  aiModel: '',
  aiBaseURL: '',
  rpm: 60,
  burstSize: 10,
  sshEnabled: true,
  sshMaxFails: 5,
  sshBanMinutes: 30,
})

const canNext = computed(() => {
  if (step.value === 0) {
    return form.value.password.length >= 12 && form.value.password === form.value.confirmPassword
  }
  if (step.value === 1) {
    return !!form.value.backendAddr
  }
  return true
})

async function nextStep() {
  if (step.value === 0) {
    try {
      await api.post('/setup/password', { password: form.value.password })
    } catch (e) {
      ElMessage.error('密码设置失败')
      return
    }
  }
  step.value++
}

async function applySetup() {
  applying.value = true
  try {
    const f = form.value
    const res = await api.post('/setup/apply', {
      password: f.password,
      backend_addr: f.backendAddr,
      listen_port: f.listenPort,
      iptables_enable: f.iptablesEnable,
      iptables_port: f.iptablesPort,
      ai_enabled: f.aiEnabled,
      api_key: f.apiKey,
      ai_model: f.aiModel,
      ai_base_url: f.aiBaseURL,
      rpm: f.rpm,
      burst_size: f.burstSize,
      ssh_enabled: f.sshEnabled,
      ssh_max_fails: f.sshMaxFails,
      ssh_ban_minutes: f.sshBanMinutes,
    })
    if (res.token) setAuthToken(res.token)
    ElMessage.success('配置完成，正在跳转...')
    setTimeout(() => router.push('/welcome'), 1000)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '配置应用失败')
  } finally {
    applying.value = false
  }
}
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f4f6fb 0%, #e8ecf4 100%);
  padding: 20px;
}

.setup-card {
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.06);
  padding: 40px;
  width: 100%;
  max-width: 680px;
}

.setup-header {
  text-align: center;
  margin-bottom: 32px;
}

.setup-logo {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  margin-bottom: 16px;
  box-shadow: 0 2px 12px rgba(99, 102, 241, 0.15);
}

.setup-header h1 {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
  margin-bottom: 6px;
}

.setup-subtitle {
  font-size: 14px;
  color: #94a3b8;
}

.steps-bar {
  margin-bottom: 32px;
}

.step-body {
  min-height: 260px;
  padding: 8px 0;
}

.step-body h2 {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 6px;
}

.step-desc {
  font-size: 13px;
  color: #94a3b8;
  margin-bottom: 24px;
}

.form-group {
  margin-bottom: 18px;
}

.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 6px;
}

.required {
  color: #e11d48;
}

.field-hint {
  display: block;
  font-size: 11.5px;
  color: #94a3b8;
  margin-top: 4px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.switch-label {
  font-size: 13px;
  color: #475569;
  margin-left: 10px;
  vertical-align: middle;
}

.section-divider {
  height: 1px;
  background: #eef0f4;
  margin: 20px 0;
}

.summary-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.summary-item {
  background: #f8f9fc;
  border-radius: 10px;
  padding: 14px 16px;
}

.summary-label {
  display: block;
  font-size: 11.5px;
  color: #94a3b8;
  margin-bottom: 4px;
}

.summary-item strong {
  font-size: 14px;
  color: #0f172a;
}

.text-green { color: #16a34a; }
.text-muted { color: #94a3b8; }

.step-nav {
  display: flex;
  align-items: center;
  margin-top: 32px;
  padding-top: 20px;
  border-top: 1px solid #eef0f4;
}

.nav-spacer {
  flex: 1;
}

@media (max-width: 640px) {
  .setup-card { padding: 24px 18px; }
  .form-row { grid-template-columns: 1fr; }
  .summary-grid { grid-template-columns: 1fr; }
}
</style>
