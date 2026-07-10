<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile, getUserVideos } from '../api'

const router = useRouter()
const user = ref(null)
const videos = ref([])
const loading = ref(true)
// Index of the currently opened series card (null = card grid view)
const openSeries = ref(null)

// Every 5 videos form one series (Vol.N).  The playlist view shows
// all videos in that group with cover + title + plays.
const GROUP_SIZE = 5

async function load() {
  try {
    user.value = await getProfile()
    videos.value = await getUserVideos(user.value.id)
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// Auto-group videos into series of GROUP_SIZE consecutive clips.
const seriesList = computed(() => {
  const groups = []
  for (let i = 0; i < videos.value.length; i += GROUP_SIZE) {
    const slice = videos.value.slice(i, i + GROUP_SIZE)
    groups.push({
      index: groups.length,
      vol: groups.length + 1,
      videos: slice,
      total: slice.length,
      plays: slice.reduce((s, v) => s + (v.plays || 0), 0),
    })
  }
  return groups
})

// The series currently expanded into the playlist view.
const currentSeries = computed(() =>
  openSeries.value === null ? null : seriesList.value[openSeries.value] || null
)

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n || 0)
}
</script>

<template>
  <div class="series-page">
    <van-nav-bar title="我的合集" left-arrow @click-left="router.back()" fixed placeholder />

    <!-- Playlist view: a single series expanded -->
    <template v-if="currentSeries">
      <div class="pl-back">
        <van-button size="small" round color="#333" @click="openSeries = null"><van-icon name="arrow-left" /> 返回合集</van-button>
      </div>
      <div class="pl-header">
        <div class="pl-title">我的作品合集 Vol.{{ currentSeries.vol }}</div>
        <div class="pl-meta">共 {{ currentSeries.total }} 个作品 · {{ fmtCount(currentSeries.plays) }} 次播放</div>
      </div>
      <div class="pl-list">
        <div
          v-for="v in currentSeries.videos"
          :key="v.id"
          class="pl-item"
          @click="router.push('/feed')"
        >
          <img class="pl-cover" :src="v.cover_url" />
          <div class="pl-info">
            <div class="pl-name van-ellipsis">{{ v.title }}</div>
            <div class="pl-plays"><van-icon name="play-circle-o" /> {{ fmtCount(v.plays) }} 播放</div>
          </div>
          <van-icon name="play" color="#fe2c55" size="20" />
        </div>
        <van-empty v-if="!currentSeries.videos.length" description="该合集暂无作品" image="search" />
      </div>
    </template>

    <!-- Card grid view: all series as collage cards -->
    <template v-else>
      <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
      <van-empty v-else-if="!seriesList.length" description="还没有作品合集">
        <van-button type="primary" color="#fe2c55" round @click="router.push('/upload')">去发布作品</van-button>
      </van-empty>
      <div v-else class="series-list">
        <div
          v-for="s in seriesList"
          :key="s.index"
          class="series-card"
          @click="openSeries = s.index"
        >
          <!-- 3-video thumbnail collage (grid of 3 covers) -->
          <div class="collage">
            <template v-if="s.videos.length">
              <div class="collage-main">
                <img :src="s.videos[0].cover_url" />
              </div>
              <div class="collage-side">
                <img v-if="s.videos[1]" :src="s.videos[1].cover_url" />
                <div v-else class="ph"></div>
                <img v-if="s.videos[2]" :src="s.videos[2].cover_url" />
                <div v-else class="ph"></div>
              </div>
            </template>
            <div class="collage-count">共 {{ s.total }} 个作品</div>
          </div>
          <div class="series-info">
            <div class="series-title">我的作品合集 Vol.{{ s.vol }}</div>
            <div class="series-sub">
              <span><van-icon name="video-o" /> {{ s.total }} 个视频</span>
              <span><van-icon name="play-circle-o" /> {{ fmtCount(s.plays) }} 次播放</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.series-page { height: 100vh; overflow-y: auto; background: #000; }
.series-page :deep(.van-nav-bar) { background: #000; }
.series-page :deep(.van-nav-bar__title) { color: #fff; }
.series-page :deep(.van-nav-bar .van-icon) { color: #fff; }
.loading { text-align: center; padding: 80px; }

.series-list { padding: 12px; display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.series-card { background: #161616; border-radius: 12px; overflow: hidden; border: 1px solid #1f1f1f; }

/* Collage: left big cover + right two stacked covers */
.collage { position: relative; display: flex; height: 140px; }
.collage-main { flex: 1.4; overflow: hidden; }
.collage-main img { width: 100%; height: 100%; object-fit: cover; }
.collage-side { flex: 1; display: flex; flex-direction: column; }
.collage-side img { flex: 1; width: 100%; object-fit: cover; }
.collage-side .ph { flex: 1; background: #222; }
.collage-count {
  position: absolute; right: 6px; bottom: 6px;
  background: rgba(0,0,0,0.6); color: #fff; font-size: 10px;
  padding: 2px 6px; border-radius: 8px;
}

.series-info { padding: 10px; }
.series-title { color: #fff; font-size: 14px; font-weight: bold; }
.series-sub { display: flex; flex-direction: column; gap: 2px; color: #888; font-size: 11px; margin-top: 6px; }
.series-sub span { display: flex; align-items: center; gap: 4px; }

/* Playlist view */
.pl-back { padding: 10px 12px 0; }
.pl-header { padding: 8px 16px 14px; }
.pl-title { color: #fff; font-size: 18px; font-weight: bold; }
.pl-meta { color: #888; font-size: 12px; margin-top: 4px; }
.pl-list { padding: 0 12px; }
.pl-item {
  display: flex; align-items: center; gap: 12px;
  background: #161616; border-radius: 10px; padding: 10px;
  margin-bottom: 8px;
}
.pl-cover { width: 64px; height: 84px; border-radius: 6px; object-fit: cover; flex-shrink: 0; }
.pl-info { flex: 1; min-width: 0; }
.pl-name { color: #fff; font-size: 14px; }
.pl-plays { color: #888; font-size: 12px; margin-top: 6px; display: flex; align-items: center; gap: 4px; }
</style>
