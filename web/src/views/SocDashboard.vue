<template>
  <!-- 非专业版 -->
  <div class="pro-gate" v-if="!isPro">
    <div class="gate-card">
      <div class="gate-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="48" height="48"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      </div>
      <h2>专业版功能</h2>
      <p>监控大屏是专业版专属功能，升级后即可使用安全态势可视化大屏。</p>
      <router-link to="/settings" class="gate-btn">升级专业版</router-link>
      <router-link to="/dashboard" class="gate-back">返回管理面板</router-link>
    </div>
  </div>

  <!-- 专业版大屏 -->
  <div class="soc" v-else>
    <header class="topbar">
      <div class="tl">
        <router-link to="/dashboard" class="back">← 管理面板</router-link>
        <div class="br">
          <div class="b-icon">W</div>
          <div>
            <div class="b-title">智域 WAF 监控大屏</div>
            <div class="b-sub">Security Operations Center</div>
          </div>
        </div>
      </div>
      <div class="tc">
        <span class="pill" :class="healthData.status === 'ok' ? 'ok' : 'err'">
          <em></em>{{ healthData.status === 'ok' ? '系统正常' : '异常' }}
        </span>
        <span class="pill" :class="healthData.ai_enabled ? 'ai' : 'off'">
          <em></em>AI {{ healthData.ai_enabled ? 'ON' : 'OFF' }}
        </span>
        <span class="pill blue"><em></em>{{ ruleCount }} 规则</span>
      </div>
      <div class="tr">
        <span class="clock">{{ currentTime }}</span>
      </div>
    </header>

    <!-- 统计 -->
    <div class="kpi">
      <div class="kpi-item" v-for="s in kpiCards" :key="s.k">
        <div class="kpi-val" :style="{ color: s.c }">{{ s.v }}</div>
        <div class="kpi-lbl">{{ s.l }}</div>
      </div>
    </div>

    <!-- 三栏 -->
    <div class="grid">
      <!-- 左 -->
      <div class="col">
        <div class="card fx">
          <div class="ch">威胁等级</div>
          <div class="ct" ref="sevRef"></div>
          <div class="legend" v-if="sevT > 0">
            <div class="leg" v-for="s in sevLeg" :key="s.k">
              <i :style="{ background: s.c }"></i><span>{{ s.n }}</span><b>{{ s.p }}%</b>
            </div>
          </div>
        </div>
        <div class="card fx">
          <div class="ch">检测来源</div>
          <div class="ct" ref="srcRef"></div>
        </div>
      </div>

      <!-- 中 -->
      <div class="col col-c">
        <div class="card fx">
          <div class="ch">全球攻击来源 <span class="cnt">{{ wRegs.length }} 国</span></div>
          <div class="cm" ref="wRef"></div>
        </div>
        <div class="card fx">
          <div class="ch">国内攻击来源 <span class="cnt">{{ cRegs.length }} 省</span></div>
          <div class="cm" ref="cRef"></div>
        </div>
      </div>

      <!-- 右 -->
      <div class="col">
        <div class="card fx">
          <div class="ch">攻击来源 TOP</div>
          <div class="rl">
            <div class="ri" v-for="(r,i) in topRegions.slice(0,10)" :key="i">
              <span class="rnk" :class="{hot:i<3}">{{i+1}}</span>
              <span class="rnm">{{r.region}}</span>
              <div class="rbr"><div class="rfl" :style="{width:rbW(r.count)}"></div></div>
              <span class="rcn">{{r.count}}</span>
            </div>
            <div class="empty" v-if="!topRegions.length">暂无数据</div>
          </div>
        </div>
        <div class="card fx">
          <div class="ch">24h 趋势</div>
          <div class="ct" ref="trRef"></div>
        </div>
        <div class="card fx guard-card">
          <div class="ch">防护状态</div>
          <div class="gd">
            <div class="gd-i"><b>{{sshStats.failed||0}}</b><span>SSH 失败</span></div>
            <div class="gd-i"><b>{{sshStats.blocked||0}}</b><span>SSH 封禁</span></div>
            <div class="gd-i"><b>{{threatCount}}</b><span>威胁IP</span></div>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick, inject } from 'vue'
import * as echarts from 'echarts'
import api from '../api'

const isPro = inject('isPro', ref(false))

const stats = ref({}), healthData = ref({}), sshStats = ref({}), threatInfo = ref({})
const ruleCount = ref(0), currentTime = ref('')

