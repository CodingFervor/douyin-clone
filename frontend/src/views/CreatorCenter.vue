<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile, getCreatorStats, getHotHashtags, getHotMusic, getLiveList } from '../api'

const router = useRouter()
const user = ref(null)
const stats = ref(null)
const tags = ref([])
const music = ref([])
const liveList = ref([])
const loading = ref(true)

// fmtCount shortens large numbers to a "w" form, matching other views.
function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n || 0)
}

onMounted(async () => {
  // Aggregate creator data drives the data cards; the other endpoints are
  // inspiration sources and are loaded best-effort so a failure of one does
  // not blank the whole page.
  try {
    const [profile, s] = await Promise.all([
      getProfile().catch(() => null),
      getCreatorStats().catch(() => null),
    ])
    user.value = profile
    stats.value = s
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
  getHotHashtags().then((data) => { tags.value = data || [] }).catch(() => {})
  getHotMusic().then((data) => { music.value = (data || []).slice(0, 3) }).catch(() => {})
  // Live list powers the "直播预告" inspiration area; fall back silently.
  getLiveList().then((data) => { liveList.value = (data || []).slice(0, 3) }).catch(() => {})
})
</script>

<template>
  <div class="cc-page">
    <van-nav-bar title="创作者中心" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <template v-else>
      <!-- Header: title + user avatar -->
      <div class="cc-header">
        <div class="cc-title">🎨 创作者中心</div>
        <div v-if="user" class="cc-user">
          <img class="cc-avatar" :src="user.avatar || 'https://via.placeholder.com/56'" />
          <div class="cc-user-info">
            <div class="cc-nick van-ellipsis">{{ user.nickname || user.username }}</div>
            <div class="cc-sub">欢迎回来，开始创作吧</div>
          </div>
        </div>
        <div v-else class="cc-user">
          <div class="cc-avatar-ph" @click="router.push('/login')"><van-icon name="user-o" size="28" /></div>
          <div class="cc-user-info">
            <div class="cc-nick" @click="router.push('/login')">点击登录</div>
          </div>
        </div>
      </div>

      <!-- Data cards row (作品数 / 总播放 / 总点赞 / 总评论) -->
      <div class="cc-section">
        <div class="cc-sec-head">📊 我的数据</div>
        <div class="cc-cards">
          <div class="cc-card">
            <div class="cc-card-value">{{ stats ? stats.video_count : 0 }}</div>
            <div class="cc-card-label">作品数</div>
          </div>
          <div class="cc-card">
            <div class="cc-card-value">{{ fmtCount(stats && stats.total_plays) }}</div>
            <div class="cc-card-label">总播放</div>
          </div>
          <div class="cc-card">
            <div class="cc-card-value">{{ fmtCount(stats && stats.total_likes) }}</div>
            <div class="cc-card-label">总点赞</div>
          </div>
          <div class="cc-card">
            <div class="cc-card-value">{{ fmtCount(stats && stats.total_comments) }}</div>
            <div class="cc-card-label">总评论</div>
          </div>
        </div>
      </div>

      <!-- 创作灵感: trending hashtags as clickable chips → /upload?tags= -->
      <div class="cc-section">
        <div class="cc-sec-head">✨ 创作灵感</div>
        <div class="cc-chips">
          <span
            v-for="t in tags.slice(0, 12)"
            :key="t.id"
            class="cc-chip"
            @click="router.push('/upload?tags=' + encodeURIComponent(t.name))"
          >
            #{{ t.name }} <small v-if="t.uses">{{ fmtCount(t.uses) }}</small>
          </span>
          <span v-if="!tags.length" class="cc-empty-inline">暂无热门话题</span>
        </div>
      </div>

      <!-- 热门音乐推荐: top 3 music with 使用 buttons → /upload?music= -->
      <div class="cc-section">
        <div class="cc-sec-head">🎵 热门音乐推荐</div>
        <div v-if="music.length" class="cc-music">
          <div v-for="(m, i) in music" :key="i" class="cc-music-item">
            <span class="cc-rank" :class="{ top: i < 3 }">{{ i + 1 }}</span>
            <div class="cc-music-info">
              <div class="cc-music-name van-ellipsis">🎵 {{ m.music }}</div>
              <div class="cc-music-uses">{{ fmtCount(m.uses) }}个作品使用</div>
            </div>
            <van-button size="mini" round color="#fe2c55" @click="router.push('/upload?music=' + encodeURIComponent(m.music))">使用</van-button>
          </div>
        </div>
        <div v-else class="cc-empty-inline">暂无音乐推荐</div>
      </div>

      <!-- 直播预告 inspiration: upcoming/suggested live rooms -->
      <div class="cc-section">
        <div class="cc-sec-head">🔴 直播预告</div>
        <div v-if="liveList.length" class="cc-live">
          <div v-for="l in liveList" :key="l.id" class="cc-live-item" @click="router.push('/live/' + l.id)">
            <img class="cc-live-cover" :src="l.cover_url || l.avatar || 'https://via.placeholder.com/64'" />
            <div class="cc-live-info">
              <div class="cc-live-title van-ellipsis">{{ l.title || (l.nickname + '的直播间') }}</div>
              <div class="cc-live-host">{{ l.nickname }}</div>
            </div>
          </div>
        </div>
        <div v-else class="cc-empty-inline">暂无直播预告，去开启一场直播吧</div>
      </div>

      <!-- Quick action buttons -->
      <div class="cc-actions">
        <van-button block round color="#fe2c55" @click="router.push('/upload')"><van-icon name="plus" /> 发布视频</van-button>
        <van-button block round plain color="#fe2c55" @click="router.push('/creator-stats')">数据中心</van-button>
        <van-button block round plain color="#25f4ee" @click="router.push('/live')">直播预告</van-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.cc-page { height: 100vh; overflow-y: auto; background: #000; padding-bottom: 24px; }
.loading { text-align: center; padding: 60px; }

/* Header */
.cc-header { background: linear-gradient(135deg, #fe2c55, #ff7a9a); padding: 20px 16px; }
.cc-title { color: #fff; font-size: 20px; font-weight: bold; }
.cc-user { display: flex; align-items: center; gap: 12px; margin-top: 14px; }
.cc-avatar { width: 56px; height: 56px; border-radius: 50%; border: 2px solid rgba(255,255,255,0.7); }
.cc-avatar-ph { width: 56px; height: 56px; border-radius: 50%; background: rgba(255,255,255,0.25); display: flex; align-items: center; justify-content: center; color: #fff; }
.cc-user-info { flex: 1; min-width: 0; }
.cc-nick { color: #fff; font-size: 16px; font-weight: bold; }
.cc-sub { color: rgba(255,255,255,0.85); font-size: 12px; margin-top: 2px; }

/* Sections */
.cc-section { background: #161616; margin: 10px 12px; border-radius: 12px; padding: 14px; }
.cc-sec-head { color: #fff; font-size: 15px; font-weight: bold; margin-bottom: 12px; }

/* Data cards */
.cc-cards { display: grid; grid-template-columns: 1fr 1fr 1fr 1fr; gap: 8px; }
.cc-card { background: #1f1f1f; border-radius: 10px; padding: 14px 6px; text-align: center; }
.cc-card-value { color: #fe2c55; font-size: 20px; font-weight: bold; }
.cc-card-label { color: #888; font-size: 11px; margin-top: 4px; }

/* Hashtag chips */
.cc-chips { display: flex; flex-wrap: wrap; gap: 8px; }
.cc-chip {
  padding: 6px 12px;
  background: #222;
  border: 1px solid #2a2a2a;
  border-radius: 16px;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}
.cc-chip:active { background: rgba(254,44,85,0.2); border-color: #fe2c55; }
.cc-chip small { color: #666; font-size: 10px; }

/* Music list */
.cc-music-item { display: flex; align-items: center; gap: 12px; padding: 8px 0; }
.cc-rank { width: 22px; text-align: center; color: #666; font-weight: bold; font-size: 16px; }
.cc-rank.top { color: #fe2c55; }
.cc-music-info { flex: 1; min-width: 0; }
.cc-music-name { color: #fff; font-size: 14px; }
.cc-music-uses { color: #888; font-size: 11px; margin-top: 2px; }

/* Live inspiration */
.cc-live-item { display: flex; align-items: center; gap: 10px; padding: 8px 0; }
.cc-live-cover { width: 56px; height: 56px; border-radius: 8px; object-fit: cover; }
.cc-live-info { flex: 1; min-width: 0; }
.cc-live-title { color: #fff; font-size: 14px; }
.cc-live-host { color: #888; font-size: 12px; margin-top: 2px; }

.cc-empty-inline { color: #666; font-size: 13px; }

/* Quick actions */
.cc-actions { display: flex; flex-direction: column; gap: 10px; margin: 16px 12px; }
</style>
