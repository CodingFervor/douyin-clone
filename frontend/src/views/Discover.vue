<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getFeed, searchVideos } from '../api'

const router = useRouter()
const allVideos = ref([])
const videos = ref([])
const keyword = ref('')
const loading = ref(true)
const searching = ref(false)

onMounted(async () => {
  try {
    const data = await getFeed(30)
    allVideos.value = data
    videos.value = data
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})

async function doSearch() {
  const kw = keyword.value.trim()
  if (!kw) {
    // empty search restores the full list
    videos.value = allVideos.value
    return
  }
  searching.value = true
  try {
    videos.value = await searchVideos(kw)
  } catch (e) {
    // fall back to client-side filter if the search endpoint fails
    videos.value = allVideos.value.filter((v) =>
      v.title.includes(kw) || (v.tags || '').includes(kw) || (v.author_name || '').includes(kw)
    )
  } finally {
    searching.value = false
  }
}

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
</script>

<template>
  <div class="discover-page">
    <van-search v-model="keyword" placeholder="搜索视频、用户、话题" shape="round" show-action @search="doSearch" background="#161616">
      <template #action><span style="color: #fe2c55" @click="doSearch">搜索</span></template>
    </van-search>
    <div class="hot-tags">
      <van-tag v-for="t in ['旅行', '美食', '舞蹈', '萌宠', '摄影', 'vlog']" :key="t" round plain color="#fe2c55" size="medium" @click="keyword = t; doSearch()">#{{ t }}</van-tag>
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <div class="grid" v-else>
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
.discover-page { height: 100vh; overflow-y: auto; background: #000; }
.hot-tags { display: flex; flex-wrap: wrap; gap: 8px; padding: 8px 12px; }
.loading { text-align: center; padding: 60px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.grid-item { background: #111; border-radius: 6px; overflow: hidden; }
.cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.grid-title { color: #fff; font-size: 12px; line-height: 16px; padding: 4px 6px; height: 32px; }
.grid-meta { color: #999; font-size: 11px; padding: 0 6px 6px; }
</style>