let clk
function tick() { currentTime.value = new Date().toLocaleString('zh-CN', { hour12: false }) }

const kpiCards = computed(() => {
  const s = stats.value, t = s.total_requests||0, b = s.blocked_count||0, a = s.ai_count||0
  return [
    { k:'t', l:'总检测量', v:t.toLocaleString(), c:'#3b82f6' },
    { k:'b', l:'拦截次数', v:b.toLocaleString(), c:'#ef4444' },
    { k:'a', l:'AI 检出', v:a.toLocaleString(), c:'#8b5cf6' },
    { k:'r', l:'来源地区', v:String((s.top_regions||[]).length), c:'#10b981' },
    { k:'p', l:'拦截率', v:t?Math.round(b/t*100)+'%':'0%', c:'#f59e0b' },
  ]
})
const topRegions = computed(() => stats.value.top_regions||[])
const threatCount = computed(() => (threatInfo.value.threat_ips||[]).length || threatInfo.value.ip_count || 0)

const CN_PROVS = ['北京','上海','广东','深圳','浙江','江苏','四川','湖北','湖南','福建','山东','河南','河北','安徽','辽宁','陕西','重庆','云南','广西','山西','贵州','江西','黑龙江','吉林','甘肃','内蒙古','新疆','海南','宁夏','青海','西藏','天津','中国台湾','中国香港','中国澳门']
const cRegs = computed(() => topRegions.value.filter(r => CN_PROVS.some(p => r.region.includes(p) || p.includes(r.region))))
const wRegs = computed(() => topRegions.value.filter(r => !CN_PROVS.some(p => r.region.includes(p) || p.includes(r.region))))

const sevT = computed(() => { const s = stats.value.by_severity||{}; return Object.values(s).reduce((a,b)=>a+b,0) })
const sevLeg = computed(() => {
  const s = stats.value.by_severity||{}, t = sevT.value||1
  return [
    { k:'critical', n:'严重', c:'#ef4444', p:Math.round((s.critical||0)/t*100) },
    { k:'high', n:'高危', c:'#f59e0b', p:Math.round((s.high||0)/t*100) },
    { k:'medium', n:'中危', c:'#eab308', p:Math.round((s.medium||0)/t*100) },
    { k:'low', n:'低危', c:'#22c55e', p:Math.round((s.low||0)/t*100) },
  ]
})

function rbW(c) { const m = topRegions.value[0]?.count||1; return Math.max(6,Math.round(c/m*100))+'%' }

// Charts
const sevRef=ref(null), srcRef=ref(null), wRef=ref(null), cRef=ref(null), trRef=ref(null)
let sevC,srcC,wC,cC,trC,ro

const darkTip = { backgroundColor:'rgba(15,23,42,0.92)', borderColor:'rgba(99,102,241,0.2)', textStyle:{color:'#e2e8f0',fontSize:12}, borderRadius:8 }

function initSev(){
  if(!sevRef.value) return; sevC=echarts.init(sevRef.value); updSev()
}
function updSev(){
  if(!sevC) return
  const src=stats.value.by_severity||{}, colors={critical:'#ef4444',high:'#f59e0b',medium:'#eab308',low:'#22c55e'}
  const labels={critical:'严重',high:'高危',medium:'中危',low:'低危'}
  const data=['critical','high','medium','low'].map(k=>({name:labels[k],value:src[k]||0,itemStyle:{color:colors[k]}})).filter(d=>d.value>0)
  const total=data.reduce((s,d)=>s+d.value,0)
  sevC.setOption({
    tooltip:{...darkTip, trigger:'item'},
    series:[{type:'pie',radius:['48%','74%'],center:['50%','44%'],itemStyle:{borderRadius:6,borderColor:'#fff',borderWidth:3},label:{show:false},emphasis:{scaleSize:5},
      data:data.length?data:[{name:'暂无',value:1,itemStyle:{color:'#f1f5f9'}}]}],
    graphic:total?[{type:'group',left:'center',top:'36%',children:[
      {type:'text',style:{text:String(total),fontSize:22,fontWeight:800,fill:'#0f172a',textAlign:'center'},left:'center',top:-12},
      {type:'text',style:{text:'事件',fontSize:11,fill:'#94a3b8',textAlign:'center'},left:'center',top:12},
    ]}]:[],
  },true)
}

