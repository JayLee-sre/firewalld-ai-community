<template>
  <!-- 专业版引导 -->
  <div class="pro-gate" v-if="!isPro">
    <div class="gate-card">
      <div class="gate-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="48" height="48"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      </div>
      <h2>专业版功能</h2>
      <p>地理封锁是专业版专属功能，支持按国家/地区精确屏蔽访问来源，升级后即可使用。</p>
      <div class="gate-features">
        <div class="gate-feat"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="20 6 9 17 4 12"/></svg> 按国家/地区一键封锁</div>
        <div class="gate-feat"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="20 6 9 17 4 12"/></svg> 封锁与放行双重策略</div>
        <div class="gate-feat"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="20 6 9 17 4 12"/></svg> 实时生效无需重启</div>
      </div>
      <router-link to="/settings" class="gate-btn">升级专业版</router-link>
      <router-link to="/dashboard" class="gate-back">返回管理面板</router-link>
    </div>
  </div>

  <div class="geo-page" v-else>
    <div class="page-toolbar">
      <div class="heading-group">
        <div class="heading-icon indigo"><el-icon :size="18"><Location /></el-icon></div>
        <div>
          <div class="page-heading">地理封锁</div>
          <div class="page-sub">按国家/地区屏蔽访问来源，增强安全防护</div>
        </div>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stat-row">
      <div class="stat-card">
        <div class="stat-icon blocked"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/></svg></div>
        <div class="stat-info">
          <div class="stat-num">{{ blockedCount }}</div>
          <div class="stat-label">已封锁</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon allowed"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg></div>
        <div class="stat-info">
          <div class="stat-num">{{ allowedCount }}</div>
          <div class="stat-label">已放行</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon total"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg></div>
        <div class="stat-info">
          <div class="stat-num">{{ rules.length }}</div>
          <div class="stat-label">总规则数</div>
        </div>
      </div>
    </div>

    <!-- 添加区域 -->
    <div class="add-section">
      <div class="add-header">
        <h3>添加封锁规则</h3>
        <div class="action-toggle">
          <button class="toggle-btn" :class="{ active: newAction === 'block' }" @click="newAction = 'block'">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px"><circle cx="12" cy="12" r="10"/><line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/></svg>
            封锁
          </button>
          <button class="toggle-btn" :class="{ active: newAction === 'allow' }" @click="newAction = 'allow'">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            放行
          </button>
        </div>
      </div>

      <!-- 搜索 -->
      <div class="search-bar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="search-icon"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input v-model="searchQuery" placeholder="搜索国家/地区..." class="search-input" />
      </div>

      <!-- 区域分组 -->
      <div class="region-grid">
        <div class="region-group" v-for="region in filteredRegions" :key="region.name">
          <div class="region-label">{{ region.name }}</div>
          <div class="country-chips">
            <button
              v-for="c in region.countries" :key="c.name"
              class="country-chip"
              :class="{ selected: selectedCountry === c.name, disabled: isExisting(c.name) }"
              @click="selectCountry(c.name)"
              :disabled="isExisting(c.name)"
            >
              <span class="chip-flag">{{ c.flag }}</span>
              <span class="chip-name">{{ c.name }}</span>
              <span v-if="isExisting(c.name)" class="chip-badge">{{ getExistingAction(c.name) }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 确认添加 -->
      <div class="confirm-bar" v-if="selectedCountry">
        <div class="confirm-info">
          <span class="confirm-flag">{{ getFlag(selectedCountry) }}</span>
          <span>将 <strong>{{ selectedCountry }}</strong> 加入<strong :class="newAction === 'block' ? 'text-block' : 'text-allow'">{{ newAction === 'block' ? '封锁' : '放行' }}</strong>列表</span>
        </div>
        <div class="confirm-actions">
          <button class="btn-ghost" @click="selectedCountry = ''">取消</button>
          <button class="btn-primary" @click="addRule">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            确认添加
          </button>
        </div>
      </div>
    </div>

    <!-- 已有规则 -->
    <div class="rules-section">
      <div class="rules-header">
        <h3>当前规则 <span class="rule-count">{{ rules.length }}</span></h3>
        <div class="filter-tabs" v-if="rules.length > 0">
          <button class="filter-tab" :class="{ active: filterAction === 'all' }" @click="filterAction = 'all'">全部</button>
          <button class="filter-tab" :class="{ active: filterAction === 'block' }" @click="filterAction = 'block'">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:12px;height:12px"><circle cx="12" cy="12" r="10"/><line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/></svg>
            封锁
          </button>
          <button class="filter-tab" :class="{ active: filterAction === 'allow' }" @click="filterAction = 'allow'">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:12px;height:12px"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            放行
          </button>
        </div>
      </div>

      <div class="rules-grid" v-if="filteredRules.length > 0">
        <div class="rule-card" v-for="rule in filteredRules" :key="rule.id" :class="{ disabled: !rule.enabled }">
          <div class="rule-top">
            <span class="rule-flag">{{ getFlag(rule.country) }}</span>
            <span class="rule-action-badge" :class="rule.action">{{ rule.action === 'block' ? '封锁' : '放行' }}</span>
          </div>
          <div class="rule-country">{{ rule.country }}</div>
          <div class="rule-code">{{ rule.country_code || getCode(rule.country) }}</div>
          <div class="rule-bottom">
            <span class="rule-time">{{ formatTime(rule.created_at) }}</span>
            <div class="rule-actions">
              <button class="icon-btn" :class="rule.enabled ? 'disable' : 'enable'" @click="toggleRule(rule)" :title="rule.enabled ? '禁用' : '启用'">
                <svg v-if="rule.enabled" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              </button>
              <button class="icon-btn delete" @click="removeRule(rule.id)" title="删除">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="empty-state" v-else-if="rules.length === 0 && !loading">
        <div class="empty-illustration">
          <svg viewBox="0 0 120 120" fill="none">
            <circle cx="60" cy="60" r="50" fill="#eef2ff"/>
            <circle cx="60" cy="60" r="35" stroke="#c7d2fe" stroke-width="2" stroke-dasharray="6 4"/>
            <path d="M60 30a30 30 0 1 1 0 60 30 30 0 0 1 0-60z" stroke="#818cf8" stroke-width="2"/>
            <line x1="30" y1="60" x2="90" y2="60" stroke="#c7d2fe" stroke-width="1.5"/>
            <ellipse cx="60" cy="60" rx="14" ry="30" stroke="#c7d2fe" stroke-width="1.5"/>
            <path d="M38 45h22M38 75h22" stroke="#c7d2fe" stroke-width="1.5"/>
          </svg>
        </div>
        <div class="empty-text">暂无地理封锁规则</div>
        <div class="empty-desc">在上方选择国家/地区来添加封锁或放行规则</div>
      </div>

      <div class="empty-state small" v-else-if="filteredRules.length === 0 && rules.length > 0">
        <div class="empty-text">没有{{ filterAction === 'block' ? '封锁' : '放行' }}规则</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, inject } from 'vue'
