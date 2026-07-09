<script setup>
import { ref, computed, onMounted, onActivated, onUnmounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getFeed, getRecommendFeed, getFollowingFeed, recordPlay, toggleLike, toggleFavorite, toggleFollow, getComments, createComment, likeComment, reportVideo, dismissVideo, getSuggestFollows } from '../api'

const router = useRouter()
const videos = ref([])
const index = ref(0)
const loading = ref(true)
const activeTab = ref('recommend')
const showComment = ref(false)
const commentList = ref([])
const commentText = ref('')
const replyText = ref('')
const replyTo = ref(null) // comment being replied to (null = top-level)
const currentVideoId = ref(null)
const dragging = ref(false)
const startY = ref(0)
// Video progress (时长进度条)
const progress = ref(0) // 0-100 for the current video
const currentTime = ref('00:00')
const duration = ref('00:00')

// ---- Feature 1: Comment @mention (评论区at提及) ----
// When the user types "@", show a popup of suggested users from
// getSuggestFollows. On select, "@nickname " is inserted at the cursor.
const mentionMode = ref(false) // true while the @ popup is open
const mentionList = ref([]) // suggested users to display
const mentionLoaded = ref(false) // true once we've fetched suggestions once

// ---- Feature 2: Video quality switch (清晰度切换) ----
// 'sd' = 标清 (standard), 'hd' = 高清 (high definition). The actual
// video_url is unchanged — this is a visual/demo control.
const quality = ref('sd')

// ---- Feature 4: Playback speed (视频慢放/倍速) ----
// Cycles through 0.5x / 1.0x / 1.5x / 2.0x. Applied to the active video
// element immediately on change and re-applied in playCurrent() so the
// rate persists across slides.
const SPEEDS = [0.5, 1.0, 1.5, 2.0]
const playbackRate = ref(1.0)
function cycleSpeed() {
  const cur = SPEEDS.indexOf(playbackRate.value)
  playbackRate.value = SPEEDS[(cur + 1) % SPEEDS.length]
  // Apply immediately to the currently-playing video element.
  const vids = document.querySelectorAll('.feed-video')
  const v = vids[index.value]
  if (v) v.playbackRate = playbackRate.value
}

// ---- Feature 3: Expandable caption (视频文案展示) ----
// Per-video expand state for long descriptions, keyed by video id. When a
// description is collapsed (-webkit-line-clamp: 2) we show an "展开" toggle;
// once expanded we show the full text with a "收起" toggle.
const expandedDesc = ref({})
// Toggle the expand/collapse state for a given video id.
function toggleDesc(id) {
  expandedDesc.value[id] = !expandedDesc.value[id]
}
// Whether a description is long enough to warrant a toggle (heuristic by
// length — two visual lines roughly correspond to ~40 chars / 80 for CJK).
function descIsLong(text) {
  if (!text) return false
  return text.length > 40
}

// ---- Feature 1: Double-tap Like animation (双击点赞动画) ----
// A double-tap anywhere on the video (two taps within 300ms) bursts a large
// heart in the center and auto-likes the video. A single tap still toggles
// play/pause; to avoid the heart firing on a single tap, the play toggle is
// deferred 300ms and cancelled if a second tap lands in time.
const heartBurst = ref([]) // active heart animations [{id, x, y}]
let lastTapTime = 0
let singleTapTimer = null
let heartSeq = 0

// onVideoTap distinguishes single vs. double tap using a 300ms window. On the
// second tap within the window it cancels the pending play-toggle and triggers
// the heart burst + like instead.
function onVideoTap(i, e) {
  // The first tap on the video dismisses the one-time guide overlay.
  if (showGuide.value) dismissGuide()
  const now = Date.now()
  if (now - lastTapTime < 300 && lastTapTime > 0) {
    // Double tap — cancel any pending single-tap play toggle.
    if (singleTapTimer) {
      clearTimeout(singleTapTimer)
      singleTapTimer = null
    }
    lastTapTime = 0
    triggerHeart(i, e)
    return
  }
  lastTapTime = now
  // Defer the play/pause toggle so a quick second tap can cancel it.
  if (singleTapTimer) clearTimeout(singleTapTimer)
  singleTapTimer = setTimeout(() => {
    singleTapTimer = null
    togglePlay(i)
  }, 300)
}

