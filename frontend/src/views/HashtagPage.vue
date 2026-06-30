<script setup>
import { ref, onMounted } from 'vue'
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
</script>

<template>
  <div class="tag-page">
    <van-nav-bar :title="'#' + tag" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="tag-banner">
      <div class="tb-hash">#{{ tag }}</div>
      <div class="tb-count">{{ videos.length }} 个作品</div>
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
.loading { text-align: center; padding: 60px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.grid-item { background: #111; border-radius: 6px; overflow: hidden; }
.cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.grid-title { color: #fff; font-size: 12px; line-height: 16px; padding: 4px 6px; height: 32px; }
.grid-meta { color: #999; font-size: 11px; padding: 0 6px 6px; }
</style>
