<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
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

// ===================== Feature: 热门话题 rotating banner =====================
// A set of hot hashtags that rotate automatically every 3 seconds. The banner
// shows one hashtag at a time with a slide animation; tapping it searches.
const bannerHashtags = [
  { tag: '夏日清凉挑战', text: '🔥 #夏日清凉挑战 一起加入夏日清凉挑战吧！' },
  { tag: '热门舞蹈', text: '🔥 #热门舞蹈 跟着节奏一起摇摆起来～' },
  { tag: '美食探店', text: '🔥 #美食探店 发现身边隐藏的美味店铺' },
  { tag: '旅行日记', text: '🔥 #旅行日记 记录旅途中的美好瞬间' },
  { tag: '萌宠日常', text: '🔥 #萌宠日常 你家的毛孩子在干嘛呢？' },
  { tag: 'vlog记录', text: '🔥 #vlog记录 用镜头记录平凡而闪光的每一天' },
]
const bannerIndex = ref(0)
let bannerTimer = null
const currentBanner = computed(() => bannerHashtags[bannerIndex.value] || bannerHashtags[0])

function startBanner() {
  stopBanner()
  bannerTimer = setInterval(() => {
    bannerIndex.value = (bannerIndex.value + 1) % bannerHashtags.length
  }, 3000)
}
function stopBanner() {
  if (bannerTimer) { clearInterval(bannerTimer); bannerTimer = null }
}
function tapBanner() {
  searchHot(currentBanner.value.tag)
}
// Re-seed the banner from real hashtags once they load (falls back to defaults
// if the API returns nothing).
watch(tags, (data) => {
  if (data && data.length) {
    bannerHashtags.length = 0
    data.slice(0, 6).forEach((t) => bannerHashtags.push({ tag: t.name, text: `🔥 #${t.name} ${fmtCount(t.uses)}次使用` }))
    if (bannerIndex.value >= bannerHashtags.length) bannerIndex.value = 0
  }
}, { immediate: false })