// triggerHeart spawns a heart at the tap position (or center) and auto-likes.
function triggerHeart(i, e) {
  const id = ++heartSeq
  const rect = e && e.target ? e.target.getBoundingClientRect() : null
  const x = e && typeof e.clientX === 'number' ? e.clientX : (rect ? rect.left + rect.width / 2 : window.innerWidth / 2)
  const y = e && typeof e.clientY === 'number' ? e.clientY : (rect ? rect.top + rect.height / 2 : window.innerHeight / 2)
  heartBurst.value.push({ id, x, y })
  // Remove once the ~800ms animation finishes.
  setTimeout(() => {
    heartBurst.value = heartBurst.value.filter((h) => h.id !== id)
  }, 800)
  // Auto-like the current video (only if not already liked).
  const v = videos.value[i]
  if (v && !v.liked) doLike(v)
}

// ---- Feature 2: Follow feed unread badge (关注Tab红点) ----
// While on the 推荐 tab, periodically poll the following feed in the
// background and light up a red dot on the 关注 tab when new videos appear.
// Switching to 关注 marks them seen and clears the dot.
const followHasNew = ref(false)
let followCheckTimer = null
let lastFollowVideoId = null // newest video id seen in the following feed

// checkFollowNew fetches the following feed (best-effort) and compares its
// newest video id against the last one we showed the user. On the very first
// check we just record the baseline so we don't badge on initial load.
async function checkFollowNew() {
  try {
    const data = await getFollowingFeed(1)
    const latestId = data && data.length ? data[0].id : null
    if (latestId === null) return
    if (lastFollowVideoId === null) {
      lastFollowVideoId = latestId
      return
    }
    if (latestId !== lastFollowVideoId) {
      lastFollowVideoId = latestId
      followHasNew.value = true
    }
  } catch (e) {
    // Not logged in or transient error — ignore.
  }
}

function startFollowCheck() {
  if (activeTab.value !== 'recommend') return
  // Initial baseline / check, then poll every 30s.
  checkFollowNew()
  stopFollowCheck()
  followCheckTimer = setInterval(checkFollowNew, 30000)
}

function stopFollowCheck() {
  if (followCheckTimer) {
    clearInterval(followCheckTimer)
    followCheckTimer = null
  }
}

// ===================== Feature: First-time swipe-up guide =====================
// A one-time overlay shown to new users explaining how to swipe to the next
// video and double-tap to like. It auto-dismisses after 4s, or on the first
// swipe/tap, and records 'dy_guide_shown' in localStorage so it never repeats.
const GUIDE_KEY = 'dy_guide_shown'
const showGuide = ref(false)
let guideTimer = null

// Returns true once per browser when the guide has not been shown yet.
function guideAlreadyShown() {
  try {
    return localStorage.getItem(GUIDE_KEY) === '1'
  } catch (e) {
    return false
  }
}

// showSwipeGuide displays the overlay and arms the 4s auto-dismiss timer.
function showSwipeGuide() {
  if (guideAlreadyShown()) return
  showGuide.value = true
  if (guideTimer) clearTimeout(guideTimer)
  guideTimer = setTimeout(dismissGuide, 4000)
}

// dismissGuide hides the overlay and persists the flag so it won't reappear.
function dismissGuide() {
  if (!showGuide.value) return
  showGuide.value = false
  if (guideTimer) {
    clearTimeout(guideTimer)
    guideTimer = null
  }
  try {
    localStorage.setItem(GUIDE_KEY, '1')
  } catch (e) {
    // localStorage may be unavailable (private mode) — ignore.
  }
}

// ===================== Feature: Pinned comment (评论置顶) =====================
// Frontend-only: pin the video author's comment (user_id === author_id), or
// failing that the most-liked top-level comment, to the top of the list with a
// 置顶 badge. It is removed from the regular list to avoid duplication.
// pinnedComment is the chosen comment (or null); regularComments is the
// remaining list (excluding the pinned one), order preserved.
const pinnedComment = ref(null)
const regularComments = ref([])