import { Location } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const isPro = inject('isPro', ref(false))

const rules = ref([]), loading = ref(false)
const newAction = ref('block')
const selectedCountry = ref('')
const searchQuery = ref('')
const filterAction = ref('all')

const blockedCount = computed(() => rules.value.filter(r => r.action === 'block' && r.enabled).length)
const allowedCount = computed(() => rules.value.filter(r => r.action === 'allow' && r.enabled).length)

const filteredRules = computed(() => {
  if (filterAction.value === 'all') return rules.value
  return rules.value.filter(r => r.action === filterAction.value)
})

const countryData = [
  { name: '东亚', countries: [
    { name: '中国', flag: ' ' }, { name: '日本', flag: ' ' }, { name: '韩国', flag: ' ' },
    { name: '中国台湾', flag: ' ' }, { name: '中国香港', flag: '  ' }, { name: '中国澳门', flag: '  ' },
    { name: '朝鲜', flag: ' ' },
  ]},
  { name: '东南亚', countries: [
    { name: '新加坡', flag: ' ' }, { name: '马来西亚', flag: ' ' }, { name: '泰国', flag: ' ' },
    { name: '越南', flag: ' ' }, { name: '菲律宾', flag: ' ' }, { name: '印度尼西亚', flag: ' ' },
  ]},
  { name: '南亚/中亚', countries: [
    { name: '印度', flag: ' ' }, { name: '沙特阿拉伯', flag: ' ' }, { name: '以色列', flag: ' ' },
    { name: '伊朗', flag: ' ' }, { name: '土耳其', flag: ' ' },
  ]},
  { name: '欧洲', countries: [
    { name: '俄罗斯', flag: ' ' }, { name: '德国', flag: ' ' }, { name: '英国', flag: ' ' },
    { name: '法国', flag: ' ' }, { name: '荷兰', flag: ' ' }, { name: '意大利', flag: ' ' },
    { name: '西班牙', flag: ' ' }, { name: '波兰', flag: ' ' }, { name: '乌克兰', flag: ' ' },
  ]},
  { name: '北美洲', countries: [
    { name: '美国', flag: ' ' }, { name: '加拿大', flag: ' ' }, { name: '墨西哥', flag: ' ' },
  ]},
  { name: '南美洲', countries: [
    { name: '巴西', flag: ' ' }, { name: '阿根廷', flag: ' ' }, { name: '哥伦比亚', flag: ' ' },
  ]},
  { name: '大洋洲', countries: [
    { name: '澳大利亚', flag: ' ' },
  ]},
  { name: '非洲', countries: [
    { name: '南非', flag: ' ' }, { name: '尼日利亚', flag: ' ' }, { name: '埃及', flag: ' ' },
  ]},
]