function initSrc(){
  if(!srcRef.value) return; srcC=echarts.init(srcRef.value); updSrc()
}
function updSrc(){
  if(!srcC) return
  const s=stats.value.by_source||{}, rv=s.rule_engine||s.rule||0, ai=s.ai||0
  srcC.setOption({
    tooltip:{...darkTip, trigger:'item'},
    series:[{type:'pie',radius:['40%','68%'],center:['50%','50%'],itemStyle:{borderRadius:5,borderColor:'#fff',borderWidth:2},
      label:{show:true,position:'outside',formatter:'{b}\n{d}%',fontSize:11,color:'#64748b',lineHeight:15},labelLine:{lineStyle:{color:'#cbd5e1'}},
      data:(rv||ai)?[{name:'规则引擎',value:rv,itemStyle:{color:'#6366f1'}},{name:'AI 分析',value:ai,itemStyle:{color:'#8b5cf6'}}]
        :[{name:'暂无',value:1,itemStyle:{color:'#f1f5f9'}}]}],},true)
}

function initTrend(){
  if(!trRef.value) return; trC=echarts.init(trRef.value); updTrend()
}
function updTrend(){
  if(!trC) return
  const hours=[],now=new Date()
  for(let i=23;i>=0;i--){const d=new Date(now-i*36e5);hours.push(d.getHours()+':00')}
  trC.setOption({
    tooltip:{...darkTip, trigger:'axis'},
    grid:{top:10,right:8,bottom:20,left:28},
    xAxis:{type:'category',data:hours,boundaryGap:false,axisLine:{lineStyle:{color:'#e2e8f0'}},axisLabel:{color:'#94a3b8',fontSize:9,interval:5}},
    yAxis:{type:'value',splitLine:{lineStyle:{color:'#f1f5f9'}},axisLabel:{color:'#94a3b8',fontSize:9}},
    series:[{type:'line',data:new Array(24).fill(0),smooth:true,symbol:'none',lineStyle:{color:'#6366f1',width:2},
      areaStyle:{color:new echarts.graphic.LinearGradient(0,0,0,1,[{offset:0,color:'rgba(99,102,241,0.3)'},{offset:1,color:'rgba(99,102,241,0.02)'}])}}],},true)
}

// Maps
let mapsOk=false
async function loadMaps(){
  if(mapsOk) return
  try{
    const[w,c]=await Promise.all([fetch('/world.json').then(r=>r.json()),fetch('/china.json').then(r=>r.json())])
    echarts.registerMap('world',w); echarts.registerMap('china',c); mapsOk=true
  }catch(e){console.warn('Map load failed:',e)}
}
function initW(){if(!wRef.value)return;wC=echarts.init(wRef.value);updW()}
function updW(){
  if(!wC||!mapsOk) return
  const data=wRegs.value.map(r=>{const c=wCoord(r.region);return c[0]?{name:r.region,value:[...c,r.count]}:null}).filter(Boolean)
  const lines=data.map(d=>({coords:[d.value.slice(0,2),[104,35]],lineStyle:{color:'rgba(239,68,68,0.15)',width:1,curveness:0.3}}))
  wC.setOption({
    tooltip:{...darkTip,trigger:'item',formatter:p=>p.value&&p.value[2]?`<b>${p.name}</b><br/>攻击 ${p.value[2]} 次`:p.name},
    geo:{map:'world',roam:true,zoom:1.2,center:[20,20],scaleLimit:{min:1,max:8},
      itemStyle:{areaColor:'#eef2f7',borderColor:'#cbd5e1',borderWidth:0.6},
      emphasis:{itemStyle:{areaColor:'#dbeafe'},label:{show:true,fontSize:11,color:'#1e293b',fontWeight:700}},
      label:{show:true,fontSize:8,color:'#64748b',fontWeight:500}},
    series:[
      {type:'scatter',coordinateSystem:'geo',data,symbol:'circle',
        symbolSize:v=>Math.max(6,Math.min(24,Math.sqrt(v[2])*3.5)),
        itemStyle:{color:'#ef4444',shadowBlur:8,shadowColor:'rgba(239,68,68,0.3)',opacity:0.85},
        label:{show:true,formatter:p=>`${p.name} ${p.value[2]}`,position:'right',color:'#475569',fontSize:9,distance:4}},
      {type:'lines',coordinateSystem:'geo',data:lines,effect:{show:true,period:5,trailLength:0.2,symbol:'circle',symbolSize:3,color:'#ef4444'},lineStyle:{width:0,curveness:0.3}},
    ],},true)
}

