<template>
  <div class="login-page">
    <!-- 装饰 -->
    <div class="deco deco-1"></div>
    <div class="deco deco-2"></div>
    <div class="deco deco-3"></div>

    <div class="login-wrap">
      <!-- 左侧品牌区 -->
      <div class="brand-panel">
        <div class="brand-inner">
          <div class="brand-logo">
            <img src="/logo.png" alt="智域 WAF" />
          </div>
          <h1 class="brand-name">智域 WAF</h1>
          <p class="brand-desc">AI 驱动的新一代<br/>智能 Web 应用防火墙</p>
          <div class="features">
            <div class="feat">
              <div class="feat-icon fi-1">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
              </div>
              <div><b>智能防护</b><span>规则 + AI 双引擎检测</span></div>
            </div>
            <div class="feat">
              <div class="feat-icon fi-2">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              </div>
              <div><b>全球覆盖</b><span>地理封锁 + 威胁情报</span></div>
            </div>
            <div class="feat">
              <div class="feat-icon fi-3">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
              </div>
              <div><b>极速响应</b><span>毫秒级检测 &lt; 5ms</span></div>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧表单 -->
      <div class="form-panel">
        <div class="form-inner">
          <div class="form-header">
            <h2>欢迎回来</h2>
            <p>登录控制台管理您的安全策略</p>
          </div>

          <el-form :model="form" :rules="rules" ref="formRef" @submit.prevent="handleLogin" class="login-form">
            <el-form-item prop="username">
              <label class="input-label">用户名</label>
              <el-input v-model="form.username" placeholder="请输入管理员用户名" size="large" :prefix-icon="User" />
            </el-form-item>
            <el-form-item prop="password">
              <label class="input-label">密码</label>
              <el-input v-model="form.password" type="password" placeholder="请输入登录密码" size="large" :prefix-icon="Lock" show-password />
            </el-form-item>

            <div class="terms-row" :class="{ active: form.acceptedTerms }" @click="form.acceptedTerms = !form.acceptedTerms">
              <span class="check-mark">
                <svg v-if="form.acceptedTerms" viewBox="0 0 16 16"><path d="M3.2 8.2 6.5 11.4 12.8 4.6" /></svg>
              </span>
              <span>我已阅读并同意<a href="/agreement.html" target="_blank" rel="noopener" @click.stop>用户协议</a>和<a href="/privacy.html" target="_blank" rel="noopener" @click.stop>隐私政策</a></span>
            </div>

            <el-button type="primary" size="large" :loading="loading" :disabled="!form.acceptedTerms" class="login-btn" @click="handleLogin">
              {{ loading ? '验证中...' : '登 录' }}
            </el-button>
          </el-form>

          <div class="form-footer">
            <span class="dot"></span>
            安全会话 · 操作留痕 · 数据加密
          </div>
        </div>
      </div>
    </div>

    <div class="page-footer">© 2026 小睿科技 版权所有</div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import api, { setAuthToken } from '../api'

const router = useRouter()
const formRef = ref()
const loading = ref(false)
const form = reactive({ username: '', password: '', acceptedTerms: false })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  if (!form.acceptedTerms) {
    ElMessage.warning('请先阅读并勾选同意用户协议和隐私政策')
    return
  }
  await formRef.value.validate()
  loading.value = true
  try {
    const { username, password } = form
    const res = await api.post('/auth/login', { username, password }, { suppressError: true })
    setAuthToken(res.token)
    ElMessage.success('登录成功')
    // Check if setup wizard is needed
    try {
      const status = await api.get('/setup/status', { suppressError: true })
      if (status?.needed) {
        router.push('/setup')
        return
      }
    } catch {}
    router.push('/dashboard')
  } catch {
    ElMessage.error('用户名或密码错误')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #f4f6fb;
  position: relative;
  overflow: hidden;
}

/* 装饰圆 */
.deco {
  position: absolute;
  border-radius: 50%;
  opacity: 0.5;
  filter: blur(80px);
  pointer-events: none;
}
.deco-1 {
  width: 500px; height: 500px;
  background: rgba(99, 102, 241, 0.12);
  top: -150px; right: -100px;
}
.deco-2 {
  width: 400px; height: 400px;
  background: rgba(59, 130, 246, 0.08);
  bottom: -100px; left: -80px;
}
.deco-3 {
  width: 300px; height: 300px;
  background: rgba(139, 92, 246, 0.06);
  top: 50%; left: 50%;
  transform: translate(-50%, -50%);
}

/* 主卡片 */
.login-wrap {
  display: flex;
  width: 820px;
  max-width: 96vw;
  min-height: 480px;
  background: #fff;
  border-radius: 20px;
  box-shadow:
    0 4px 6px -1px rgba(0, 0, 0, 0.04),
    0 20px 50px -12px rgba(0, 0, 0, 0.08);
  overflow: hidden;
  border: 1px solid rgba(226, 232, 240, 0.8);
  position: relative;
  z-index: 1;
}

