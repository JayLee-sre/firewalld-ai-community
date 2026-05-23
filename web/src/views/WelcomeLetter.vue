<template>
  <div class="welcome-page">
    <div class="welcome-bg">
      <div class="bg-circle c1"></div>
      <div class="bg-circle c2"></div>
      <div class="bg-circle c3"></div>
      <div class="bg-circle c4"></div>
    </div>

    <!-- 顶部导航栏 -->
    <div class="top-nav">
      <div class="nav-inner">
        <div class="nav-brand">
          <img src="/logo.png" alt="智域 WAF" class="nav-logo" />
          <span class="nav-name">智域 WAF</span>
        </div>
        <div class="nav-badge">
          <span class="badge-dot"></span>
          <span>by 小睿科技</span>
        </div>
      </div>
    </div>

    <!-- Hero 区域 -->
    <div class="hero-section">
      <div class="hero-badge">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
        <span>写给每一位新朋友的信</span>
      </div>
      <h1 class="hero-title">感谢你选择我们</h1>
      <p class="hero-subtitle">在这个产品多到数不清的时代，你愿意花时间试试我们做的东西，这件事本身就值得我们好好珍惜。</p>
    </div>

    <!-- 信件卡片 -->
    <div class="letter-card">
      <div class="letter-header">
        <div class="letter-avatar">
          <img src="/logo.png" alt="智域 WAF" />
        </div>
        <div class="letter-meta">
          <span class="letter-from">小睿科技</span>
          <span class="letter-date">{{ today }}</span>
        </div>
        <div class="letter-tag">致你的一封信</div>
      </div>

      <div class="letter-body">
        <p class="letter-greeting">亲爱的朋友，你好：</p>

        <p>
          看到你打开了这个页面，说实话我们特别开心。在这个产品多到数不清的时代，你愿意花时间试试我们做的东西，这件事本身就值得我们好好珍惜。
        </p>

        <p>
          智域 WAF 是我们一点点做出来的。没有风投，没有豪华团队，有的就是几台电脑、无数杯咖啡，和一个很朴素的想法——
        </p>

        <div class="vision-banner">
          <div class="vision-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="28" height="28"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
          </div>
          <p class="vision-text">让安全防护不再是少数人才能用得起的东西。</p>
        </div>

        <p>
          选择做 WAF，是因为我们看到太多小团队、个人开发者，面对攻击时束手无策。大厂的安全产品动辄上万，配置复杂到让人头秃。我们想，AI 都能写代码了，为什么不能帮普通人守住自己的网站？
        </p>
      </div>

      <!-- 三个核心价值 -->
      <div class="values-section">
        <div class="value-card">
          <div class="value-icon v1">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="22" height="22"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
          </div>
          <h3>用心打磨</h3>
          <p>每一行代码都是认认真真写的，没有敷衍，没有凑数</p>
        </div>
        <div class="value-card">
          <div class="value-icon v2">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="22" height="22"><circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/></svg>
          </div>
          <h3>真诚相待</h3>
          <p>不画饼、不忽悠，有问题就改，有反馈就听</p>
        </div>
        <div class="value-card">
          <div class="value-icon v3">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="22" height="22"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
          </div>
          <h3>持续进化</h3>
          <p>产品还很年轻，但我们在不停迭代，每天都在变好</p>
        </div>
      </div>

      <!-- 坦诚告白 -->
      <div class="letter-body">
        <div class="highlight-card">
          <div class="highlight-badge">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0zM12 9v4M12 17h.01"/></svg>
            <span>坦诚告白</span>
          </div>
          <p>
            我们也清楚，这个产品还很稚嫩。<b>你可能会碰到 bug，会觉得某个功能还不够好，会对着某些提示一头雾水。</b>这些我们心里有数，也在不停地改。如果你愿意把遇到的问题告诉我们，哪怕只是一句"这里不太对"，对我们来说都是最宝贵的反馈。
          </p>
        </div>

        <p>
          我们不敢说这个产品有多完美，但我们敢说，<b>每一行代码都是认认真真写的</b>。你的每一次使用，都在帮我们变得更好。
        </p>

        <p>
          未来的路还很长，但你来了，我们就有了继续走下去的理由。
        </p>

        <p class="letter-closing">
          谢谢你愿意给我们一个机会。<br/>
          我们会努力不辜负这份信任。
        </p>
      </div>

      <!-- 签名区 -->
      <div class="signature-area">
        <div class="sig-line"></div>
        <div class="sig-content">
          <div class="sig-left">
            <span class="sig-team">小睿科技</span>
            <span class="sig-sub">智域 WAF 开发团队</span>
          </div>
          <div class="sig-right">
            <span class="sig-date">{{ today }}</span>
            <span class="sig-location">小睿科技</span>
          </div>
        </div>
      </div>

      <!-- 底部统计 -->
      <div class="stats-bar">
        <div class="stat-item">
          <span class="stat-num">100%</span>
          <span class="stat-label">开源精神</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-num">24/7</span>
          <span class="stat-label">在线守护</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-num">&lt;5ms</span>
          <span class="stat-label">检测速度</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-num">0元</span>
          <span class="stat-label">社区版永久免费</span>
        </div>
      </div>

      <button class="enter-btn" @click="enter">
        <span>进入控制台</span>
        <span class="btn-arrow">&rarr;</span>
      </button>
    </div>

    <!-- 底部 -->
    <div class="page-footer">
      <span>&copy; 2026 小睿科技 版权所有</span>
      <span class="footer-dot"></span>
      <span>用爱与代码守护每一份安全</span>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()
