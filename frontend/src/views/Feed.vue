<script setup>
import { ref, onMounted, onActivated, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getFeed, getRecommendFeed, getFollowingFeed, recordPlay, toggleLike, toggleFavorite, toggleFollow, getComments, createComment, likeComment, reportVideo, dismissVideo } from '../api'

const router = useRouter()
const videos = ref([])
const index = ref(0)
const loading = ref(true)
const activeTab = ref('recommend')
const showComment = ref(false)
const commentList = ref([])
const commentText = ref('')
const currentVideoId = ref(null)
const dragging = ref(false)
const startY = ref(0)
// Video progress (时长进度条)
const progress = ref(0) // 0-100 for the current video
const currentTime = ref('00:00')
const duration = ref('00:00')

onMounted(() => loadFeed('recommend'))
onActivated(() => { if (!videos.value.length) loadFeed(activeTab.value) })

// Reload the feed when the user switches between 关注/推荐.
watch(activeTab, (tab) => loadFeed(tab))

async function loadFeed(tab) {
  loading.value = true
  try {
    // "follow" shows videos from the users the current user follows; "recommend"
    // uses the collaborative-filtering engine. If the following feed is empty,
    // fall back to the general feed so the tab is never blank.
    let data
    if (tab === 'follow') {
      try {
        data = await getFollowingFeed(20)
      } catch (e) {
        data = [] // not logged in or error
      }
      if (!data.length) data = await getFeed(20)
    } else {
      data = await getRecommendFeed(20)
      if (!data.length) data = await getFeed(20)
    }
    videos.value = data
    index.value = 0
    await nextTick()
    playCurrent()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}

function playCurrent() {
  // Pause all videos, play the current one; report completion for the previous.
  document.querySelectorAll('.feed-video').forEach((v, i) => {
    if (i === index.value) {
      v.play().catch(() => {})
      // Report a completion ratio for the now-playing video as implicit feedback.
      const vid = videos.value[i]
      if (vid) {
        // Report ~0.5 completion when it starts, and the 'ended' listener below
        // upgrades it to 1.0 on full completion.
        reportPlay(vid.id, 0.5)
        v.onended = () => reportPlay(vid.id, 1.0)
      }
      // Reset + wire the progress bar.
      progress.value = 0
      v.ontimeupdate = () => {
        if (v.duration > 0) {
          progress.value = Math.min(100, (v.currentTime / v.duration) * 100)
          currentTime.value = fmtTime(v.currentTime)
          duration.value = fmtTime(v.duration)
        }
      }
    } else {
      v.pause()
    }
  })
}

function fmtTime(s) {
  if (!s || isNaN(s)) return '00:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
}

function reportPlay(videoID, completion) {
  // Best-effort; ignore failures (user may not be logged in).
  recordPlay(videoID, completion).catch(() => {})
}

// togglePlay pauses/plays the tapped video (tap = pause toggle).
function togglePlay(i) {
  if (i !== index.value) return
  const vids = document.querySelectorAll('.feed-video')
  const v = vids[i]
  if (!v) return
  if (v.paused) v.play().catch(() => {})
  else v.pause()
}

// Touch-based swipe to switch videos.
function onTouchStart(e) {
  dragging.value = true
  startY.value = e.touches[0].clientY
}
function onTouchEnd(e) {
  if (!dragging.value) return
  dragging.value = false
  const dy = e.changedTouches[0].clientY - startY.value
  if (dy < -50 && index.value < videos.value.length - 1) {
    index.value++
    playCurrent()
  } else if (dy > 50 && index.value > 0) {
    index.value--
    playCurrent()
  }
}

async function doLike(v) {
  try {
    const res = await toggleLike(v.id)
    v.liked = res.liked
    v.likes = res.likes
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}
async function doFav(v) {
  try {
    const res = await toggleFavorite(v.id)
    v.favorited = res.favorited
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}
async function doFollow(v) {
  try {
    const res = await toggleFollow(v.author_id)
    v.followed = res.following
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}

async function openComments(v) {
  currentVideoId.value = v.id
  showComment.value = true
  try {
    commentList.value = await getComments(v.id)
  } catch (e) {
    commentList.value = []
  }
}
async function sendComment() {
  if (!commentText.value.trim()) return
  try {
    const cm = await createComment({ video_id: currentVideoId.value, content: commentText.value })
    commentList.value.unshift(cm)
    commentText.value = ''
    const v = videos.value[index.value]
    if (v) v.comments_count++
  } catch (e) {
    showToast('请先登录')
  }
}
async function doCommentLike(c) {
  try {
    const res = await likeComment(c.id)
    c.liked = res.liked
    c.likes += res.liked ? 1 : -1
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
// Video download/save (视频下载)
function openShareSheet(v) {
  showDialog({
    title: '更多操作',
    message: '保存视频 / 不感兴趣 / 举报',
    showCancelButton: true,
    confirmButtonText: '保存',
    cancelButtonText: '不感兴趣',
  }).then(() => {
    saveVideo(v)
  }).catch(() => {
    doDismiss(v)
  })
}
async function doDismiss(v) {
  try {
    await dismissVideo(v.id)
    // Remove from the feed + advance to next.
    videos.value = videos.value.filter((vid) => vid.id !== v.id)
    if (index.value >= videos.value.length) index.value = Math.max(0, videos.value.length - 1)
    nextTick(() => playCurrent())
    showSuccessToast('已减少推荐')
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}
async function doReport(v) {
  try {
    await reportVideo(v.id, 'user_report')
    showSuccessToast('举报已提交')
  } catch (e) {
    showToast('举报失败')
  }
}
async function saveVideo(v) {
  try {
    const res = await fetch(v.video_url)
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = (v.title || 'douyin') + '.mp4'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    showSuccessToast('视频已保存')
  } catch (e) {
    showToast('保存失败，已复制链接')
    copyLink(v)
  }
}
async function copyLink(v) {
  try {
    await navigator.clipboard.writeText(v.video_url)
    showSuccessToast('链接已复制')
  } catch (e) {
    showToast('复制失败')
  }
}
</script>

<template>
  <div class="feed-page" @touchstart="onTouchStart" @touchend="onTouchEnd">
    <!-- Top tabs -->
    <div class="top-tabs">
      <span :class="{ active: activeTab === 'follow' }" @click="activeTab = 'follow'">关注</span>
      <span class="sep">|</span>
      <span :class="{ active: activeTab === 'recommend' }" @click="activeTab = 'recommend'">推荐</span>
      <van-icon name="search" class="search-btn" size="22" @click="router.push('/discover')" />
    </div>

    <!-- Video stack -->
    <div class="video-stack" v-if="videos.length">
      <div
        v-for="(v, i) in videos"
        :key="v.id"
        class="video-slide"
        :class="{ active: i === index }"
        :style="{ transform: `translateY(${(i - index) * 100}%)` }"
      >
        <video
          class="feed-video"
          :class="'filter-' + (v.filter || 'none')"
          :src="v.video_url"
          :poster="v.cover_url"
          loop
          playsinline
          webkit-playsinline
          @click="togglePlay(i)"
        ></video>
        <!-- Right action rail -->
        <div class="action-rail">
          <div class="avatar-wrap" @click="router.push('/user/' + v.author_id)">
            <img class="avatar" :src="v.author_avatar || 'https://via.placeholder.com/48'" />
            <div v-if="!v.followed" class="follow-plus"><van-icon name="plus" color="#fff" size="12" /></div>
          </div>
          <div class="action-item" @click="doLike(v)">
            <van-icon :name="v.liked ? 'like' : 'like-o'" :color="v.liked ? '#fe2c55' : '#fff'" size="32" />
            <span>{{ fmtCount(v.likes) }}</span>
          </div>
          <div class="action-item" @click="openComments(v)">
            <van-icon name="chat-o" color="#fff" size="32" />
            <span>{{ fmtCount(v.comments_count) }}</span>
          </div>
          <div class="action-item" @click="doFav(v)">
            <van-icon :name="v.favorited ? 'star' : 'star-o'" :color="v.favorited ? '#ffc107' : '#fff'" size="32" />
            <span>{{ fmtCount(v.shares) }}</span>
          </div>
          <div class="action-item" @click="router.push('/duet/' + v.id)">
            <van-icon name="exchange" color="#25f4ee" size="32" />
            <span>合拍</span>
          </div>
          <div class="action-item" @click="openShareSheet(v)">
            <van-icon name="share-o" color="#fff" size="32" />
            <span>分享</span>
          </div>
          <div class="disc"><van-icon name="music-o" /></div>
        </div>
        <!-- Bottom info -->
        <!-- Video progress bar (only for the active slide) -->
        <div v-if="i === index" class="progress-bar">
          <div class="pb-track"><div class="pb-fill" :style="{ width: progress + '%' }"></div></div>
          <span class="pb-time">{{ currentTime }} / {{ duration }}</span>
        </div>
        <div class="bottom-info">
          <div class="author">@{{ v.author_name }}</div>
          <div class="title">{{ v.title }}</div>
          <div v-if="v.description" class="desc">{{ v.description }}</div>
          <div class="tags">
            <span v-for="t in (v.tags || '').split(',')" :key="t" class="tag">#{{ t }}</span>
          </div>
          <div class="music-row" @click.stop="router.push('/music/' + v.id)"><van-icon name="music-o" size="14" /><span class="music-name">{{ v.music }}</span></div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>

    <!-- Comment popup -->
    <van-popup v-model:show="showComment" position="bottom" round :style="{ height: '50%' }">
      <div class="comment-panel">
        <div class="cp-head">{{ commentList.length }} 条评论</div>
        <div class="cp-list">
          <div v-for="c in commentList" :key="c.id" class="cp-item">
            <img class="cp-avatar" :src="c.avatar || 'https://via.placeholder.com/36'" />
            <div class="cp-body">
              <div class="cp-user">{{ c.username }}</div>
              <div class="cp-content">{{ c.content }}</div>
            </div>
            <div class="cp-like" :class="{ active: c.liked }" @click="doCommentLike(c)">
              <van-icon :name="c.liked ? 'like' : 'like-o'" size="16" :color="c.liked ? '#fe2c55' : '#999'" /><span>{{ c.likes }}</span>
            </div>
          </div>
          <div v-if="!commentList.length" class="cp-empty">暂无评论，来说点什么吧</div>
        </div>
        <div class="cp-input">
          <van-field v-model="commentText" placeholder="说点什么..." class="cp-field" @keyup.enter="sendComment" />
          <van-button size="small" type="primary" color="#fe2c55" @click="sendComment">发送</van-button>
        </div>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.feed-page { height: 100vh; background: #000; overflow: hidden; position: relative; }
.top-tabs { position: fixed; top: 0; left: 0; right: 0; z-index: 20; display: flex; align-items: center; justify-content: center; gap: 12px; padding: 12px 0; background: transparent; }
.top-tabs span { color: rgba(255,255,255,0.7); font-size: 17px; }
.top-tabs span.active { color: #fff; font-weight: bold; }
.top-tabs .sep { color: rgba(255,255,255,0.3); }
.search-btn { position: absolute; right: 16px; color: #fff; }
.video-stack { height: 100%; position: relative; }
.video-slide { position: absolute; top: 0; left: 0; width: 100%; height: 100%; transition: transform 0.35s ease; }
.feed-video { width: 100%; height: 100%; object-fit: cover; background: #000; }
/* CSS-based video filters (特效滤镜) — applied via v.filter class */
.feed-video.filter-none { filter: none; }
.feed-video.filter-vintage { filter: sepia(0.5) contrast(1.1) brightness(1.05); }
.feed-video.filter-warm { filter: saturate(1.3) hue-rotate(-10deg) brightness(1.05); }
.feed-video.filter-cool { filter: saturate(1.1) hue-rotate(15deg) brightness(0.98); }
.feed-video.filter-mono { filter: grayscale(1) contrast(1.1); }
.feed-video.filter-vivid { filter: saturate(1.8) contrast(1.15); }
.action-rail { position: absolute; right: 10px; bottom: 100px; display: flex; flex-direction: column; align-items: center; gap: 18px; z-index: 10; }
.avatar-wrap { position: relative; margin-bottom: 6px; }
.avatar { width: 48px; height: 48px; border-radius: 50%; border: 2px solid #fff; }
.follow-plus { position: absolute; bottom: -8px; left: 50%; transform: translateX(-50%); width: 20px; height: 20px; background: #fe2c55; border-radius: 50%; display: flex; align-items: center; justify-content: center; }
.action-item { display: flex; flex-direction: column; align-items: center; gap: 3px; }
.action-item span { color: #fff; font-size: 12px; }
.disc { width: 48px; height: 48px; border-radius: 50%; background: #222; display: flex; align-items: center; justify-content: center; animation: spin 4s linear infinite; }
.disc .van-icon { color: #25f4ee; font-size: 24px; }
@keyframes spin { from { transform: rotate(0); } to { transform: rotate(360deg); } }
.bottom-info { position: absolute; left: 12px; right: 76px; bottom: 20px; z-index: 10; }
.progress-bar { position: absolute; left: 12px; right: 12px; bottom: 96px; z-index: 11; display: flex; align-items: center; gap: 8px; }
.pb-track { flex: 1; height: 3px; background: rgba(255,255,255,0.3); border-radius: 2px; overflow: hidden; }
.pb-fill { height: 100%; background: #fe2c55; transition: width 0.2s linear; }
.pb-time { color: rgba(255,255,255,0.8); font-size: 11px; font-variant-numeric: tabular-nums; }
.author { color: #fff; font-size: 15px; font-weight: bold; margin-bottom: 6px; }
.title { color: #fff; font-size: 14px; line-height: 20px; margin-bottom: 6px; }
.desc { color: rgba(255,255,255,0.85); font-size: 13px; line-height: 18px; margin-bottom: 6px; }
.tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }
.tag { color: #fff; font-size: 13px; }
.music-row { display: flex; align-items: center; gap: 6px; color: #fff; font-size: 12px; }
.loading { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); z-index: 5; }
.comment-panel { display: flex; flex-direction: column; height: 100%; background: #161616; }
.cp-head { text-align: center; padding: 14px; color: #fff; font-size: 15px; border-bottom: 1px solid #222; }
.cp-list { flex: 1; overflow-y: auto; padding: 8px 12px; }
.cp-item { display: flex; gap: 10px; padding: 12px 0; }
.cp-avatar { width: 36px; height: 36px; border-radius: 50%; flex-shrink: 0; }
.cp-body { flex: 1; }
.cp-user { color: #888; font-size: 13px; }
.cp-content { color: #fff; font-size: 14px; margin-top: 3px; }
.cp-like { display: flex; flex-direction: column; align-items: center; color: #888; font-size: 11px; cursor: pointer; }
.cp-like.active { color: #fe2c55; }
.cp-empty { text-align: center; color: #666; padding: 40px; }
.cp-input { display: flex; gap: 8px; padding: 10px; border-top: 1px solid #222; }
.cp-field { background: #222; border-radius: 18px; }
</style>
