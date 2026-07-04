<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getLiveList, getLiveCities, getLiveSchedules } from '../api'

const router = useRouter()
const rooms = ref([])
const cities = ref([])
const schedules = ref([])
const loading = ref(true)
const activeTab = ref('live')

onMounted(async () => {
  try {
    rooms.value = await getLiveList()
    getLiveCities().then((d) => { cities.value = d || [] }).catch(() => {})
    getLiveSchedules().then((d) => { schedules.value = d || [] }).catch(() => {})
  }
  catch (e) { showToast('加载失败') }
  finally { loading.value = false }
})

function fmtTime(t) {
  if (!t) return ''
  return String(t).slice(5, 16).replace('T', ' ')
}

function fmt(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
function rankColor(i) {
  return ['gold', 'silver', 'bronze'][i] || 'normal'
}
</script>

<template>
  <div class="live-page">
    <van-nav-bar title="抖音直播" />
    <!-- Tab switcher: 直播中 / 预告 -->
    <div class="tab-bar">
      <span :class="{ active: activeTab === 'live' }" @click="activeTab = 'live'">直播中</span>
      <span :class="{ active: activeTab === 'schedule' }" @click="activeTab = 'schedule'">预告</span>
    </div>
    <!-- City channel selector -->
    <div v-if="cities.length && activeTab === 'live'" class="city-bar">
      <span class="city-label">📍 城市</span>
      <span class="city-chip" @click="router.push('/live')">全部</span>
      <span v-for="c in cities" :key="c.city" class="city-chip" @click="router.push('/city/' + c.city)">{{ c.city }} <small>{{ c.count }}</small></span>
    </div>
    <!-- Leaderboard banner -->
    <div class="lb-banner">🏆 直播人气榜</div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!rooms.length" description="暂无直播" />
    <div v-else class="live-grid">
      <div v-for="(r, i) in rooms" :key="r.id" class="live-card" @click="router.push('/live/' + r.id)">
        <div class="cover-wrap">
          <img class="cover" :src="r.cover_url" />
          <div class="rank-badge" :class="rankColor(i)" v-if="i < 10">No.{{ i + 1 }}</div>
          <div class="live-badge"><span class="dot"></span> 直播中</div>
          <div class="viewers">{{ fmt(r.viewers) }}人在看</div>
        </div>
        <div class="lc-info">
          <div class="lc-title van-ellipsis">{{ r.title }}</div>
          <div class="lc-host">
            <img class="lc-avatar" :src="r.host_avatar" />
            <span class="lc-name">{{ r.host_name }}</span>
            <span v-if="r.city" class="lc-city">{{ r.city }}</span>
          </div>
        </div>
      </div>
    </div>
    <!-- Schedule list (直播预告) -->
    <div v-if="activeTab === 'schedule'">
      <van-empty v-if="!schedules.length" description="暂无直播预告" />
      <div v-for="s in schedules" :key="s.id" class="schedule-card">
        <img class="sc-cover" :src="s.cover_url" />
        <div class="sc-info">
          <div class="sc-title van-multi-ellipsis--l2">{{ s.title }}</div>
          <div class="sc-host">
            <img class="sc-avatar" :src="s.host_avatar" />
            <span>{{ s.host_name }}</span>
          </div>
          <div class="sc-time">🕐 {{ fmtTime(s.scheduled_time) }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.live-page { height: 100vh; overflow-y: auto; background: #000; }
.tab-bar { display: flex; justify-content: center; gap: 24px; padding: 12px; background: #161616; }
.tab-bar span { color: #888; font-size: 16px; padding: 4px 16px; }
.tab-bar span.active { color: #fff; font-weight: bold; border-bottom: 2px solid #fe2c55; }
.schedule-card { display: flex; gap: 12px; padding: 12px; background: #161616; border-bottom: 1px solid #1a1a1a; }
.sc-cover { width: 80px; height: 100px; border-radius: 8px; object-fit: cover; flex-shrink: 0; }
.sc-info { flex: 1; }
.sc-title { color: #fff; font-size: 14px; line-height: 20px; }
.sc-host { display: flex; align-items: center; gap: 6px; margin-top: 8px; }
.sc-avatar { width: 24px; height: 24px; border-radius: 50%; }
.sc-host span { color: #999; font-size: 12px; }
.sc-time { color: #fe2c55; font-size: 12px; margin-top: 8px; }
.loading { text-align: center; padding: 60px; }
.city-bar { display: flex; align-items: center; gap: 8px; padding: 10px 12px; overflow-x: auto; scrollbar-width: none; }
.city-bar::-webkit-scrollbar { display: none; }
.city-label { color: #999; font-size: 12px; white-space: nowrap; }
.city-chip { background: #222; color: #fff; font-size: 12px; padding: 4px 12px; border-radius: 14px; white-space: nowrap; }
.city-chip small { color: #fe2c55; }
.lb-banner { background: linear-gradient(90deg, #fe2c55, #ff9800); color: #fff; text-align: center; padding: 8px; font-size: 14px; font-weight: bold; }
.live-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.live-card { background: #111; border-radius: 8px; overflow: hidden; }
.cover-wrap { position: relative; }
.cover { width: 100%; aspect-ratio: 3/4; object-fit: cover; }
.rank-badge { position: absolute; top: 8px; right: 8px; font-size: 10px; padding: 2px 8px; border-radius: 10px; font-weight: bold; }
.rank-badge.gold { background: linear-gradient(90deg, #ffd700, #ffaa00); color: #333; }
.rank-badge.silver { background: linear-gradient(90deg, #c0c0c0, #999); color: #333; }
.rank-badge.bronze { background: linear-gradient(90deg, #cd7f32, #b87333); color: #fff; }
.rank-badge.normal { background: rgba(0,0,0,0.6); color: #fff; }
.live-badge { position: absolute; top: 8px; left: 8px; background: #fe2c55; color: #fff; font-size: 10px; padding: 2px 8px; border-radius: 10px; display: flex; align-items: center; gap: 3px; }
.dot { width: 6px; height: 6px; background: #fff; border-radius: 50%; animation: pulse 1s infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
.viewers { position: absolute; bottom: 8px; right: 8px; background: rgba(0,0,0,0.5); color: #fff; font-size: 10px; padding: 2px 6px; border-radius: 4px; }
.lc-info { padding: 6px 8px; }
.lc-title { color: #fff; font-size: 13px; font-weight: bold; }
.lc-host { display: flex; align-items: center; gap: 4px; margin-top: 4px; }
.lc-avatar { width: 18px; height: 18px; border-radius: 50%; }
.lc-name { color: #999; font-size: 11px; }
.lc-city { color: #25f4ee; font-size: 10px; margin-left: auto; }
</style>