const today = new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })

function enter() {
  router.push('/dashboard')
}
</script>

<style scoped>
.welcome-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: #f4f6fb;
  padding: 0 24px 40px;
  position: relative;
  overflow: hidden;
}

.welcome-bg {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

.bg-circle {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.3;
}
.bg-circle.c1 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, #a78bfa 0%, transparent 70%);
  top: -150px; right: -100px;
}
.bg-circle.c2 {
  width: 400px; height: 400px;
  background: radial-gradient(circle, #60a5fa 0%, transparent 70%);
  bottom: -120px; left: -80px;
}
.bg-circle.c3 {
  width: 300px; height: 300px;
  background: radial-gradient(circle, #f9a8d4 0%, transparent 70%);
  top: 40%; left: 50%;
  transform: translate(-50%, -50%);
  opacity: 0.15;
}
.bg-circle.c4 {
  width: 350px; height: 350px;
  background: radial-gradient(circle, #34d399 0%, transparent 70%);
  top: 20%; left: 10%;
  opacity: 0.1;
}

/* ===== 顶部导航 ===== */
.top-nav {
  width: 100%;
  max-width: 720px;
  padding: 20px 0 0;
  position: relative;
  z-index: 1;
}
.nav-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.nav-brand {
  display: flex;
  align-items: center;
  gap: 10px;
}
.nav-logo {
  width: 32px; height: 32px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.15);
}
.nav-name {
  font-size: 16px;
  font-weight: 800;
  color: #0f172a;
  letter-spacing: -0.5px;
}
.nav-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #94a3b8;
  padding: 4px 12px;
  background: rgba(255,255,255,0.8);
  border-radius: 20px;
  border: 1px solid #eef0f4;
}
.badge-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 6px rgba(34, 197, 94, 0.5);
}

