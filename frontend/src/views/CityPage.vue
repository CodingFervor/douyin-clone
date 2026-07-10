<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getLiveByCity } from '../api'

const route = useRoute()
const router = useRouter()
const city = ref(decodeURIComponent(route.params.city))
const rooms = ref([])
const loading = ref(true)

onMounted(async () => {
  try { rooms.value = await getLiveByCity(city.value) }
  catch (e) { showToast('加载失败') }
  finally { loading.value = false }
})

function fmt(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}

// Deterministic 32-bit hash (FNV-1a)
function hashStr(str) {
  let h = 2166136261 >>> 0
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i)
    h = Math.imul(h, 16777619) >>> 0
  }
  return h >>> 0
}

const WEATHERS = [
  { icon: '☀️', key: 'sun', label: '晴', tint: 'rgba(255,196,0,0.14), rgba(255,140,40,0.10)' },
  { icon: '☁️', key: 'cloud', label: '多云', tint: 'rgba(148,163,184,0.16), rgba(100,116,139,0.10)' },
  { icon: '🌧️', key: 'rain', label: '雨', tint: 'rgba(56,189,248,0.16), rgba(37,99,235,0.10)' },
  { icon: '❄️', key: 'snow', label: '雪', tint: 'rgba(186,230,253,0.18), rgba(125,211,252,0.10)' },
]

const weather = computed(() => {
  const h = hashStr(city.value)
  const temp = 15 + (h % 21) // 15..35
  const w = WEATHERS[h % WEATHERS.length]
  return { ...w, temp }
})

const weatherLabel = computed(() => weather.value.label)

const headerStyle = computed(() => ({
  backgroundImage: `linear-gradient(180deg, ${weather.value.tint})`,
}))
</script>

<template>
  <div class="city-page">
    <van-nav-bar :title="city + '直播'" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="city-header" :style="headerStyle">
      <div class="weather-badge">
        <span class="weather-icon">{{ weather.icon }}</span>
        <span class="weather-text">{{ city }} · {{ weatherLabel }} {{ weather.temp }}°</span>
      </div>
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!rooms.length" :description="city + '暂无直播'" />
    <div v-else class="live-grid">
      <div v-for="(r, i) in rooms" :key="r.id" class="live-card" @click="router.push('/live/' + r.id)">
        <div class="cover-wrap">
          <img class="cover" :src="r.cover_url" />
          <div class="live-badge"><span class="dot"></span> 直播中</div>
          <div class="viewers">{{ fmt(r.viewers) }}人在看</div>
        </div>
        <div class="lc-info">
          <div class="lc-title van-ellipsis">{{ r.title }}</div>
          <div class="lc-host">
            <img class="lc-avatar" :src="r.host_avatar" />
            <span class="lc-name">{{ r.host_name }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.city-page { height: 100vh; overflow-y: auto; background: #000; }
.city-header { padding: 14px 16px; border-bottom: 1px solid #1a1a1a; }
.weather-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 18px;
  background: rgba(255,255,255,0.08);
  backdrop-filter: blur(6px);
  color: #fff;
  font-size: 13px;
}
.weather-icon { font-size: 18px; }
.weather-text { font-weight: 500; letter-spacing: 0.3px; }
.loading { text-align: center; padding: 60px; }
.live-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.live-card { background: #111; border-radius: 8px; overflow: hidden; }
.cover-wrap { position: relative; }
.cover { width: 100%; aspect-ratio: 3/4; object-fit: cover; }
.live-badge { position: absolute; top: 8px; left: 8px; background: #fe2c55; color: #fff; font-size: 10px; padding: 2px 8px; border-radius: 10px; display: flex; align-items: center; gap: 3px; }
.dot { width: 6px; height: 6px; background: #fff; border-radius: 50%; animation: pulse 1s infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
.viewers { position: absolute; bottom: 8px; right: 8px; background: rgba(0,0,0,0.5); color: #fff; font-size: 10px; padding: 2px 6px; border-radius: 4px; }
.lc-info { padding: 6px 8px; }
.lc-title { color: #fff; font-size: 13px; font-weight: bold; }
.lc-host { display: flex; align-items: center; gap: 4px; margin-top: 4px; }
.lc-avatar { width: 18px; height: 18px; border-radius: 50%; }
.lc-name { color: #999; font-size: 11px; }
</style>
