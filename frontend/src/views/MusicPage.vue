<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getVideosByMusic } from '../api'

const route = useRoute()
const router = useRouter()
const music = ref('')
const videos = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await getVideosByMusic(route.params.id)
    music.value = res.music || ''
    videos.value = res.data || []
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
</script>

<template>
  <div class="music-page">
    <van-nav-bar title="同款BGM" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="music-banner">
      <div class="mb-icon">🎵</div>
      <div class="mb-name">{{ music || '原声' }}</div>
      <div class="mb-count">{{ videos.length }} 个同款作品</div>
    </div>
    <div class="visualizer" aria-hidden="true">
      <span class="vbar vbar1"></span>
      <span class="vbar vbar2"></span>
      <span class="vbar vbar3"></span>
      <span class="vbar vbar4"></span>
      <span class="vbar vbar5"></span>
      <span class="vbar vbar6"></span>
      <span class="vbar vbar7"></span>
      <span class="vbar vbar8"></span>
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!videos.length" description="还没有同款作品" />
    <div v-else class="grid">
      <div v-for="v in videos" :key="v.id" class="grid-item" @click="router.push('/feed')">
        <img class="cover" :src="v.cover_url" />
        <div class="grid-title van-multi-ellipsis--l2">{{ v.title }}</div>
        <div class="grid-meta">
          <span><van-icon name="like-o" /> {{ fmtCount(v.likes) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.music-page { height: 100vh; overflow-y: auto; background: #000; }
.music-banner { background: linear-gradient(135deg, #1989fa, #5b8def); padding: 28px 20px; text-align: center; color: #fff; }
.mb-icon { font-size: 40px; }
.mb-name { font-size: 20px; font-weight: bold; margin-top: 8px; }
.mb-count { font-size: 13px; opacity: 0.9; margin-top: 4px; }
.visualizer {
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: 4px;
  height: 56px;
  padding: 8px 12px;
  background: #111;
}
.vbar {
  width: 6px;
  height: 16px;
  border-radius: 3px;
  background: linear-gradient(180deg, #25f4ee, #1989fa 55%, #fe2c55);
  animation: eq 1.1s ease-in-out infinite;
  transform-origin: bottom;
}
.vbar1 { animation-delay: 0s;    animation-duration: 1.0s; }
.vbar2 { animation-delay: 0.3s;  animation-duration: 0.9s; background: linear-gradient(180deg, #fe2c55, #ff5577 55%, #ff8a9d); }
.vbar3 { animation-delay: 0.5s;  animation-duration: 1.3s; }
.vbar4 { animation-delay: 0.15s; animation-duration: 0.8s; background: linear-gradient(180deg, #fe2c55, #ff5577 55%, #ff8a9d); }
.vbar5 { animation-delay: 0.4s;  animation-duration: 1.1s; }
.vbar6 { animation-delay: 0.65s; animation-duration: 0.7s; background: linear-gradient(180deg, #25f4ee, #5b8def 55%, #1989fa); }
.vbar7 { animation-delay: 0.25s; animation-duration: 1.2s; background: linear-gradient(180deg, #fe2c55, #ff5577 55%, #ff8a9d); }
.vbar8 { animation-delay: 0.55s; animation-duration: 0.95s; }
@keyframes eq {
  0%, 100% { height: 10px; }
  20%      { height: 44px; }
  40%      { height: 22px; }
  60%      { height: 50px; }
  80%      { height: 16px; }
}
.loading { text-align: center; padding: 60px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.grid-item { background: #111; border-radius: 6px; overflow: hidden; }
.cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.grid-title { color: #fff; font-size: 12px; line-height: 16px; padding: 4px 6px; height: 32px; }
.grid-meta { color: #999; font-size: 11px; padding: 0 6px 6px; }
</style>