onMounted(async () => {
  startBanner()
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

onUnmounted(stopBanner)

// ===================== Feature: 排行榜 quick links =====================
// Three ranking shortcuts that route to existing pages.
const rankingLinks = [
  { key: 'hot', label: '热搜榜', icon: '🔥', desc: '实时热搜', to: '/discover' },
  { key: 'music', label: '音乐榜', icon: '🎵', desc: '热门音乐', to: '/hot-music' },
  { key: 'fan', label: '贡献榜', icon: '🏅', desc: '粉丝贡献', to: '/fan-club' },
]
function goRank(link) {
  if (link.key === 'hot') {
    // No dedicated ranking page; jump to the inline hot-search list already
    // on this page by clearing any active filter and scrolling to it.
    keyword.value = ''
    videos.value = allVideos.value
    showToast('查看下方热搜榜')
    return
  }
  router.push(link.to)
}

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

// ===================== Feature: Discover daily pick (发现页每日精选) =====================
// A curated "💎 每日精选" section at the top of Discover. Three picks are chosen
// deterministically from the loaded feed based on a hash of today's date string,
// so every user sees the same 3 picks on a given day, and they change each day.
// Each pick shows the video's cover, an "编辑推荐" badge, and a reason picked
// deterministically from the reasons pool. Clicking a pick navigates to /feed.
const PICK_REASONS = ['今日热门', '优质内容', '创意满分', '治愈系', '正能量', '视觉震撼']

// dateHash turns today's date into a small non-negative integer hash. Using the
// YYYY-MM-DD string keeps it stable for the whole day and changes at midnight.
function dateHash(dateStr) {
  let h = 0
  for (let i = 0; i < dateStr.length; i++) {
    h = (h * 31 + dateStr.charCodeAt(i)) >>> 0
  }
  return h
}

function todayDateStr() {
  const d = new Date()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}

// dailyPicks derives the 3 deterministic picks from the loaded feed + today's
// hash. Returns [] until the feed has loaded so the section hides gracefully.
const dailyPicks = computed(() => {
  const pool = allVideos.value
  if (!pool || pool.length === 0) return []
  const h = dateHash(todayDateStr())
  const picks = []
  const usedIdx = new Set()
  let idx = h
  for (let n = 0; n < 3; n++) {
    // Step through the pool deterministically; wrap around.
    idx = (idx + n * 7 + 3) % pool.length
    let attempts = 0
    while (usedIdx.has(idx) && attempts < pool.length) {
      idx = (idx + 1) % pool.length
      attempts++
    }
    if (usedIdx.has(idx)) break
    usedIdx.add(idx)
    const reason = PICK_REASONS[(h + n * 5) % PICK_REASONS.length]
    picks.push({ video: pool[idx], reason })
  }
  return picks
})

</script>

<template>
  <div class="discover-page">
    <van-search v-model="keyword" placeholder="搜索视频、用户、话题" shape="round" show-action @search="doSearch" background="#161616">
      <template #action><span style="color: #fe2c55" @click="doSearch">搜索</span></template>
    </van-search>

    <!-- ===================== Feature: 每日精选 (daily pick) =====================
         A curated daily section: 3 picks chosen deterministically by date hash,
         each showing a cover image, an "编辑推荐" badge, and a pick reason.
         Clicking a pick navigates to /feed. Changes daily. -->
    <div v-if="dailyPicks.length" class="daily-picks">
      <div class="dp-head">
        <span class="dp-title">💎 每日精选</span>
        <span class="dp-sub">{{ todayDateStr() }} · 每日更新</span>
      </div>
      <div class="dp-row">
        <div
          v-for="(p, i) in dailyPicks"
          :key="i"
          class="dp-item"
          @click="router.push('/feed')"
        >
          <div class="dp-cover-wrap">
            <img class="dp-cover" :src="p.video.cover_url || 'https://via.placeholder.com/120x160'" />
            <span class="dp-badge">编辑推荐</span>
          </div>
          <div class="dp-reason">{{ p.reason }}</div>
          <div class="dp-name van-ellipsis">{{ p.video.title }}</div>
        </div>
      </div>
    </div>

    <!-- ===================== Feature: 热门话题 rotating banner ===================== -->
    <div class="rotating-banner" @click="tapBanner">
      <transition name="banner-slide" mode="out-in">
        <span :key="bannerIndex" class="banner-text">{{ currentBanner.text }}</span>
      </transition>
      <div class="banner-dots">
        <span
          v-for="(b, i) in bannerHashtags"
          :key="i"
          class="dot"
          :class="{ active: i === bannerIndex }"
        ></span>
      </div>
    </div>

    <!-- ===================== Feature: 排行榜 quick links ===================== -->
    <div class="rank-links">
      <div v-for="link in rankingLinks" :key="link.key" class="rank-link" @click="goRank(link)">
        <div class="rl-icon">{{ link.icon }}</div>
        <div class="rl-label">{{ link.label }}</div>
        <div class="rl-desc">{{ link.desc }}</div>
      </div>
    </div>

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

/* ===================== Feature: 每日精选 (daily pick) ===================== */
.daily-picks {
  margin: 8px 12px;
  background: linear-gradient(135deg, #1a0a14, #161616);
  border: 1px solid rgba(254, 44, 85, 0.25);
  border-radius: 14px;
  padding: 14px 12px 12px;
}
.dp-head { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 12px; }
.dp-title { color: #fff; font-size: 16px; font-weight: bold; }
.dp-sub { color: #888; font-size: 11px; }
.dp-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.dp-item { cursor: pointer; }
.dp-cover-wrap { position: relative; border-radius: 10px; overflow: hidden; }
.dp-cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; display: block; background: #222; }
.dp-badge {
  position: absolute;
  top: 6px;
  left: 6px;
  font-size: 9px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(90deg, #fe2c55, #ff6b9d);
  padding: 1px 6px;
  border-radius: 8px;
  white-space: nowrap;
}
.dp-reason {
  margin-top: 6px;
  color: #fe2c55;
  font-size: 12px;
  font-weight: 600;
  text-align: center;
}
.dp-name { color: rgba(255,255,255,0.85); font-size: 11px; margin-top: 2px; text-align: center; }

/* ===================== Feature: 热门话题 rotating banner ===================== */
.rotating-banner {
  position: relative; margin: 8px 12px; padding: 14px 16px; border-radius: 12px;
  background: linear-gradient(135deg, #fe2c55, #ff6b9d); color: #fff;
  cursor: pointer; overflow: hidden; min-height: 52px;
  display: flex; align-items: center; box-shadow: 0 4px 14px rgba(254,44,85,0.3);
}
.banner-text { font-size: 14px; font-weight: bold; line-height: 20px; flex: 1; }
.banner-dots { position: absolute; bottom: 6px; right: 14px; display: flex; gap: 4px; }
.dot { width: 5px; height: 5px; border-radius: 50%; background: rgba(255,255,255,0.45); transition: all 0.3s; }
.dot.active { background: #fff; width: 14px; border-radius: 3px; }
/* slide animation for the rotating banner text */
.banner-slide-enter-active, .banner-slide-leave-active { transition: all 0.5s ease; }
.banner-slide-enter-from { opacity: 0; transform: translateX(24px); }
.banner-slide-leave-to { opacity: 0; transform: translateX(-24px); }

/* ===================== Feature: 排行榜 quick links ===================== */
.rank-links {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px;
  margin: 0 12px 4px;
}
.rank-link {
  background: #161616; border-radius: 10px; padding: 12px 6px; text-align: center;
  cursor: pointer; transition: transform 0.15s, background 0.2s; border: 1px solid #1f1f1f;
}
.rank-link:active { transform: scale(0.96); }
.rank-link:hover { background: #1f1f1f; }
.rl-icon { font-size: 22px; line-height: 26px; }
.rl-label { color: #fff; font-size: 13px; font-weight: bold; margin-top: 4px; }
.rl-desc { color: #888; font-size: 10px; margin-top: 2px; }
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