function initC(){if(!cRef.value)return;cC=echarts.init(cRef.value);updC()}
function updC(){
  if(!cC||!mapsOk) return
  const data=cRegs.value.map(r=>{const c=cCoord(r.region);return c[0]?{name:r.region,value:[...c,r.count]}:null}).filter(Boolean)
  cC.setOption({
    tooltip:{...darkTip,trigger:'item',formatter:p=>p.value&&p.value[2]?`<b>${p.name}</b><br/>攻击 ${p.value[2]} 次`:p.name},
    geo:{map:'china',roam:true,zoom:1.15,scaleLimit:{min:1,max:8},
      itemStyle:{areaColor:'#eef2f7',borderColor:'#cbd5e1',borderWidth:0.6},
      emphasis:{itemStyle:{areaColor:'#dbeafe'},label:{show:true,fontSize:12,color:'#1e293b',fontWeight:700}},
      label:{show:true,fontSize:10,color:'#475569',fontWeight:600}},
    series:[{type:'scatter',coordinateSystem:'geo',data,symbol:'circle',
      symbolSize:v=>Math.max(8,Math.min(28,Math.sqrt(v[2])*4)),
      itemStyle:{color:'#6366f1',shadowBlur:8,shadowColor:'rgba(99,102,241,0.3)',opacity:0.85},
      label:{show:true,formatter:p=>`${p.name} ${p.value[2]}`,position:'right',color:'#334155',fontSize:10,fontWeight:600,distance:4}}],},true)
}

function wCoord(r){
  const m={'美国':[-95,38],'日本':[138,36],'韩国':[127,37],'印度':[78,21],'俄罗斯':[100,60],'德国':[10,51],'英国':[-3,55],'法国':[2,46],
    '巴西':[-51,-14],'加拿大':[-106,56],'澳大利亚':[133,-25],'荷兰':[5,52],'新加坡':[103,1],'印度尼西亚':[118,-1],'泰国':[100,15],
    '越南':[108,14],'菲律宾':[122,13],'伊朗':[53,32],'土耳其':[35,39],'意大利':[12,42],'西班牙':[-4,40],'波兰':[20,52],
    '乌克兰':[32,49],'墨西哥':[-102,23],'南非':[23,-30],'尼日利亚':[8,10],'埃及':[30,27],'沙特阿拉伯':[45,24],
    '以色列':[35,31],'马来西亚':[102,4],'中国台湾':[121,24],'中国香港':[114,22],'朝鲜':[127,40],'阿根廷':[-63,-34]}
  for(const[k,c] of Object.entries(m)){if(r.includes(k)||k.includes(r))return c}
  return[0,0]
}
function cCoord(r){
  const m={'北京':[116.4,39.9],'上海':[121.5,31.2],'广东':[113.3,23.1],'深圳':[114.1,22.5],'浙江':[120.2,30.3],'江苏':[118.8,32.1],
    '四川':[104.1,30.6],'湖北':[114.3,30.6],'湖南':[112.9,28.2],'福建':[119.3,26.1],'山东':[117.0,36.7],'河南':[113.7,34.8],
    '河北':[114.5,38.0],'安徽':[117.3,31.8],'辽宁':[123.4,41.8],'陕西':[108.9,34.3],'重庆':[106.6,29.6],'云南':[102.7,25.0],
    '广西':[108.3,22.8],'山西':[112.5,37.9],'贵州':[106.7,26.6],'江西':[115.9,28.7],'黑龙江':[126.7,45.8],'吉林':[125.3,43.9],
    '甘肃':[103.8,36.1],'内蒙古':[111.7,40.8],'新疆':[87.6,43.8],'海南':[110.3,20.0],'宁夏':[106.3,38.5],'青海':[101.8,36.6],
    '西藏':[91.1,29.7],'天津':[117.2,39.1],'中国台湾':[121.5,25],'中国香港':[114.2,22.3],'中国澳门':[113.5,22.2]}
  for(const[k,c] of Object.entries(m)){if(r.includes(k)||k.includes(r))return c}
  return[0,0]
}