// recomputePinned splits commentList into a pinned entry + the rest. The author's
// own top-level comment wins; otherwise the top-level comment with the most
// likes is pinned. Child comments (parent_id != 0) are never pinned.
function recomputePinned() {
  const list = commentList.value || []
  const current = videos.value[index.value]
  const authorId = current ? current.author_id : null
  const topLevel = list.filter((c) => !c.parent_id || c.parent_id === 0)

  let pinned = null
  if (authorId != null) {
    pinned = topLevel.find((c) => c.user_id === authorId) || null
  }
  if (!pinned && topLevel.length) {
    // Most-liked top-level comment; ties broken by original order (stable find).
    let best = null
    for (const c of topLevel) {
      if (!best || (c.likes || 0) > (best.likes || 0)) best = c
    }
    pinned = best
  }
  pinnedComment.value = pinned
  regularComments.value = pinned
    ? list.filter((c) => c.id !== pinned.id)
    : list.slice()
}

onMounted(() => loadFeed('recommend'))
onActivated(() => { if (!videos.value.length) loadFeed(activeTab.value) })

// Reload the feed when the user switches between 关注/推荐. Also manage the
// follow-feed unread badge: start polling while on 推荐, clear the badge +
// update the baseline when the user views the 关注 tab.
watch(activeTab, (tab) => {
  loadFeed(tab)
  if (tab === 'follow') {
    // Viewing the following feed marks new videos as seen.
    followHasNew.value = false
    stopFollowCheck()
    // Refresh the baseline so we don't immediately re-badge the same videos.
    getFollowingFeed(1)
      .then((data) => {
        lastFollowVideoId = data && data.length ? data[0].id : lastFollowVideoId
      })
      .catch(() => {})
  } else {
    startFollowCheck()
  }
})

// Kick off the background check once the feed first loads on 推荐.
onMounted(() => startFollowCheck())
// Show the one-time swipe-up guide for new users once the feed mounts.
onMounted(() => showSwipeGuide())