const flagMap = {}
countryData.forEach(r => r.countries.forEach(c => { flagMap[c.name] = c.flag }))

const filteredRegions = computed(() => {
  if (!searchQuery.value) return countryData
  const q = searchQuery.value.toLowerCase()
  return countryData.map(r => ({
    ...r,
    countries: r.countries.filter(c => c.name.toLowerCase().includes(q))
  })).filter(r => r.countries.length > 0)
})

function getFlag(name) { return flagMap[name] || '  ' }
function getCode(name) {
  const map = { '美国':'US','日本':'JP','韩国':'KR','印度':'IN','俄罗斯':'RU','巴西':'BR','德国':'DE','英国':'GB','法国':'FR','加拿大':'CA','澳大利亚':'AU','荷兰':'NL','新加坡':'SG','印度尼西亚':'ID','泰国':'TH','越南':'VN','菲律宾':'PH','伊朗':'IR','朝鲜':'KP','土耳其':'TR','意大利':'IT','西班牙':'ES','波兰':'PL','乌克兰':'UA','墨西哥':'MX','阿根廷':'AR','哥伦比亚':'CO','南非':'ZA','尼日利亚':'NG','埃及':'EG','沙特阿拉伯':'SA','以色列':'IL','马来西亚':'MY','中国台湾':'TW','中国香港':'HK','中国':'CN','中国澳门':'MO' }
  return map[name] || ''
}

function isExisting(name) { return rules.value.some(r => r.country === name) }
function getExistingAction(name) {
  const r = rules.value.find(r => r.country === name)
  return r ? (r.action === 'block' ? '已封锁' : '已放行') : ''
}
function formatTime(ts) { return ts ? new Date(ts).toLocaleString('zh-CN') : '-' }

function selectCountry(name) {
  if (isExisting(name)) return
  selectedCountry.value = selectedCountry.value === name ? '' : name
}

async function loadRules() {
  loading.value = true
  try {
    const res = await api.get('/geo/rules') || []
    rules.value = res
  } catch {} finally { loading.value = false }
}

async function addRule() {
  const country = selectedCountry.value.trim()
  if (!country) { ElMessage.warning('请选择国家/地区'); return }
  try {
    await api.post('/geo/rules', { country, action: newAction.value })
    ElMessage.success('添加成功')
    selectedCountry.value = ''
    loadRules()
  } catch {}
}

async function toggleRule(rule) {
  try {
    await api.put(`/geo/rules/${rule.id}`, { country: rule.country, action: rule.action, enabled: !rule.enabled })
    ElMessage.success(rule.enabled ? '已禁用' : '已启用')
    loadRules()
  } catch {}
}

