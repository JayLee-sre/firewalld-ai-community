package proxy

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var cookieSecret []byte

func init() {
	cookieSecret = make([]byte, 32)
	rand.Read(cookieSecret)
}

func signCookie(value string) string {
	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyCookie(cookie string) bool {
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		return false
	}
	ts := parts[0]
	sig := parts[1]

	expectedSig := signCookie(ts)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return false
	}

	// Check expiry (24h)
	var t int64
	fmt.Sscanf(ts, "%d", &t)
	if time.Now().Unix()-t > 86400 {
		return false
	}
	return true
}

func makeVerifiedCookie() string {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := signCookie(ts)
	return ts + "." + sig
}

func getCookieValue(req *http.Request, name string) string {
	for _, c := range req.Header["Cookie"] {
		parts := strings.Split(c, ";")
		for _, p := range parts {
			kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
			if len(kv) == 2 && kv[0] == name {
				return kv[1]
			}
		}
	}
	return ""
}

func isStaticAsset(path string) bool {
	for _, ext := range []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"} {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}

var challengeHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>安全检测 - ZhiYu-WAF</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"SF Pro Display","Segoe UI",Roboto,"PingFang SC","Microsoft YaHei",sans-serif;background:#06080f;min-height:100vh;display:flex;flex-direction:column;align-items:center;justify-content:center;color:#fff;overflow:hidden}
.bg{position:fixed;inset:0;overflow:hidden;pointer-events:none}
.bg::before{content:'';position:absolute;inset:0;background:radial-gradient(ellipse 80% 60% at 50% 0%,rgba(99,102,241,0.12) 0%,transparent 60%),radial-gradient(ellipse 60% 50% at 80% 100%,rgba(139,92,246,0.08) 0%,transparent 50%),radial-gradient(ellipse 50% 40% at 10% 60%,rgba(59,130,246,0.06) 0%,transparent 50%)}
.bg .orb{position:absolute;border-radius:50%;filter:blur(80px);opacity:.35;animation:drift 20s ease-in-out infinite alternate}
.bg .orb:nth-child(1){width:500px;height:500px;background:rgba(99,102,241,0.2);top:-10%;left:-5%}
.bg .orb:nth-child(2){width:400px;height:400px;background:rgba(139,92,246,0.15);bottom:-10%;right:-5%;animation-delay:-7s}
.bg .orb:nth-child(3){width:300px;height:300px;background:rgba(59,130,246,0.12);top:40%;left:60%;animation-delay:-13s}
@keyframes drift{0%{transform:translate(0,0) scale(1)}100%{transform:translate(40px,30px) scale(1.1)}}
.scanline{position:fixed;left:0;right:0;height:2px;background:linear-gradient(90deg,transparent,rgba(99,102,241,0.3),rgba(139,92,246,0.2),transparent);z-index:0;pointer-events:none;animation:scan 4s linear infinite;opacity:.6}
@keyframes scan{0%{top:-2px}100%{top:100vh}}
.grid-canvas{position:fixed;inset:0;pointer-events:none;opacity:.5}
.card{position:relative;z-index:1;text-align:center;padding:44px 40px 32px;width:440px;max-width:94vw;background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.06);border-radius:24px;backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);box-shadow:0 40px 80px rgba(0,0,0,0.4),inset 0 1px 0 rgba(255,255,255,0.05)}
.card::before{content:'';position:absolute;inset:-1px;border-radius:24px;padding:1px;background:linear-gradient(135deg,rgba(99,102,241,0.25),transparent 40%,transparent 60%,rgba(139,92,246,0.18));-webkit-mask:linear-gradient(#fff 0 0) content-box,linear-gradient(#fff 0 0);-webkit-mask-composite:xor;mask-composite:exclude;pointer-events:none}
.shield{width:100px;height:100px;margin:0 auto 24px;position:relative}
.shield .logo{width:100%;height:100%;border-radius:26px;filter:drop-shadow(0 0 40px rgba(99,102,241,0.45));position:relative;z-index:1;display:flex;align-items:center;justify-content:center;background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.06)}
.shield .logo svg{width:58px;height:58px}
.shield-ring{position:absolute;inset:-14px;border-radius:50%;border:1.5px solid rgba(99,102,241,0.2);animation:ring-pulse 2.5s ease-out infinite}
.shield-ring:nth-child(2){inset:-28px;border-color:rgba(99,102,241,0.1);animation-delay:.6s}
.shield-ring:nth-child(3){inset:-42px;border-color:rgba(139,92,246,0.06);animation-delay:1.2s}
@keyframes ring-pulse{0%{transform:scale(.85);opacity:1}100%{transform:scale(1.3);opacity:0}}
.hex-border{position:absolute;inset:-6px;border-radius:26px;z-index:0}
.hex-border::before{content:'';position:absolute;inset:0;border-radius:26px;border:2px solid transparent;background:conic-gradient(from 0deg,transparent,rgba(99,102,241,0.3),transparent,rgba(139,92,246,0.25),transparent) border-box;-webkit-mask:linear-gradient(#fff 0 0) padding-box,linear-gradient(#fff 0 0);-webkit-mask-composite:xor;mask-composite:exclude;animation:hex-spin 6s linear infinite}
@keyframes hex-spin{to{transform:rotate(360deg)}}
.brand{font-size:26px;font-weight:800;letter-spacing:-0.5px;margin-bottom:4px;background:linear-gradient(135deg,#a5b4fc 0%,#c4b5fd 50%,#818cf8 100%);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}
.sub{font-size:11px;color:rgba(255,255,255,0.25);letter-spacing:4px;text-transform:uppercase;margin-bottom:28px}
.features{display:flex;justify-content:center;gap:8px;flex-wrap:wrap;margin-bottom:28px}
.feat-tag{display:inline-flex;align-items:center;gap:4px;padding:4px 10px;border-radius:6px;font-size:10px;color:rgba(255,255,255,0.35);background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.06);letter-spacing:0.3px}
.feat-tag .dot{width:4px;height:4px;border-radius:50%;background:#818cf8}
.status{font-size:15px;color:rgba(255,255,255,0.6);margin-bottom:20px;min-height:24px;transition:color .3s}
.status .ok{color:#4ade80}
.progress-wrap{margin:0 auto 28px;position:relative}
.progress-bar{width:300px;height:3px;background:rgba(255,255,255,0.06);border-radius:3px;overflow:hidden;margin:0 auto}
.progress-fill{height:100%;width:0%;background:linear-gradient(90deg,#6366f1,#a78bfa,#818cf8);background-size:200% 100%;border-radius:3px;animation:fill-bar 2.2s cubic-bezier(.4,0,.2,1) forwards,shimmer 1.5s linear infinite}
@keyframes fill-bar{0%{width:0%}100%{width:100%}}
@keyframes shimmer{0%{background-position:200% 0}100%{background-position:-200% 0}}
.steps{display:flex;justify-content:center;gap:28px;margin-bottom:24px}
.step{display:flex;flex-direction:column;align-items:center;gap:8px;opacity:0;transform:translateY(8px);transition:all .4s cubic-bezier(.4,0,.2,1)}
.step.visible{opacity:1;transform:translateY(0)}
.step-dot{width:32px;height:32px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:14px;transition:all .4s;position:relative}
.step-dot.pending{background:rgba(255,255,255,0.04);border:1.5px solid rgba(255,255,255,0.08);color:rgba(255,255,255,0.2)}
.step-dot.active{background:rgba(99,102,241,0.15);border:1.5px solid rgba(99,102,241,0.4);color:#818cf8;box-shadow:0 0 16px rgba(99,102,241,0.2)}
.step-dot.done{background:rgba(34,197,94,0.12);border:1.5px solid rgba(34,197,94,0.3);color:#4ade80}
.step-label{font-size:10px;color:rgba(255,255,255,0.25);letter-spacing:0.5px;transition:color .3s;white-space:nowrap}
.step.visible .step-label{color:rgba(255,255,255,0.35)}
.step.done-step .step-label{color:rgba(74,222,128,0.6)}
.checks{text-align:left;display:inline-block;width:100%;padding:0 8px;margin-bottom:20px}
.check{font-size:13px;color:rgba(255,255,255,0.2);margin:10px 0;display:flex;align-items:center;gap:10px;opacity:0;transform:translateX(-12px);transition:all .35s cubic-bezier(.4,0,.2,1)}
.check.done{opacity:1;transform:translateX(0);color:rgba(74,222,128,0.8)}
.check.active{opacity:1;transform:translateX(0);color:rgba(255,255,255,0.6)}
.check-icon{width:18px;height:18px;display:inline-flex;align-items:center;justify-content:center;flex-shrink:0}
.spinner{width:14px;height:14px;border:2px solid rgba(99,102,241,0.15);border-top-color:#818cf8;border-radius:50%;animation:spin .7s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
.check .tick{font-size:14px;color:#4ade80}
.info-bar{display:flex;justify-content:center;gap:16px;margin-top:4px;padding-top:16px;border-top:1px solid rgba(255,255,255,0.04)}
.info-item{display:flex;align-items:center;gap:5px;font-size:10px;color:rgba(255,255,255,0.2)}
.info-val{color:rgba(255,255,255,0.4);font-family:"JetBrains Mono",monospace}
.bottom-bar{position:fixed;bottom:0;left:0;right:0;z-index:10;display:flex;align-items:center;justify-content:space-between;padding:14px 32px;background:linear-gradient(180deg,rgba(15,18,30,0.9) 0%,rgba(6,8,15,0.95) 100%);backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);border-top:1px solid rgba(99,102,241,0.08)}
.copyright{font-size:12px;color:rgba(255,255,255,0.4);display:flex;align-items:center;gap:10px;font-weight:500}
.copyright .brand-mini{display:inline-flex;align-items:center;gap:5px;color:rgba(165,180,252,0.7);font-weight:700}
.copyright .sep{color:rgba(255,255,255,0.12);margin:0 2px}
.cta-btn{display:inline-flex;align-items:center;gap:7px;padding:9px 20px;border-radius:10px;font-size:13px;font-weight:700;color:#fff;background:linear-gradient(135deg,#6366f1,#7c3aed);border:none;cursor:pointer;text-decoration:none;transition:all .25s;box-shadow:0 2px 16px rgba(99,102,241,0.3);letter-spacing:0.3px}
.cta-btn:hover{background:linear-gradient(135deg,#818cf8,#a78bfa);box-shadow:0 4px 24px rgba(99,102,241,0.45);transform:translateY(-1px)}
.cta-btn .pulse-dot{width:6px;height:6px;border-radius:50%;background:#4ade80;box-shadow:0 0 6px rgba(74,222,128,0.5);animation:cta-pulse 2s ease-in-out infinite}
@keyframes cta-pulse{0%,100%{opacity:1}50%{opacity:.4}}
@media(max-width:480px){.card{padding:32px 20px 28px}.shield{width:76px;height:76px}.brand{font-size:22px}.steps{gap:16px}.bottom-bar{padding:10px 16px;flex-direction:column;gap:8px}}
</style>
</head>
<body>
<div class="bg"><div class="orb"></div><div class="orb"></div><div class="orb"></div></div>
<div class="scanline"></div>
<canvas class="grid-canvas" id="gridCanvas"></canvas>
<div class="card">
  <div class="shield">
    <div class="hex-border"></div>
    <div class="shield-ring"></div><div class="shield-ring"></div><div class="shield-ring"></div>
    <div class="logo" aria-label="ZhiYu-WAF">
      <svg viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-hidden="true">
        <path d="M32 6L54 18V36C54 48 44 56 32 58C20 56 10 48 10 36V18L32 6Z" fill="url(#g)"/>
        <path d="M23 34L30 41L42 28" stroke="white" stroke-width="6" stroke-linecap="round" stroke-linejoin="round" opacity="0.9"/>
        <defs>
          <linearGradient id="g" x1="10" y1="6" x2="54" y2="58" gradientUnits="userSpaceOnUse">
            <stop stop-color="#6366F1"/>
            <stop offset="1" stop-color="#A78BFA"/>
          </linearGradient>
        </defs>
      </svg>
    </div>
  </div>
  <div class="brand">ZhiYu-WAF</div>
  <div class="sub">安全检测中</div>
  <div class="features">
    <span class="feat-tag"><span class="dot"></span>AI 智能防护</span>
    <span class="feat-tag"><span class="dot"></span>规则引擎</span>
    <span class="feat-tag"><span class="dot"></span>实时检测</span>
    <span class="feat-tag"><span class="dot"></span>零信任验证</span>
  </div>
  <div class="status" id="status">正在初始化安全检测</div>
  <div class="progress-wrap"><div class="progress-bar"><div class="progress-fill"></div></div></div>
  <div class="steps" id="stepsWrap">
    <div class="step" id="s1"><div class="step-dot pending">1</div><div class="step-label">环境检测</div></div>
    <div class="step" id="s2"><div class="step-dot pending">2</div><div class="step-label">指纹校验</div></div>
    <div class="step" id="s3"><div class="step-dot pending">3</div><div class="step-label">安全连接</div></div>
  </div>
  <div class="checks">
    <div class="check" id="c1"><span class="check-icon"><div class="spinner"></div></span> 正在验证浏览器环境</div>
    <div class="check" id="c2"><span class="check-icon"></span> 正在校验安全指纹</div>
    <div class="check" id="c3"><span class="check-icon"></span> 正在建立安全连接</div>
  </div>
  <div class="info-bar">
    <div class="info-item">TLS 1.3</div>
    <div class="info-item"><span class="info-val" id="clientIP">Protected</span></div>
    <div class="info-item">AES-256</div>
  </div>
</div>
<div class="bottom-bar">
  <div class="copyright">
    <span class="brand-mini">ZhiYu-WAF</span>
    <span class="sep">|</span>
    &copy; 2026 小睿科技 版权所有
  </div>
  <a class="cta-btn" href="#" target="_blank" rel="noopener">
    <span class="pulse-dot"></span>
    获取同款 WAF 系统
  </a>
</div>
<script>
var checks=['c1','c2','c3'],steps=['s1','s2','s3'];
var statusEl=document.getElementById('status');
var msgs=['正在验证浏览器环境','正在校验安全指纹','正在建立安全连接','安全检测通过'];
steps.forEach(function(id,i){setTimeout(function(){document.getElementById(id).classList.add('visible')},200+i*150)});
function runChecks(){
  checks.forEach(function(id,i){
    setTimeout(function(){
      if(i>0){
        document.getElementById(checks[i-1]).className='check done';
        document.getElementById(checks[i-1]).querySelector('.check-icon').innerHTML='<span class="tick">&#10003;</span>';
        document.getElementById(steps[i-1]).querySelector('.step-dot').className='step-dot done';
        document.getElementById(steps[i-1]).querySelector('.step-dot').innerHTML='&#10003;';
        document.getElementById(steps[i-1]).classList.add('done-step');
      }
      var el=document.getElementById(id);
      el.className='check active';
      el.querySelector('.check-icon').innerHTML='<div class="spinner"></div>';
      document.getElementById(steps[i]).querySelector('.step-dot').className='step-dot active';
      statusEl.innerHTML=msgs[i];
    },i*700);
  });
  setTimeout(function(){
    document.getElementById(checks[2]).className='check done';
    document.getElementById(checks[2]).querySelector('.check-icon').innerHTML='<span class="tick">&#10003;</span>';
    document.getElementById(steps[2]).querySelector('.step-dot').className='step-dot done';
    document.getElementById(steps[2]).querySelector('.step-dot').innerHTML='&#10003;';
    document.getElementById(steps[2]).classList.add('done-step');
    statusEl.innerHTML=msgs[3]+' <span class="ok">&#10003;</span>';
    setTimeout(doVerify,500);
  },checks.length*700);
}
function doVerify(){
  var xhr=new XMLHttpRequest();
  xhr.open('POST','/__zhiyu_waf_verify',true);
  xhr.setRequestHeader('Content-Type','application/json');
  xhr.onload=function(){if(xhr.status===200)location.reload();};
  xhr.send('{}');
}
setTimeout(runChecks,600);
</script>
</body>
</html>`
