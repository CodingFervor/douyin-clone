<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getLiveList, getContributors } from '../api'

const router = useRouter()
const rooms = ref([])
const contributors = ref([])
const loading = ref(true)
const lastUpdated = ref('')
let timer = null

async function load() {
  try {
    const data = await getLiveList()
    rooms.value = data || []
    lastUpdated.value = new Date().toLocaleTimeString()
    // Best-effort: pull the contribution board for the hottest room as a sample.
    if (rooms.value.length) {
      const sampleId = rooms.value[0].id
      getContributors(sampleId).then((d) => { contributors.value = d || [] }).catch(() => {})
    }
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  // Auto-refresh the dashboard every 30 seconds.
  timer = setInterval(load, 30000)
})
onUnmounted(() => {
  if (timer) { clearInterval(timer); timer = null }
})

// ---- Aggregate stats (top row) ----
const totalRooms = computed(() => rooms.value.length)
const totalViewers = computed(() => rooms.value.reduce((s, r) => s + (r.viewers || 0), 0))
const totalLikes = computed(() => rooms.value.reduce((s, r) => s + (r.likes || 0), 0))
// Active hosts = distinct host_ids currently live.
const activeHosts = computed(() => new Set(rooms.value.map((r) => r.host_id)).size)

// ---- Top 5 rooms by viewer count (middle bar chart) ----
const topRooms = computed(() =>
  [...rooms.value]
    .sort((a, b) => (b.viewers || 0) - (a.viewers || 0))
    .slice(0, 5)
    .map((r) => ({ ...r, pct: 0 }))
)
const maxViewers = computed(() => Math.max(1, ...topRooms.value.map((r) => r.viewers || 0)))

// ---- Category breakdown (旅行/美食/舞蹈/萌宠) ----
const categories = ['旅行', '美食', '舞蹈', '萌宠']
const categoryStats = computed(() => {
  const counts = {}
  let total = 0
  categories.forEach((c) => (counts[c] = 0))
  rooms.value.forEach((r) => {
    const cat = (r.category || '').trim()
    if (categories.includes(cat)) { counts[cat]++; total++ }
  })
  const totalCat = Math.max(1, total)
  return categories.map((c) => ({ name: c, count: counts[c], pct: Math.round((counts[c] / totalCat) * 100) }))
})

function fmt(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
</script>

<template>
  <div class="dash-page">
    <van-nav-bar title="直播数据大屏" left-arrow @click-left="router.back()" fixed placeholder />

    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <template v-else>
      <!-- Top stat cards -->
      <div class="stat-row">
        <div class="stat-card">
          <div class="sc-value">{{ totalRooms }}</div>
          <div class="sc-label">在线直播 / 间</div>
        </div>
        <div class="stat-card">
          <div class="sc-value">{{ fmt(totalViewers) }}</div>
          <div class="sc-label">总观看 / 人</div>
        </div>
        <div class="stat-card">
          <div class="sc-value">{{ fmt(totalLikes) }}</div>
          <div class="sc-label">总点赞</div>
        </div>
        <div class="stat-card">
          <div class="sc-value">{{ activeHosts }}</div>
          <div class="sc-label">活跃主播</div>
        </div>
      </div>

      <!-- Top 5 rooms by viewer count -->
      <div class="panel">
        <div class="panel-head">📈 人气直播间 TOP 5</div>
        <div v-if="!topRooms.length" class="empty-tip">暂无直播数据</div>
        <div v-else class="bar-chart">
          <div v-for="r in topRooms" :key="r.id" class="bar-row" @click="router.push('/live/' + r.id)">
            <div class="bar-name van-ellipsis">{{ r.title }}</div>
            <div class="bar-track">
              <div class="bar-fill" :style="{ width: Math.max(6, Math.round(((r.viewers || 0) / maxViewers) * 100)) + '%' }"></div>
            </div>
            <div class="bar-value">{{ fmt(r.viewers || 0) }}</div>
          </div>
        </div>
      </div>

      <!-- Category breakdown -->
      <div class="panel">
        <div class="panel-head">🎯 直播分类占比</div>
        <div class="cat-list">
          <div v-for="c in categoryStats" :key="c.name" class="cat-row">
            <div class="cat-label">
              <span>{{ c.name }}</span>
              <small>{{ c.count }} 间</small>
            </div>
            <div class="cat-track">
              <div class="cat-fill" :style="{ width: c.pct + '%' }"></div>
            </div>
          </div>
        </div>
      </div>

      <div class="updated">数据每 30 秒自动刷新 · 最近更新 {{ lastUpdated || '--' }}</div>
    </template>
  </div>
</template>

<style scoped>
.dash-page { height: 100vh; overflow-y: auto; background: #000; }
.dash-page :deep(.van-nav-bar) { background: #000; }
.dash-page :deep(.van-nav-bar__title) { color: #fff; }
.dash-page :deep(.van-nav-bar .van-icon) { color: #fff; }
.loading { text-align: center; padding: 80px; }

/* Top stat row */
.stat-row { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; padding: 16px; }
.stat-card { background: #161616; border-radius: 12px; padding: 18px 12px; text-align: center; border: 1px solid #1f1f1f; }
.sc-value { color: #fe2c55; font-size: 28px; font-weight: bold; }
.sc-label { color: #888; font-size: 12px; margin-top: 6px; }

/* Panel wrapper */
.panel { background: #161616; border-radius: 12px; margin: 0 16px 16px; padding: 14px; border: 1px solid #1f1f1f; }
.panel-head { color: #fff; font-size: 15px; font-weight: bold; margin-bottom: 14px; }
.empty-tip { color: #666; font-size: 13px; text-align: center; padding: 20px 0; }

/* Horizontal bar chart */
.bar-chart { display: flex; flex-direction: column; gap: 12px; }
.bar-row { display: flex; align-items: center; gap: 10px; cursor: pointer; }
.bar-name { width: 80px; color: #ccc; font-size: 12px; flex-shrink: 0; }
.bar-track { flex: 1; height: 14px; background: #2a2a2a; border-radius: 7px; overflow: hidden; }
.bar-fill { height: 100%; background: linear-gradient(90deg, #fe2c55, #ff6a88); border-radius: 7px; transition: width 0.5s ease; }
.bar-value { width: 44px; text-align: right; color: #fe2c55; font-size: 12px; font-weight: bold; flex-shrink: 0; }

/* Category breakdown */
.cat-list { display: flex; flex-direction: column; gap: 12px; }
.cat-row { display: flex; flex-direction: column; gap: 6px; }
.cat-label { display: flex; justify-content: space-between; color: #ccc; font-size: 13px; }
.cat-label small { color: #666; }
.cat-track { height: 10px; background: #2a2a2a; border-radius: 5px; overflow: hidden; }
.cat-fill { height: 100%; background: linear-gradient(90deg, #25f4ee, #fe2c55); border-radius: 5px; transition: width 0.5s ease; }

.updated { text-align: center; color: #555; font-size: 11px; padding: 4px 16px 30px; }
</style>