async function loadAll(){
  const[st,h,ssh,ti,rl]=await Promise.allSettled([
    api.get('/stats'),api.get('/health'),api.get('/ssh/stats'),api.get('/threatintel/status'),api.get('/rules')])
  if(st.status==='fulfilled')stats.value=st.value||{}
  if(h.status==='fulfilled')healthData.value=h.value||{}
  if(ssh.status==='fulfilled')sshStats.value=ssh.value||{}
  if(ti.status==='fulfilled')threatInfo.value=ti.value||{}
  if(rl.status==='fulfilled')ruleCount.value=Array.isArray(rl.value)?rl.value.length:0
}

onMounted(async()=>{
  tick(); clk=setInterval(tick,1000)
  await loadAll(); await loadMaps(); await nextTick()
  initSev(); initSrc(); initW(); initC(); initTrend()
  ro=new ResizeObserver(()=>{sevC?.resize();srcC?.resize();wC?.resize();cC?.resize();trC?.resize()})
  ;[sevRef,srcRef,wRef,cRef,trRef].forEach(r=>{if(r.value)ro.observe(r.value)})
  setInterval(loadAll,30000)
})
watch(()=>stats.value.by_severity,()=>updSev(),{deep:true})
watch(()=>stats.value.by_source,()=>updSrc(),{deep:true})
watch(wRegs,()=>updW(),{deep:true})
watch(cRegs,()=>updC(),{deep:true})
onBeforeUnmount(()=>{clearInterval(clk);ro?.disconnect();[sevC,srcC,wC,cC,trC].forEach(c=>c?.dispose())})
</script>

