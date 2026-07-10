<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getVideosByTag } from '../api'

const route = useRoute()
const router = useRouter()
const tag = ref(route.params.tag || '')
const videos = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    videos.value = await getVideosByTag(tag.value)
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}

// Deterministic 32-bit hash from a string
function hashStr(str) {
  let h = 2166136261 >>> 0
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i)
    h = Math.imul(h, 16777619) >>> 0
  }
  return h >>> 0
}

const trend = computed(() => {
  const h = hashStr(tag.value)
  const bucket = h % 3 // 0=rising, 1=stable, 2=falling
  const pct = (h % 50) + 1 // 1..50
  const type = bucket === 0 ? 'rising' : bucket === 1 ? 'stable' : 'falling'
  return { type, pct }
})
</script>

<template>
  <div class="tag-page">
    <van-nav-bar :title="'#' + tag" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="tag-banner">
      <div class="tb-hash">#{{ tag }}</div>
      <div class="tb-count">{{ videos.length }} 个作品</div>
      <div class="trend-row">
        <span class="trend-pill" :class="trend.type">
          <span class="trend-arrow">{{ trend.type === 'rising' ? '📈' : trend.type === 'stable' ? '➡️' : '📉' }}</span>
          <span class="trend-label">{{ trend.type === 'rising' ? '上升' : trend.type === 'stable' ? '平稳' : '下降' }}</span>
          <span class="trend-pct">{{ trend.type === 'stable' ? '0.0' : (trend.pct % 10).toFixed(1) }}%</span>
        </span>
      </div>
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!videos.length" description="暂无相关视频" />
    <div v-else class="grid">
      <div v-for="v in videos" :key="v.id" class="grid-item" @click="router.push('/feed')">
        <img class="cover" :src="v.cover_url" />
        <div class="grid-title van-multi-ellipsis--l2">{{ v.title }}</div>
        <div class="grid-meta"><span><van-icon name="like-o" /> {{ fmtCount(v.likes) }}</span></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tag-page { height: 100vh; overflow-y: auto; background: #000; }
.tag-banner { background: linear-gradient(135deg, #fe2c55, #ff5577); padding: 24px 20px; text-align: center; color: #fff; }
.tb-hash { font-size: 26px; font-weight: bold; }
.tb-count { font-size: 13px; opacity: 0.9; margin-top: 4px; }
.trend-row { margin-top: 10px; display: flex; justify-content: center; }
.trend-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border-radius: 14px;
  font-size: 13px;
  font-weight: bold;
  background: rgba(255,255,255,0.15);
  backdrop-filter: blur(4px);
}
.trend-arrow { font-size: 14px; }
.trend-pill.rising { color: #4ade80; background: rgba(74,222,128,0.18); }
.trend-pill.stable { color: #d1d5db; background: rgba(209,213,219,0.18); }
.trend-pill.falling { color: #f87171; background: rgba(248,113,113,0.18); }
.loading { text-align: center; padding: 60px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.grid-item { background: #111; border-radius: 6px; overflow: hidden; }
.cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.grid-title { color: #fff; font-size: 12px; line-height: 16px; padding: 4px 6px; height: 32px; }
.grid-meta { color: #999; font-size: 11px; padding: 0 6px 6px; }
</style>
