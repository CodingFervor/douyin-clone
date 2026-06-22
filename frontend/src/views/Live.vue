<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getLiveList } from '../api'

const router = useRouter()
const rooms = ref([])
const loading = ref(true)

onMounted(async () => {
  try { rooms.value = await getLiveList() }
  catch (e) { showToast('加载失败') }
  finally { loading.value = false }
})

function fmt(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
</script>

<template>
  <div class="live-page">
    <van-nav-bar title="抖音直播" />
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!rooms.length" description="暂无直播" />
    <div v-else class="live-grid">
      <div v-for="r in rooms" :key="r.id" class="live-card" @click="router.push('/live/' + r.id)">
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
.live-page { height: 100vh; overflow-y: auto; background: #000; }
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