onUnmounted(() => {
  stopFollowCheck()
  if (singleTapTimer) clearTimeout(singleTapTimer)
  if (guideTimer) clearTimeout(guideTimer)
})

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
      // Feature 4: re-apply the selected playback rate so it persists
      // across slides / after the element is re-rendered.
      v.playbackRate = playbackRate.value
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
  // The first swipe dismisses the one-time guide overlay.
  if (showGuide.value) dismissGuide()
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
  replyTo.value = null
  replyText.value = ''
  showComment.value = true
  try {
    commentList.value = await getComments(v.id)
  } catch (e) {
    commentList.value = []
  }
  // Recompute the pinned comment for the now-open video's comment list.
  recomputePinned()
}
// Start composing a reply to a specific comment (or cancel if already replying).
function startReply(c) {
  if (replyTo.value && replyTo.value.id === c.id) {
    replyTo.value = null
  } else {
    replyTo.value = c
    replyText.value = ''
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
    recomputePinned()
  } catch (e) {
    showToast('请先登录')
  }
}
// Submit a reply to the comment currently held in replyTo; passes parent_id so
// the backend stores it as a nested child comment.
async function sendReply() {
  if (!replyText.value.trim() || !replyTo.value) return
  try {
    const cm = await createComment({
      video_id: currentVideoId.value,
      content: replyText.value,
      parent_id: replyTo.value.id,
    })
    // Place the reply right after its parent in the flat list; the indented
    // styling (margin-left) reflects the parent_id link.
    const parentIdx = commentList.value.findIndex((c) => c.id === replyTo.value.id)
    if (parentIdx >= 0) {
      commentList.value.splice(parentIdx + 1, 0, cm)
    } else {
      commentList.value.push(cm)
    }
    replyText.value = ''
    replyTo.value = null
    const v = videos.value[index.value]
    if (v) v.comments_count++
    recomputePinned()
  } catch (e) {
    showToast('请先登录')
  }
}
async function doCommentLike(c) {
  try {
    const res = await likeComment(c.id)
    c.liked = res.liked
    c.likes += res.liked ? 1 : -1
    // A change in likes could change which comment is "most-liked", so refresh
    // the pinned selection (only re-pins when no author comment exists).
    if (!pinnedComment.value || pinnedComment.value.user_id !== videos.value[index.value]?.author_id) {
      recomputePinned()
    }
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}

// ===================== Feature 1: @mention (评论区at提及) =====================

// onCommentInput watches the comment field. When the last typed char is "@",
// open the suggestion popup and lazily load the suggested-follows list once.
function onCommentInput() {
  const text = commentText.value
  if (!text) {
    mentionMode.value = false
    return
  }
  const atIdx = text.lastIndexOf('@')
  if (atIdx >= 0) {
    // Only treat "@" as a mention trigger when it is at the start of the
    // input or follows whitespace — avoids matching emails ("a@b.com").
    const prev = atIdx > 0 ? text[atIdx - 1] : ' '
    const tail = text.slice(atIdx + 1)
    if (/\s/.test(prev) && !/\s/.test(tail)) {
      openMention()
      return
    }
  }
  mentionMode.value = false
}

// Load the suggestion list (cached after the first fetch) and open the popup.
function openMention() {
  mentionMode.value = true
  if (mentionLoaded.value) return
  getSuggestFollows()
    .then((data) => {
      mentionList.value = data || []
      mentionLoaded.value = true
    })
    .catch(() => {
      mentionList.value = []
    })
}

// Insert "@nickname " into the comment text at the cursor / end, replacing
// the half-typed "@..." token, then close the popup.
function selectMention(u) {
  const text = commentText.value
  const atIdx = text.lastIndexOf('@')
  if (atIdx >= 0) {
    commentText.value = text.slice(0, atIdx) + '@' + u.nickname + ' '
  } else {
    commentText.value = text + '@' + u.nickname + ' '
  }
  mentionMode.value = false
}

// closeMention dismisses the popup (used on blur / Escape / explicit close).
function closeMention() {
  mentionMode.value = false
}

// parseMentions splits a comment into segments, marking @username tokens so
// the template can render them in the accent cyan color.
function parseMentions(content) {
  if (!content) return []
  // @name consists of word chars / dots / underscores / Chinese chars and
  // must be preceded by start-of-string or whitespace.
  const re = /(^|\s)@([\w.\u4e00-\u9fa5]+)/g
  const out = []
  let last = 0
  let m
  while ((m = re.exec(content)) !== null) {
    const ws = m[1]
    const name = m[2]
    const wsLen = ws ? ws.length : 0
    const matchStart = m.index + wsLen
    if (matchStart > last) {
      out.push({ type: 'text', value: content.slice(last, matchStart) })
    }
    if (wsLen) out.push({ type: 'text', value: ws })
    out.push({ type: 'mention', value: '@' + name })
    last = matchStart + name.length + 1
  }
  if (last < content.length) out.push({ type: 'text', value: content.slice(last) })
  return out
}

// ===================== Feature 2: Quality switch (清晰度切换) =====================

// toggleQuality flips between 标清 (sd) and 高清 (hd). The video URL is
// unchanged — purely a visual control with an HD badge.
function toggleQuality() {
  quality.value = quality.value === 'sd' ? 'hd' : 'sd'
}
// Video download/save (视频下载)
function openShareSheet(v) {
  showDialog({
    title: '分享',
    message: '分享给好友 / 保存视频 / 不感兴趣',
    showCancelButton: true,
    confirmButtonText: '分享给好友',
    cancelButtonText: '不感兴趣',
  }).then(() => {
    // "分享给好友" — use the Web Share API when available (系统转发), else
    // fall back to copying the link.
    shareToFriend(v)
  }).catch(() => {
    doDismiss(v)
  })
}
// shareToFriend invokes navigator.share() (系统分享面板). If the browser
// doesn't support it (desktop/older browsers), copy the link as a fallback.
async function shareToFriend(v) {
  if (navigator.share) {
    try {
      await navigator.share({
        title: v.title || '抖音',
        text: v.description || v.title || '',
        url: v.video_url,
      })
      showSuccessToast('已分享')
    } catch (e) {
      // User cancelled — do nothing.
    }
  } else {
    copyLink(v)
  }
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
      <span class="tab-follow" :class="{ active: activeTab === 'follow' }" @click="activeTab = 'follow'">
        关注
        <!-- Feature 2: red dot badge shown when the following feed has new videos -->
        <i v-if="followHasNew && activeTab !== 'follow'" class="follow-dot"></i>
      </span>
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
          @click="onVideoTap(i, $event)"
        ></video>
        <!-- Feature 1: Double-tap heart burst overlay — hearts render on top
             of the active slide and animate (scale + fade) via CSS. -->
        <template v-if="i === index">
          <div
            v-for="h in heartBurst"
            :key="h.id"
            class="heart-burst"
            :style="{ left: h.x + 'px', top: h.y + 'px' }"
          >❤</div>
        </template>
        <!-- Feature 2: Quality switch (清晰度切换) — top-right corner -->
        <div v-if="i === index" class="quality-toggle" @click.stop="toggleQuality">
          {{ quality === 'hd' ? '高清' : '标清' }}
        </div>
        <!-- HD badge shown only while 高清 is selected -->
        <div v-if="i === index && quality === 'hd'" class="hd-badge">HD</div>
        <!-- Feature 4: Playback speed toggle (视频慢放/倍速) — cycles 0.5/1.0/1.5/2.0 -->
        <div v-if="i === index" class="speed-toggle" @click.stop="cycleSpeed">
          {{ playbackRate }}x
        </div>
        <!-- Floating speed badge — only shown when not at normal speed -->
        <div v-if="i === index && playbackRate !== 1.0" class="speed-badge">
          {{ playbackRate }}x
        </div>
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
          <!-- Feature 3: Expandable caption (视频文案展示) — clamp to 2 lines
               with an inline 展开/收起 toggle when the text is long. -->
          <div v-if="v.description" class="desc-wrap">
            <span class="desc" :class="{ expanded: expandedDesc[v.id] }">{{ v.description }}</span>
            <span
              v-if="descIsLong(v.description)"
              class="desc-toggle"
              @click.stop="toggleDesc(v.id)"
            >{{ expandedDesc[v.id] ? '收起' : '展开' }}</span>
          </div>
          <!-- Tags rendered as rounded chips (话题标签) -->
          <div class="tags">
            <span v-for="t in (v.tags || '').split(',').filter(Boolean)" :key="t" class="tag">#{{ t }}</span>
          </div>
          <div class="music-row" @click.stop="router.push('/music/' + v.id)"><van-icon name="music-o" size="14" /><span class="music-name">{{ v.music }}</span></div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>

    <!-- ===================== Feature: First-time swipe-up guide =====================
         One-time overlay for new users. Semi-transparent dark backdrop, a large
         bouncing 👆 emoji, and hint text. Auto-dismisses after 4s or on the first
         swipe/tap. Only shown when 'dy_guide_shown' is absent in localStorage. -->
    <div v-if="showGuide" class="guide-overlay" @click.stop="dismissGuide" @touchstart.stop="dismissGuide">
      <div class="guide-finger">👆</div>
      <div class="guide-text-main">上滑切换视频</div>
      <div class="guide-text-sub">双击点赞❤️</div>
    </div>

    <!-- Comment popup -->
    <van-popup v-model:show="showComment" position="bottom" round :style="{ height: '50%' }">
      <div class="comment-panel">
        <div class="cp-head">{{ commentList.length }} 条评论</div>
        <div class="cp-list">
          <!-- ===================== Feature: Pinned comment (评论置顶) =====================
               The author's comment or the most-liked comment is pinned at the top
               with a 置顶 badge + 📌 icon and a subtle highlight background. -->
          <div v-if="pinnedComment" class="cp-item cp-item-pinned" :key="'pinned-' + pinnedComment.id">
            <img class="cp-avatar" :src="pinnedComment.avatar || 'https://via.placeholder.com/36'" />
            <div class="cp-body">
              <div class="cp-user-row">
                <span class="cp-user">{{ pinnedComment.username }}</span>
                <span class="cp-pin-tag">📌 置顶</span>
              </div>
              <div class="cp-content">
                <template v-for="(seg, si) in parseMentions(pinnedComment.content)" :key="si">
                  <span v-if="seg.type === 'mention'" class="cp-mention">{{ seg.value }}</span>
                  <span v-else>{{ seg.value }}</span>
                </template>
              </div>
              <div class="cp-reply-btn" :class="{ active: replyTo && replyTo.id === pinnedComment.id }" @click="startReply(pinnedComment)">回复</div>
              <div v-if="replyTo && replyTo.id === pinnedComment.id" class="cp-sub-input">
                <van-field
                  v-model="replyText"
                  :placeholder="'回复 @' + pinnedComment.username"
                  class="cp-field"
                  @keyup.enter="sendReply"
                />
                <van-button size="mini" type="primary" color="#fe2c55" @click="sendReply">发送</van-button>
              </div>
            </div>
            <div class="cp-like" :class="{ active: pinnedComment.liked }" @click="doCommentLike(pinnedComment)">
              <van-icon :name="pinnedComment.liked ? 'like' : 'like-o'" size="16" :color="pinnedComment.liked ? '#fe2c55' : '#999'" /><span>{{ pinnedComment.likes }}</span>
            </div>
          </div>
          <div
            v-for="c in regularComments"
            :key="c.id"
            class="cp-item"
            :class="{ 'cp-item-child': c.parent_id && c.parent_id !== 0 }"
          >
            <img class="cp-avatar" :src="c.avatar || 'https://via.placeholder.com/36'" />
            <div class="cp-body">
              <div class="cp-user">{{ c.username }}</div>
              <div class="cp-content">
                <template v-for="(seg, si) in parseMentions(c.content)" :key="si">
                  <span v-if="seg.type === 'mention'" class="cp-mention">{{ seg.value }}</span>
                  <span v-else>{{ seg.value }}</span>
                </template>
              </div>
              <div class="cp-reply-btn" :class="{ active: replyTo && replyTo.id === c.id }" @click="startReply(c)">回复</div>
              <!-- Inline sub-input shown when replying to this comment -->
              <div v-if="replyTo && replyTo.id === c.id" class="cp-sub-input">
                <van-field
                  v-model="replyText"
                  :placeholder="'回复 @' + c.username"
                  class="cp-field"
                  @keyup.enter="sendReply"
                />
                <van-button size="mini" type="primary" color="#fe2c55" @click="sendReply">发送</van-button>
              </div>
            </div>
            <div class="cp-like" :class="{ active: c.liked }" @click="doCommentLike(c)">
              <van-icon :name="c.liked ? 'like' : 'like-o'" size="16" :color="c.liked ? '#fe2c55' : '#999'" /><span>{{ c.likes }}</span>
            </div>
          </div>
          <div v-if="!commentList.length" class="cp-empty">暂无评论，来说点什么吧</div>
        </div>
        <div class="cp-input">
          <van-field
            v-model="commentText"
            placeholder="说点什么，用 @ 提及好友"
            class="cp-field"
            @input="onCommentInput"
            @keyup.enter="sendComment"
            @blur="() => setTimeout(closeMention, 150)"
          />
          <van-button size="small" type="primary" color="#fe2c55" @click="sendComment">发送</van-button>
        </div>
        <!-- @mention suggestion popup (评论区at提及) -->
        <div v-if="mentionMode" class="mention-popup">
          <div class="mp-head">
            <span>选择提及的用户</span>
            <van-icon name="cross" size="14" color="#999" @click="closeMention" />
          </div>
          <div class="mp-list">
            <div
              v-for="u in mentionList"
              :key="u.id"
              class="mp-item"
              @mousedown.prevent="selectMention(u)"
            >
              <img class="mp-avatar" :src="u.avatar || 'https://via.placeholder.com/32'" />
              <div class="mp-info">
                <div class="mp-name van-ellipsis">@{{ u.nickname }}</div>
                <div class="mp-fans">{{ fmtCount(u.followers_count) }} 粉丝</div>
              </div>
              <van-icon name="success" size="14" color="#25f4ee" />
            </div>
            <div v-if="!mentionList.length" class="mp-empty">暂无可提及的用户</div>
          </div>
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
/* Feature 3: Expandable caption — collapsed state clamps to 2 lines. */
.desc-wrap { margin-bottom: 6px; }
.desc-wrap .desc { display: inline; margin-bottom: 0; }
.desc-wrap .desc:not(.expanded) {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.desc-toggle {
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  margin-left: 4px;
  cursor: pointer;
  background: rgba(0,0,0,0.4);
  padding: 0 6px;
  border-radius: 8px;
  white-space: nowrap;
}
.desc-toggle:active { opacity: 0.7; }
.tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }
/* Tags as rounded chips (话题标签) with theme background. */
.tag {
  color: #fff;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(254,44,85,0.75);
  line-height: 16px;
}
.music-row { display: flex; align-items: center; gap: 6px; color: #fff; font-size: 12px; }
.loading { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); z-index: 5; }
.comment-panel { display: flex; flex-direction: column; height: 100%; background: #161616; }
.cp-head { text-align: center; padding: 14px; color: #fff; font-size: 15px; border-bottom: 1px solid #222; }
.cp-list { flex: 1; overflow-y: auto; padding: 8px 12px; }
.cp-item { display: flex; gap: 10px; padding: 12px 0; }
/* Child comments (parent_id != 0) are indented to reflect the reply nesting. */
.cp-item-child { margin-left: 40px; }
.cp-item-child .cp-avatar { width: 28px; height: 28px; }
.cp-avatar { width: 36px; height: 36px; border-radius: 50%; flex-shrink: 0; }
.cp-body { flex: 1; }
.cp-user { color: #888; font-size: 13px; }
.cp-content { color: #fff; font-size: 14px; margin-top: 3px; }
.cp-reply-btn { color: #888; font-size: 12px; margin-top: 6px; display: inline-block; cursor: pointer; }
.cp-reply-btn.active { color: #fe2c55; }
.cp-sub-input { display: flex; gap: 8px; align-items: center; margin-top: 8px; }
.cp-like { display: flex; flex-direction: column; align-items: center; color: #888; font-size: 11px; cursor: pointer; }
.cp-like.active { color: #fe2c55; }
.cp-empty { text-align: center; color: #666; padding: 40px; }
.cp-input { display: flex; gap: 8px; padding: 10px; border-top: 1px solid #222; }
.cp-field { background: #222; border-radius: 18px; }

/* ===================== Feature 1: @mention styles ===================== */
/* Highlighted @username inside rendered comments */
.cp-mention { color: #25f4ee; font-weight: 500; }
/* Suggestion popup anchored above the comment input */
.mention-popup { position: absolute; left: 0; right: 0; bottom: 60px; z-index: 30; background: #1f1f1f; border-top: 1px solid #2a2a2a; border-bottom: 1px solid #2a2a2a; max-height: 200px; display: flex; flex-direction: column; }
.mp-head { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; color: #999; font-size: 12px; border-bottom: 1px solid #2a2a2a; }
.mp-list { overflow-y: auto; }
.mp-item { display: flex; align-items: center; gap: 10px; padding: 8px 12px; cursor: pointer; }
.mp-item:active { background: #2a2a2a; }
.mp-avatar { width: 32px; height: 32px; border-radius: 50%; flex-shrink: 0; }
.mp-info { flex: 1; min-width: 0; }
.mp-name { color: #fff; font-size: 13px; }
.mp-fans { color: #888; font-size: 11px; margin-top: 2px; }
.mp-empty { text-align: center; color: #666; padding: 20px; font-size: 12px; }

/* ===================== Feature 2: Quality switch styles ===================== */
/* Toggle button in the top-right corner of the active slide */
.quality-toggle { position: absolute; top: 52px; right: 14px; z-index: 12; padding: 4px 10px; font-size: 12px; color: #fff; background: rgba(0,0,0,0.45); border: 1px solid rgba(255,255,255,0.35); border-radius: 12px; backdrop-filter: blur(4px); cursor: pointer; user-select: none; }
/* HD badge — accent cyan, shown when 高清 is selected */
.hd-badge { position: absolute; top: 52px; right: 64px; z-index: 12; padding: 2px 6px; font-size: 11px; font-weight: bold; color: #0a1f1f; background: #25f4ee; border-radius: 4px; letter-spacing: 0.5px; }

/* ===================== Feature 4: Playback speed (视频慢放/倍速) styles ===================== */
/* Speed toggle button, positioned just below the quality toggle */
.speed-toggle {
  position: absolute;
  top: 84px;
  right: 14px;
  z-index: 12;
  padding: 4px 10px;
  font-size: 12px;
  color: #fff;
  background: rgba(0,0,0,0.45);
  border: 1px solid rgba(255,255,255,0.35);
  border-radius: 12px;
  backdrop-filter: blur(4px);
  cursor: pointer;
  user-select: none;
}
.speed-toggle:active { background: rgba(254,44,85,0.7); }
/* Floating speed badge — theme color, only while a non-normal rate is active */
.speed-badge {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 11;
  padding: 6px 16px;
  font-size: 20px;
  font-weight: bold;
  color: #fff;
  background: rgba(254,44,85,0.85);
  border-radius: 20px;
  pointer-events: none;
  box-shadow: 0 2px 12px rgba(254,44,85,0.6);
  animation: speedBadgeIn 0.2s ease-out;
}
@keyframes speedBadgeIn { from { opacity: 0; transform: translate(-50%, -50%) scale(0.8); } to { opacity: 1; transform: translate(-50%, -50%) scale(1); } }

/* ===================== Feature 1: Double-tap heart burst ===================== */
/* The burst heart is anchored at the tap point and runs an 800ms animation:
   scale 0→1.2 (fade in), then 1.2→1.5 (fade out). Pointer events are disabled
   so it never blocks the underlying video tap. */
.heart-burst {
  position: absolute;
  z-index: 15;
  font-size: 100px;
  color: #fe2c55;
  transform: translate(-50%, -50%);
  pointer-events: none;
  text-shadow: 0 4px 24px rgba(0, 0, 0, 0.35);
  animation: heartBurstAnim 0.8s ease-out forwards;
}
@keyframes heartBurstAnim {
  0% { transform: translate(-50%, -50%) scale(0); opacity: 0; }
  35% { transform: translate(-50%, -50%) scale(1.2); opacity: 1; }
  100% { transform: translate(-50%, -50%) scale(1.5); opacity: 0; }
}

/* ===================== Feature 2: Follow tab unread badge ===================== */
/* The 关注 tab label is positioned relative so the 8px red dot can sit at its
   top-right corner. */
.top-tabs .tab-follow { position: relative; cursor: pointer; }
.follow-dot {
  position: absolute;
  top: -2px;
  right: -10px;
  width: 8px;
  height: 8px;
  background: #fe2c55;
  border-radius: 50%;
  display: inline-block;
}

/* ===================== Feature: First-time swipe-up guide ===================== */
/* Full-screen semi-transparent overlay. The finger emoji bounces upward
   repeatedly; tapping/swiping (or 4s timeout) dismisses it for good. */
.guide-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  text-align: center;
}
.guide-finger {
  font-size: 72px;
  line-height: 1;
  animation: guideBounce 1.2s ease-in-out infinite;
  text-shadow: 0 6px 18px rgba(0, 0, 0, 0.4);
}
.guide-text-main {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 1px;
}
.guide-text-sub {
  color: rgba(255, 255, 255, 0.85);
  font-size: 14px;
}
/* The finger translates up then settles back, suggesting a swipe-up gesture. */
@keyframes guideBounce {
  0%   { transform: translateY(0) scale(1); opacity: 0.85; }
  45%  { transform: translateY(-32px) scale(1.05); opacity: 1; }
  60%  { transform: translateY(-32px) scale(1.05); opacity: 1; }
  100% { transform: translateY(0) scale(1); opacity: 0.85; }
}

/* ===================== Feature: Pinned comment (评论置顶) ===================== */
/* Subtle highlight + left accent border to set the pinned comment apart. */
.cp-item-pinned {
  background: rgba(254, 44, 85, 0.08);
  border-left: 3px solid #fe2c55;
  border-radius: 8px;
  padding-left: 10px;
  margin-bottom: 4px;
}
.cp-user-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.cp-pin-tag {
  display: inline-flex;
  align-items: center;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
  background: #fe2c55;
  padding: 1px 6px;
  border-radius: 4px;
  line-height: 16px;
}
</style>