async function removeRule(id) {
  try {
    await ElMessageBox.confirm('确定删除此地理封锁规则？', '确认', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
    await api.delete(`/geo/rules/${id}`)
    ElMessage.success('已删除')
    loadRules()
  } catch {}
}

onMounted(loadRules)
</script>

<style scoped>
.geo-page { max-width: 1100px; }

/* Pro Gate */
.pro-gate { display: flex; align-items: center; justify-content: center; min-height: 60vh; }
.gate-card {
  text-align: center; max-width: 420px; padding: 48px 40px;
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-lg, 20px);
  box-shadow: 0 4px 24px rgba(0,0,0,.06);
}
.gate-icon { margin-bottom: 20px; color: #d97706; }
.gate-card h2 { font-size: 22px; font-weight: 800; color: var(--text-primary); margin-bottom: 10px; }
.gate-card p { font-size: 14px; color: var(--text-muted); line-height: 1.6; margin-bottom: 20px; }
.gate-features { text-align: left; margin-bottom: 24px; }
.gate-feat {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 0; font-size: 13px; color: var(--text-secondary); font-weight: 500;
}
.gate-feat svg { color: #059669; flex-shrink: 0; }
.gate-btn {
  display: inline-block; padding: 10px 28px; border-radius: 10px;
  background: var(--primary); color: #fff; font-size: 14px; font-weight: 700;
  text-decoration: none; transition: all 0.2s;
}
.gate-btn:hover { background: var(--primary-hover); }
.gate-back {
  display: block; margin-top: 12px; font-size: 13px; color: var(--text-muted);
  text-decoration: none;
}
.gate-back:hover { color: var(--primary); }

/* 统计卡片 */
.stat-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 20px; }
.stat-card {
  display: flex; align-items: center; gap: 14px;
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  padding: 16px 18px;
}
.stat-icon { width: 40px; height: 40px; border-radius: 10px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.stat-icon svg { width: 20px; height: 20px; }
.stat-icon.blocked { background: #fff1f2; color: #e11d48; }
.stat-icon.allowed { background: #ecfdf5; color: #059669; }
.stat-icon.total { background: #eef2ff; color: #4f46e5; }
.stat-num { font-size: 22px; font-weight: 700; color: var(--text-primary); line-height: 1.2; }
.stat-label { font-size: 12px; color: var(--text-muted); }

/* 添加区域 */
.add-section {
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  padding: 20px; margin-bottom: 20px;
}
.add-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.add-header h3 { font-size: 15px; font-weight: 700; color: var(--text-primary); }
.action-toggle { display: flex; gap: 4px; background: var(--bg-subtle); border-radius: 8px; padding: 3px; }
.toggle-btn {
  display: flex; align-items: center; gap: 5px;
  padding: 6px 14px; border-radius: 6px; border: none;
  font-size: 13px; font-weight: 600; cursor: pointer;
  background: transparent; color: var(--text-muted); transition: all 0.2s;
}
.toggle-btn.active { background: #fff; color: var(--text-primary); box-shadow: 0 1px 3px rgba(0,0,0,.08); }

/* 搜索 */
.search-bar { position: relative; margin-bottom: 16px; }
.search-icon { position: absolute; left: 12px; top: 50%; transform: translateY(-50%); width: 16px; height: 16px; color: var(--text-muted); }
.search-input {
  width: 100%; padding: 9px 12px 9px 36px; border-radius: 8px;
  border: 1px solid var(--border); background: var(--bg-subtle);
  font-size: 13px; color: var(--text-primary); outline: none; transition: border-color 0.2s;
}
.search-input:focus { border-color: var(--primary); }
.search-input::placeholder { color: var(--text-muted); }

/* 区域分组 */
.region-grid { display: flex; flex-direction: column; gap: 14px; max-height: 380px; overflow-y: auto; }
.region-label { font-size: 11px; font-weight: 700; text-transform: uppercase; color: var(--text-muted); letter-spacing: 0.5px; margin-bottom: 6px; }
.country-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.country-chip {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 6px 12px; border-radius: 8px; border: 1px solid var(--border);
  background: #fff; font-size: 13px; cursor: pointer; transition: all 0.15s;
  color: var(--text-primary);
}
.country-chip:hover:not(.disabled) { border-color: var(--primary); background: var(--primary-light); }
.country-chip.selected { border-color: var(--primary); background: #eef2ff; box-shadow: 0 0 0 2px rgba(79,70,229,.15); }
.country-chip.disabled { opacity: 0.5; cursor: default; background: var(--bg-subtle); }
.chip-flag { font-size: 16px; line-height: 1; }
.chip-name { font-weight: 500; }
.chip-badge { font-size: 10px; font-weight: 700; padding: 1px 6px; border-radius: 4px; background: var(--bg-subtle); color: var(--text-muted); }

/* 确认栏 */
.confirm-bar {
  display: flex; align-items: center; justify-content: space-between;
  margin-top: 16px; padding: 12px 16px; border-radius: 10px;
  background: #eef2ff; border: 1px solid #c7d2fe;
}
.confirm-info { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--text-primary); }
.confirm-flag { font-size: 20px; }
.text-block { color: #e11d48; }
.text-allow { color: #059669; }
.confirm-actions { display: flex; gap: 8px; }
.btn-ghost {
  padding: 6px 14px; border-radius: 7px; border: 1px solid var(--border);
  background: #fff; font-size: 13px; font-weight: 600; cursor: pointer; color: var(--text-mid); transition: all 0.2s;
}
.btn-ghost:hover { border-color: var(--primary); color: var(--primary); }
.btn-primary {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 6px 16px; border-radius: 7px; border: none;
  background: var(--primary); color: #fff; font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.2s;
}
.btn-primary:hover { background: var(--primary-hover); }

/* 规则区域 */
.rules-section {
  background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius-card);
  padding: 20px;
}
.rules-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.rules-header h3 { font-size: 15px; font-weight: 700; color: var(--text-primary); display: flex; align-items: center; gap: 8px; }
.rule-count {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 22px; height: 22px; padding: 0 6px; border-radius: 6px;
  background: #eef2ff; color: var(--primary); font-size: 12px; font-weight: 700;
}
.filter-tabs { display: flex; gap: 4px; }
.filter-tab {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 5px 12px; border-radius: 6px; border: 1px solid var(--border);
  background: #fff; font-size: 12px; font-weight: 600; cursor: pointer; color: var(--text-muted); transition: all 0.2s;
}
.filter-tab.active { background: var(--primary); color: #fff; border-color: var(--primary); }
.filter-tab:hover:not(.active) { border-color: var(--primary); color: var(--primary); }

/* 规则卡片网格 */
.rules-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
.rule-card {
  padding: 14px; border-radius: 10px; border: 1px solid var(--border);
  background: #fff; transition: all 0.2s;
}
.rule-card:hover { box-shadow: 0 2px 8px rgba(0,0,0,.06); }
.rule-card.disabled { opacity: 0.55; }
.rule-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.rule-flag { font-size: 28px; line-height: 1; }
.rule-action-badge { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 5px; }
.rule-action-badge.block { background: #fff1f2; color: #e11d48; }
.rule-action-badge.allow { background: #ecfdf5; color: #059669; }
.rule-country { font-size: 15px; font-weight: 700; color: var(--text-primary); margin-bottom: 2px; }
.rule-code { font-size: 11px; color: var(--text-muted); font-family: 'SF Mono', 'Menlo', monospace; letter-spacing: 0.5px; margin-bottom: 10px; }
.rule-bottom { display: flex; align-items: center; justify-content: space-between; }
.rule-time { font-size: 11px; color: var(--text-muted); }
.rule-actions { display: flex; gap: 4px; }
.icon-btn {
  width: 28px; height: 28px; border-radius: 6px; border: none;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; transition: all 0.2s; background: var(--bg-subtle); color: var(--text-muted);
}
.icon-btn svg { width: 14px; height: 14px; }
.icon-btn.disable:hover { background: #fff1f2; color: #e11d48; }
.icon-btn.enable:hover { background: #ecfdf5; color: #059669; }
.icon-btn.delete:hover { background: #fff1f2; color: #e11d48; }

/* 空状态 */
.empty-state { text-align: center; padding: 40px 20px; }
.empty-state.small { padding: 20px; }
.empty-illustration { margin: 0 auto 16px; width: 100px; }
.empty-illustration svg { width: 100%; height: auto; }
.empty-text { font-size: 15px; font-weight: 600; color: var(--text-primary); margin-bottom: 4px; }
.empty-desc { font-size: 13px; color: var(--text-muted); }

/* 响应式 */
@media (max-width: 768px) {
  .stat-row { grid-template-columns: repeat(3, 1fr); gap: 8px; }
  .stat-card { padding: 12px; gap: 10px; }
  .stat-icon { width: 34px; height: 34px; }
  .stat-num { font-size: 18px; }
  .add-header { flex-direction: column; align-items: flex-start; gap: 10px; }
  .rules-grid { grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); }
  .rules-header { flex-direction: column; align-items: flex-start; gap: 10px; }
  .confirm-bar { flex-direction: column; gap: 10px; align-items: stretch; }
  .confirm-actions { justify-content: flex-end; }
  .region-grid { max-height: 300px; }
}
</style>
