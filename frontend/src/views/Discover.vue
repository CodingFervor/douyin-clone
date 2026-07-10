<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getFeed, searchVideos, getHotSearch, getHotHashtags, getSuggestFollows, toggleFollow } from '../api'

const router = useRouter()
const allVideos = ref([])
const videos = ref([])
const keyword = ref('')
const loading = ref(true)
const searching = ref(false)
const hotList = ref([])
const tags = ref([])
const suggestUsers = ref([])

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
  // Load the hot-search ranking + hot hashtags (best-effort).
  getHotSearch().then((data) => { hotList.value = data || [] }).catch(() => {})
  getHotHashtags().then((data) => { tags.value = data || [] }).catch(() => {})
  getSuggestFollows().then((data) => { suggestUsers.value = data || [] }).catch(() => {})
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
    // Each search feeds the hot-search ranking; refresh it.
    getHotSearch().then((data) => { hotList.value = data || [] }).catch(() => {})
  } catch (e) {
    // fall back to client-side filter if the search endpoint fails
    videos.value = allVideos.value.filter((v) =>
      v.title.includes(kw) || (v.tags || '').includes(kw) || (v.author_name || '').includes(kw)
    )
  } finally {
    searching.value = false
  }
}

function searchHot(kw) {
  keyword.value = kw
  doSearch()
}
async function doSuggestFollow(u) {
  try { await toggleFollow(u.id); u.followed = true }
  catch (e) { showToast('请先登录'); router.push('/login') }
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
    <!-- Hot search ranking -->
    <div v-if="hotList.length" class="hot-search">
      <div class="hs-head">🔥 抖音热搜榜</div>
      <div v-for="(h, i) in hotList.slice(0, 8)" :key="h.keyword" class="hs-item" @click="searchHot(h.keyword)">
        <span class="hs-rank" :class="{ top: i < 3 }">{{ i + 1 }}</span>
        <span class="hs-kw">{{ h.keyword }}</span>
        <span class="hs-count">{{ fmtCount(h.count) }}次</span>
      </div>
    </div>
    <!-- Hot hashtags (#话题) -->
    <div v-if="tags.length" class="hot-search">
      <div class="hs-head"># 热门话题</div>
      <div class="tag-chips">
        <van-tag v-for="t in tags.slice(0, 12)" :key="t.id" round plain color="#fe2c55" size="medium" @click="router.push('/tag/' + t.name)">
          #{{ t.name }} <small>{{ fmtCount(t.uses) }}</small>
        </van-tag>
      </div>
    </div>
    <!-- Hot music entry -->
    <div class="hot-search" @click="router.push('/hot-music')" style="cursor:pointer">
      <div class="hs-head">🎵 热门音乐榜 ›</div>
    </div>
    <!-- Music studio entry (音乐工作室) -->
    <div class="hot-search" @click="router.push('/music-studio')" style="cursor:pointer">
      <div class="hs-head">🎵 音乐工作室 ›</div>
    </div>
    <!-- Suggested follows (关注推荐) -->
    <div v-if="suggestUsers.length" class="hot-search">
      <div class="hs-head">👥 推荐关注</div>
      <div class="suggest-list">
        <div v-for="u in suggestUsers.slice(0, 4)" :key="u.id" class="su-item" @click="router.push('/user/' + u.id)">
          <img class="su-avatar" :src="u.avatar" />
          <div class="su-info">
            <div class="su-name van-ellipsis">{{ u.nickname }}</div>
            <div class="su-fans">{{ fmtCount(u.followers_count) }}粉丝</div>
          </div>
          <van-button size="mini" round color="#fe2c55" :disabled="u.followed" @click.stop="doSuggestFollow(u)">{{ u.followed ? '已关注' : '关注' }}</van-button>
        </div>
      </div>
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
.hot-search { background: #161616; margin: 8px 12px; border-radius: 10px; padding: 12px; }
.hs-head { color: #fff; font-size: 15px; font-weight: bold; margin-bottom: 10px; }
.hs-item { display: flex; align-items: center; gap: 12px; padding: 8px 0; }
.hs-rank { width: 22px; text-align: center; color: #999; font-size: 15px; font-weight: bold; font-style: italic; }
.hs-rank.top { color: #fe2c55; }
.hs-kw { flex: 1; color: #fff; font-size: 14px; }
.hs-count { color: #666; font-size: 11px; }
.tag-chips { display: flex; flex-wrap: wrap; gap: 8px; }
.tag-chips small { color: #666; font-size: 10px; }
.suggest-list { margin-top: 8px; }
.su-item { display: flex; align-items: center; gap: 10px; padding: 8px 0; }
.su-avatar { width: 40px; height: 40px; border-radius: 50%; }
.su-info { flex: 1; min-width: 0; }
.su-name { color: #fff; font-size: 14px; }
.su-fans { color: #666; font-size: 11px; margin-top: 2px; }
.loading { text-align: center; padding: 60px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.grid-item { background: #111; border-radius: 6px; overflow: hidden; }
.cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.grid-title { color: #fff; font-size: 12px; line-height: 16px; padding: 4px 6px; height: 32px; }
.grid-meta { color: #999; font-size: 11px; padding: 0 6px 6px; }
</style>