/* ===== Hero ===== */
.hero-section {
  text-align: center;
  padding: 48px 0 36px;
  max-width: 620px;
  position: relative;
  z-index: 1;
  animation: fadeIn 0.6s ease-out;
}
.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  background: linear-gradient(135deg, #fef2f2, #fdf2f8);
  border: 1px solid #fecdd3;
  border-radius: 20px;
  font-size: 12.5px;
  color: #e11d48;
  font-weight: 600;
  margin-bottom: 20px;
}
.hero-badge svg { color: #f43f5e; }
.hero-title {
  font-size: 36px;
  font-weight: 900;
  color: #0f172a;
  margin-bottom: 16px;
  background: linear-gradient(135deg, #6366f1, #a78bfa, #f472b6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: -1px;
  line-height: 1.2;
}
.hero-subtitle {
  font-size: 15px;
  color: #64748b;
  line-height: 1.7;
  max-width: 480px;
  margin: 0 auto;
}

/* ===== 信件卡片 ===== */
.letter-card {
  position: relative;
  z-index: 1;
  background: #fff;
  border-radius: 20px;
  box-shadow: 0 8px 40px rgba(0, 0, 0, 0.06), 0 1px 3px rgba(0, 0, 0, 0.04);
  padding: 0;
  width: 100%;
  max-width: 720px;
  overflow: hidden;
  animation: cardIn 0.6s ease-out 0.2s both;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(24px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 信件头部 */
.letter-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 24px 36px;
  border-bottom: 1px solid #f1f5f9;
  background: #fafbff;
}
.letter-avatar {
  width: 42px; height: 42px;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.12);
  flex-shrink: 0;
}
.letter-avatar img { width: 100%; height: 100%; object-fit: cover; }
.letter-meta {
  display: flex;
  flex-direction: column;
  flex: 1;
}
.letter-from {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
}
.letter-date {
  font-size: 11.5px;
  color: #94a3b8;
  margin-top: 1px;
}
.letter-tag {
  font-size: 11px;
  color: #6366f1;
  background: #eef2ff;
  padding: 3px 10px;
  border-radius: 12px;
  font-weight: 600;
}

/* 信件正文 */
.letter-body {
  padding: 32px 36px;
  line-height: 1.85;
  color: #334155;
  font-size: 14.5px;
}
.letter-greeting {
  font-size: 15px;
  color: #475569;
  margin-bottom: 16px;
  font-weight: 500;
}
.letter-body p {
  margin-bottom: 16px;
}
.letter-body strong, .letter-body b {
  color: #1e293b;
  font-weight: 700;
}

/* 愿景横幅 */
.vision-banner {
  background: linear-gradient(135deg, #eef2ff 0%, #f5f3ff 50%, #faf5ff 100%);
  border-radius: 14px;
  padding: 24px 28px;
  margin: 24px 0;
  display: flex;
  align-items: center;
  gap: 18px;
  border: 1px solid #e0e7ff;
}
.vision-icon {
  width: 52px; height: 52px;
  border-radius: 14px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #fff;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}
.vision-text {
  font-size: 16px;
  font-weight: 700;
  color: #4338ca;
  line-height: 1.6;
  margin: 0;
}

/* 三个价值 */
.values-section {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  padding: 0 36px 28px;
}
.value-card {
  text-align: center;
  padding: 24px 16px;
  background: #f8f9fc;
  border-radius: 14px;
  border: 1px solid #eef0f4;
  transition: all 0.2s;
}
.value-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0,0,0,0.06);
}
.value-icon {
  width: 46px; height: 46px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 14px;
}
.value-icon.v1 { background: #eef2ff; color: #6366f1; }
.value-icon.v2 { background: #fef2f2; color: #e11d48; }
.value-icon.v3 { background: #fffbeb; color: #d97706; }
.value-card h3 {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 6px;
}
.value-card p {
  font-size: 12px;
  color: #64748b;
  line-height: 1.6;
  margin: 0;
}

/* 坦诚告白卡片 */
.highlight-card {
  background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
  border-radius: 14px;
  padding: 22px 24px;
  margin: 8px 0 20px;
  border: 1px solid #fde68a;
}
.highlight-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 700;
  color: #92400e;
}
.highlight-badge svg { color: #f59e0b; }
.highlight-card p {
  margin: 0;
  font-size: 14px;
  color: #78350f;
  line-height: 1.8;
}

.letter-closing {
  margin-top: 24px;
  color: #475569;
  font-style: italic;
  font-size: 15px;
}

/* 签名区 */
.signature-area {
  padding: 0 36px 28px;
}
.sig-line {
  height: 1px;
  background: linear-gradient(90deg, transparent, #e2e8f0, transparent);
  margin-bottom: 20px;
}
.sig-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
}
.sig-left, .sig-right {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.sig-team {
  font-size: 16px;
  font-weight: 800;
  color: #0f172a;
}
.sig-sub {
  font-size: 12px;
  color: #94a3b8;
}
.sig-right { align-items: flex-end; }
.sig-date {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}
.sig-location {
  font-size: 11px;
  color: #94a3b8;
}

/* 统计条 */
.stats-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0;
  padding: 24px 36px;
  background: #f8f9fc;
  border-top: 1px solid #eef0f4;
}
.stat-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}
.stat-num {
  font-size: 20px;
  font-weight: 800;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.stat-label {
  font-size: 11px;
  color: #94a3b8;
  font-weight: 500;
}
.stat-divider {
  width: 1px;
  height: 32px;
  background: #e2e8f0;
}

/* 进入按钮 */
.enter-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: calc(100% - 72px);
  margin: 0 36px 32px;
  padding: 16px 24px;
  border: none;
  border-radius: 14px;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.25s ease;
  box-shadow: 0 4px 14px rgba(99, 102, 241, 0.3);
}
.enter-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.4);
}
.enter-btn:active {
  transform: translateY(0);
}
.btn-arrow {
  font-size: 20px;
  transition: transform 0.2s;
}
.enter-btn:hover .btn-arrow {
  transform: translateX(4px);
}

/* 页脚 */
.page-footer {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 32px;
  font-size: 12px;
  color: #94a3b8;
  position: relative;
  z-index: 1;
}
.footer-dot {
  width: 3px; height: 3px;
  border-radius: 50%;
  background: #cbd5e1;
}

/* ===== 响应式 ===== */
@media (max-width: 640px) {
  .welcome-page { padding: 0 12px 32px; }
  .hero-title { font-size: 26px; }
  .hero-subtitle { font-size: 14px; }
  .letter-header { padding: 18px 20px; }
  .letter-body { padding: 24px 20px; }
  .values-section { grid-template-columns: 1fr; padding: 0 20px 20px; }
  .signature-area { padding: 0 20px 20px; }
  .stats-bar { padding: 20px 16px; flex-wrap: wrap; gap: 12px; }
  .stat-divider { display: none; }
  .stat-item { min-width: 40%; }
  .enter-btn { width: calc(100% - 40px); margin: 0 20px 24px; }
  .vision-banner { flex-direction: column; text-align: center; }
  .sig-content { flex-direction: column; gap: 12px; }
  .sig-right { align-items: flex-start; }
}
</style>