<style scoped>
/* Pro Gate */
.pro-gate{min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f4f6fb}
.gate-card{text-align:center;background:#fff;border:1px solid #e8ecf1;border-radius:20px;padding:48px 40px;max-width:400px;box-shadow:0 4px 6px -1px rgba(0,0,0,0.04),0 20px 50px -12px rgba(0,0,0,0.06)}
.gate-icon{width:72px;height:72px;border-radius:18px;background:#eef2ff;color:#6366f1;display:flex;align-items:center;justify-content:center;margin:0 auto 20px}
.gate-card h2{font-size:22px;font-weight:800;color:#0f172a;margin:0 0 8px}
.gate-card p{font-size:13px;color:#94a3b8;line-height:1.6;margin:0 0 24px}
.gate-btn{display:inline-block;padding:10px 32px;border-radius:10px;background:linear-gradient(135deg,#6366f1,#4f46e5);color:#fff;font-size:14px;font-weight:700;text-decoration:none;transition:all .2s}
.gate-btn:hover{transform:translateY(-1px);box-shadow:0 6px 16px rgba(99,102,241,0.3)}
.gate-back{display:block;margin-top:14px;font-size:13px;color:#94a3b8;text-decoration:none}
.gate-back:hover{color:#6366f1}

.soc{height:100vh;display:flex;flex-direction:column;background:#f4f6fb;font-family:'Inter',-apple-system,'Microsoft YaHei',sans-serif;color:#1e293b;overflow:hidden}

/* Topbar */
.topbar{display:flex;justify-content:space-between;align-items:center;padding:0 24px;height:52px;background:#fff;border-bottom:1px solid #e8ecf1;flex-shrink:0}
.tl{display:flex;align-items:center;gap:14px}
.back{font-size:12px;color:#94a3b8;text-decoration:none;padding:5px 12px;border-radius:8px;border:1px solid #e2e8f0;transition:all .2s}
.back:hover{color:#6366f1;border-color:#c7d2fe;background:#eef2ff}
.br{display:flex;align-items:center;gap:10px}
.b-icon{width:30px;height:30px;border-radius:8px;background:linear-gradient(135deg,#6366f1,#8b5cf6);color:#fff;font-weight:800;font-size:13px;display:flex;align-items:center;justify-content:center;box-shadow:0 2px 8px rgba(99,102,241,.2)}
.b-title{font-size:15px;font-weight:800;color:#0f172a}
.b-sub{font-size:10px;color:#94a3b8;letter-spacing:.5px}
.tc{display:flex;gap:6px}
.tr{display:flex;align-items:center}
.clock{font-size:13px;font-weight:700;color:#334155;font-variant-numeric:tabular-nums}
.pill{display:flex;align-items:center;gap:4px;padding:3px 10px;border-radius:999px;font-size:11px;font-weight:600;background:#f1f5f9;color:#64748b}
.pill em{width:5px;height:5px;border-radius:50%;background:currentColor;font-style:normal}
.pill.ok{color:#10b981;background:#ecfdf5}.pill.ok em{box-shadow:0 0 4px #10b981}
.pill.err{color:#ef4444;background:#fef2f2}
.pill.ai{color:#8b5cf6;background:#f5f3ff}
.pill.off{color:#94a3b8}
.pill.blue{color:#3b82f6;background:#eff6ff}

/* KPI */
.kpi{display:grid;grid-template-columns:repeat(5,1fr);gap:12px;padding:12px 24px 0;flex-shrink:0}
.kpi-item{background:#fff;border:1px solid #e8ecf1;border-radius:12px;padding:14px 18px;text-align:center}
.kpi-val{font-size:26px;font-weight:800;line-height:1;letter-spacing:-1px}
.kpi-lbl{font-size:12px;color:#94a3b8;font-weight:600;margin-top:4px}

/* Grid */
.grid{flex:1;display:grid;grid-template-columns:1fr 1.8fr 1fr;gap:12px;padding:12px 24px;min-height:0;overflow:hidden}
.col{display:flex;flex-direction:column;gap:10px;min-height:0;overflow:hidden}

/* Card */
.card{background:#fff;border:1px solid #e8ecf1;border-radius:12px;overflow:hidden;display:flex;flex-direction:column}
.ch{display:flex;align-items:center;gap:8px;padding:9px 14px;font-size:13px;font-weight:700;color:#0f172a;border-bottom:1px solid #f1f5f9;flex-shrink:0}
.cnt{margin-left:auto;font-size:10px;color:#6366f1;font-weight:600;background:#eef2ff;padding:2px 8px;border-radius:999px}
.ct{flex:1;min-height:60px}
.cm{flex:1;min-height:60px}
.fx{flex:1}
.guard-card{flex-shrink:0}

/* Legend */
.legend{display:grid;grid-template-columns:1fr 1fr;gap:2px 8px;padding:4px 14px 8px;flex-shrink:0}
.leg{display:flex;align-items:center;gap:5px;font-size:11px;color:#64748b}
.leg i{width:7px;height:7px;border-radius:2px;flex-shrink:0;font-style:normal}
.leg b{margin-left:auto;color:#0f172a}

/* Region */
.rl{padding:4px 10px;overflow-y:auto;flex:1}
.ri{display:grid;grid-template-columns:18px 56px 1fr 30px;align-items:center;gap:5px;padding:4px 2px;font-size:11px}
.rnk{width:17px;height:17px;border-radius:4px;display:flex;align-items:center;justify-content:center;background:#f1f5f9;color:#94a3b8;font-size:10px;font-weight:800}
.rnk.hot{background:#fef2f2;color:#ef4444}
.rnm{color:#334155;font-weight:600;font-size:11px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.rbr{height:4px;border-radius:99px;background:#f1f5f9;overflow:hidden}
.rfl{height:100%;border-radius:inherit;background:linear-gradient(90deg,#6366f1,#ef4444);transition:width .6s}
.rcn{color:#94a3b8;text-align:right;font-weight:700;font-size:10px}

/* Guard */
.gd{display:grid;grid-template-columns:1fr 1fr 1fr;gap:6px;padding:8px 10px}
.gd-i{text-align:center;padding:6px;background:#f8fafc;border-radius:8px}
.gd-i b{display:block;font-size:18px;font-weight:800;color:#0f172a}
.gd-i span{font-size:10px;color:#94a3b8;font-weight:600}

.empty{text-align:center;padding:16px;color:#cbd5e1;font-size:12px}

::-webkit-scrollbar{width:4px}::-webkit-scrollbar-track{background:transparent}::-webkit-scrollbar-thumb{background:#e2e8f0;border-radius:9px}

@media(max-width:1200px){
  .grid{grid-template-columns:1fr 1fr}.col-c{grid-column:1/-1;flex-direction:row}.tc{display:none}
  .kpi{padding:8px 12px 0}.grid{padding:8px 12px}
}
@media(max-width:768px){
  .topbar{padding:0 12px}.b-sub{display:none}.kpi{grid-template-columns:repeat(3,1fr);gap:8px}
  .kpi-val{font-size:20px}.grid{grid-template-columns:1fr;padding:8px 10px}.col-c{flex-direction:column}
}
</style>