/* 左侧品牌 */
.brand-panel {
  flex: 1;
  background: linear-gradient(145deg, #6366f1 0%, #4f46e5 40%, #7c3aed 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 32px;
  position: relative;
  overflow: hidden;
}
.brand-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 30% 20%, rgba(255,255,255,0.12) 0%, transparent 40%),
    radial-gradient(circle at 80% 80%, rgba(255,255,255,0.06) 0%, transparent 40%);
}
.brand-inner {
  position: relative;
  text-align: center;
  color: #fff;
}
.brand-logo {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  border-radius: 16px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}
.brand-logo img { width: 100%; height: 100%; object-fit: cover; display: block; }
.brand-name {
  font-size: 28px;
  font-weight: 800;
  margin: 0 0 8px;
  letter-spacing: -0.5px;
}
.brand-desc {
  font-size: 14px;
  line-height: 1.6;
  opacity: 0.85;
  margin: 0 0 28px;
}

.features { display: flex; flex-direction: column; gap: 14px; text-align: left; }
.feat {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(4px);
  border: 1px solid rgba(255, 255, 255, 0.12);
}
.feat-icon {
  width: 34px; height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.15);
}
.feat-icon svg { width: 18px; height: 18px; color: #fff; }
.feat b { display: block; font-size: 13px; font-weight: 700; color: #fff; }
.feat span { font-size: 11px; color: rgba(255, 255, 255, 0.7); margin-top: 1px; display: block; }

/* 右侧表单 */
.form-panel {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 36px;
}
.form-inner { width: 100%; max-width: 320px; }
.form-header { margin-bottom: 28px; }
.form-header h2 {
  font-size: 24px;
  font-weight: 800;
  color: #0f172a;
  margin: 0 0 6px;
  letter-spacing: -0.5px;
}
.form-header p {
  font-size: 13px;
  color: #94a3b8;
  margin: 0;
}

.login-form :deep(.el-form-item) { margin-bottom: 18px; }

.input-label {
  display: block;
  margin-bottom: 6px;
  color: #475569;
  font-size: 12px;
  font-weight: 700;
}

.login-form :deep(.el-input__wrapper) {
  border-radius: 10px;
  box-shadow: 0 0 0 1px #e2e8f0;
  background: #f8fafc;
  transition: all 0.2s;
  padding: 1px 11px;
}

.login-form :deep(.el-input__inner) { color: #1e293b; }
.login-form :deep(.el-input__inner::placeholder) { color: #94a3b8; }
.login-form :deep(.el-input__prefix),
.login-form :deep(.el-input__suffix) { color: #94a3b8; }

.login-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #cbd5e1;
}

.login-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.4);
  background: #fff;
}

/* 协议 */
.terms-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  margin-bottom: 20px;
  border-radius: 10px;
  cursor: pointer;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 12.5px;
  color: #64748b;
  transition: all 0.2s;
}
.terms-row:hover { border-color: #c7d2fe; background: #f5f3ff; }
.terms-row.active { border-color: #a5b4fc; background: #eef2ff; }
.terms-row a { color: #6366f1; text-decoration: none; font-weight: 600; }
.terms-row a:hover { text-decoration: underline; }

.check-mark {
  flex-shrink: 0;
  width: 18px; height: 18px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 5px;
  background: #fff;
  border: 1.5px solid #cbd5e1;
  transition: all 0.2s;
}
.terms-row.active .check-mark {
  background: #6366f1;
  border-color: #6366f1;
}
.check-mark svg { width: 14px; height: 14px; }
.check-mark path {
  fill: none; stroke: #fff;
  stroke-width: 2.5; stroke-linecap: round; stroke-linejoin: round;
}

/* 登录按钮 */
.login-btn {
  width: 100%;
  height: 44px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, #6366f1, #4f46e5);
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 2px;
  transition: all 0.2s;
}
.login-btn:hover:not(.is-disabled) {
  background: linear-gradient(135deg, #4f46e5, #4338ca);
  transform: translateY(-1px);
  box-shadow: 0 6px 16px rgba(99, 102, 241, 0.3);
}
.login-btn.is-disabled,
.login-btn.is-disabled:hover {
  background: #e2e8f0;
  color: #94a3b8;
  transform: none;
  box-shadow: none;
}

/* 底部 */
.form-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 24px;
  font-size: 12px;
  color: #94a3b8;
}
.dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 6px rgba(34, 197, 94, 0.4);
  animation: pulse-dot 2s infinite;
}
@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.page-footer {
  margin-top: 20px;
  font-size: 11px;
  color: #94a3b8;
  position: relative;
  z-index: 1;
}

/* 响应式 */
@media (max-width: 768px) {
  .login-wrap {
    flex-direction: column;
    min-height: auto;
    max-width: 92vw;
  }
  .brand-panel {
    padding: 28px 24px;
  }
  .features { display: none; }
  .brand-desc { margin-bottom: 0; }
  .brand-name { font-size: 24px; }
  .form-panel { padding: 28px 24px; }
  .form-inner { max-width: 100%; }
}
</style>
