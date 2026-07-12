<script setup>
import { ref, computed, onMounted, onActivated, onUnmounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getFeed, getRecommendFeed, getFollowingFeed, recordPlay, toggleLike, toggleFavorite, toggleFollow, getComments, createComment, likeComment, reportVideo, dismissVideo, getSuggestFollows, getProfile } from '../api'
import { playLike, playUnlike, playComment, playFollow } from '../utils/sound'

const router = useRouter()

// ===================== Feature: Sound effects toggle (操作音效反馈) =====================
// A 🔊/🔇 button in the top bar controls whether UI interaction sounds play.
// The preference persists in localStorage 'dy_sound'; default ON.
const SOUND_KEY = 'dy_sound'
const soundOn = ref(true)
function loadSoundPref() {
  try {
    const v = localStorage.getItem(SOUND_KEY)
    // Treat explicit '0' as off; anything else (including unset) as on.
    soundOn.value = v !== '0'
  } catch (e) {
    soundOn.value = true
  }
}
function toggleSound() {
  soundOn.value = !soundOn.value
  try {
    localStorage.setItem(SOUND_KEY, soundOn.value ? '1' : '0')
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
}
// playSound wraps each effect so it only fires when the toggle is on.
function playSound(fn) {
  if (soundOn.value) fn()
}
const videos = ref([])
const index = ref(0)
const loading = ref(true)

// ===================== Feature: Video thumbnail timeline (视频缩略图时间轴) =====================
// A thin bar at the very top (below the tabs) showing one dot per loaded video.
// The current video's dot is enlarged + themed; watched videos (every index the
// user has already passed through) get a subtle fill. Clicking a dot jumps to it.
// watchedSet holds every index that has ever been the active video, so a dot
// stays filled even after the user swipes back up.
const watchedSet = ref(new Set([0]))
const activeTab = ref('recommend')
const showComment = ref(false)
const commentList = ref([])
const commentText = ref('')
const replyText = ref('')
const replyTo = ref(null) // comment being replied to (null = top-level)
const currentVideoId = ref(null)

// ---- Feature: Comment sort options (评论排序) ----
// 'default' keeps the server order (backend returns by likes desc). 'hot' re-
// sorts top-level comments by likes desc; 'new' sorts by id desc (newest first).
// The sort is applied to the regular (non-pinned) comment list only.
const commentSort = ref('default')

// ---- Feature: Comment emoji reactions (评论表情回应) ----
// Purely frontend state — reactions reset when the comment popup closes.
// commentReactions: { [commentId]: { [emoji]: count } }
// userReactions:    { [commentId]: emoji | null } (which emoji the current user picked)
const REACTION_EMOJIS = ['👍', '❤️', '😂', '🔥', '👏']
const commentReactions = ref({})
const userReactions = ref({})

// ===================== Feature: Comment poll voting (评论投票) =====================
// Upvote / downvote buttons shown next to each comment, separate from the emoji
// reactions. State is purely frontend and resets when the comment popup closes.
//   commentVotes:    { [commentId]: { up: number, down: number } } running tallies
//   userVote:        { [commentId]: 'up' | 'down' | null } the current user's pick
// Only one vote per comment is allowed: voting up cancels a prior downvote and
// vice-versa; tapping the active vote again removes it.
const commentVotes = ref({})
const userVote = ref({})

// voteScore returns (upvotes - downvotes) for a comment, seeded from the
// comment's own like count so the score never looks empty before anyone votes.
function voteScore(c) {
  const base = (c && c.likes) || 0
  const v = commentVotes.value[c && c.id]
  if (!v) return base
  return base + (v.up || 0) - (v.down || 0)
}
// ensureVotes makes sure a comment has a vote tally object before we mutate it.
function ensureVotes(c) {
  if (!c) return
  if (!commentVotes.value[c.id]) {
    commentVotes.value = { ...commentVotes.value, [c.id]: { up: 0, down: 0 } }
  }
}
// castVote records an upvote or downvote. Tapping the already-active vote
// removes it; tapping the opposite vote swaps it (cancelling the old one).
function castVote(c, kind) {
  if (!c) return
  ensureVotes(c)
  const prev = userVote.value[c.id]
  const tally = { ...commentVotes.value[c.id] }
  const nextUser = { ...userVote.value }
  // Reverse whatever the previous vote contributed.
  if (prev === 'up') tally.up = Math.max(0, (tally.up || 0) - 1)
  if (prev === 'down') tally.down = Math.max(0, (tally.down || 0) - 1)
  if (prev === kind) {
    // Same button tapped again → retract the vote.
    nextUser[c.id] = null
  } else {
    // New vote (or switched) → apply it.
    tally[kind] = (tally[kind] || 0) + 1
    nextUser[c.id] = kind
  }
  commentVotes.value = { ...commentVotes.value, [c.id]: tally }
  userVote.value = nextUser
}
function voteUp(c) { castVote(c, 'up') }
function voteDown(c) { castVote(c, 'down') }

// ===================== Feature: Comment quick emoji bar (快捷表情栏) =====================
// A horizontal bar of 8 quick emojis above the comment input. Tapping one
// appends it to the comment text. "😊更多" expands a second row of 16 more.
const QUICK_EMOJIS = ['😂', '🔥', '❤️', '👍', '👏', '😍', '🎉', '💀']
const MORE_EMOJIS = ['🤣', '😎', '🥳', '😭', '🤔', '😅', '🥰', '😋', '🤩', '🤯', '😱', '🙏', '💪', '✨', '💯', '🌹']
const moreExpanded = ref(false)
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

// ===================== Feature: Comment character limit (评论字数限制) =====================
// Comments are capped at 200 characters. While typing we show a live
// "X/200" counter; once the limit is exceeded the counter turns red and shows
// "超出字数限制" with the current length, and the send button is disabled so an
// over-limit comment can never be submitted. The threshold is shared by the
// top-level comment input and the inline reply inputs.
const COMMENT_MAX = 200
const commentOverLimit = computed(() => commentText.value.length > COMMENT_MAX)
const replyOverLimit = computed(() => replyText.value.length > COMMENT_MAX)

// ===================== Feature: Video collection playlist (视频合集播放列表) =====================
// A bottom-sheet popup that lists every loaded video as a sequential playlist.
// Each row shows cover + title + duration, the current video is highlighted,
// and tapping a row jumps to it. "顺序播放" (sequential play) auto-advances to
// the next video when the current one ends, by extending the onended handler.
const showPlaylist = ref(false) // popup visibility
const sequentialPlay = ref(false) // 顺序播放 toggle — auto-advance on end

// ===================== Feature: Video playback loop mode (视频循环播放模式) =====================
// A 🔁 toggle next to the speed/quality buttons. When ON the current video
// loops continuously (video.loop = true) regardless of the sequential-play
// setting, and a "单曲循环" badge is shown. When OFF the existing behavior is
// preserved (loop by default, or auto-advance when sequential play is on).
// The preference persists in localStorage 'dy_loop_mode'.
const LOOP_KEY = 'dy_loop_mode'
const loopMode = ref(false)
function loadLoopPref() {
  try {
    loopMode.value = localStorage.getItem(LOOP_KEY) === '1'
  } catch (e) {
    loopMode.value = false
  }
}
function toggleLoop() {
  loopMode.value = !loopMode.value
  try {
    localStorage.setItem(LOOP_KEY, loopMode.value ? '1' : '0')
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
  // Re-apply the looping state to the active video immediately so the toggle
  // takes effect without needing to re-slide.
  const vids = document.querySelectorAll('.feed-video')
  const v = vids[index.value]
  if (v) v.loop = loopMode.value || !sequentialPlay.value
}

// ---- Feature 4: Playback speed (视频慢放/倍速) ----
// Cycles through 0.5x / 1.0x / 1.5x / 2.0x. Applied to the active video
// element immediately on change and re-applied in playCurrent() so the
// rate persists across slides.
//
// ===================== Feature: Video speed memory (播放速度记忆) =====================
// The chosen speed is persisted to localStorage 'dy_playback_speed' and restored
// on mount so the user's preference carries across sessions. On restore we apply
// the rate to all videos and surface a brief "已恢复 X.X倍速" toast.
const SPEED_KEY = 'dy_playback_speed'
const SPEEDS = [0.5, 1.0, 1.5, 2.0]
const playbackRate = ref(1.0)
function cycleSpeed() {
  const cur = SPEEDS.indexOf(playbackRate.value)
  playbackRate.value = SPEEDS[(cur + 1) % SPEEDS.length]
  // Apply immediately to the currently-playing video element.
  const vids = document.querySelectorAll('.feed-video')
  const v = vids[index.value]
  if (v) v.playbackRate = playbackRate.value
  // Feature: 播放速度记忆 — persist the new speed so it restores next session.
  try {
    localStorage.setItem(SPEED_KEY, String(playbackRate.value))
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
}

// restorePlaybackSpeed reads the saved speed, applies it to every loaded video
// element, and toasts the restored value. Called once on mount.
function restorePlaybackSpeed() {
  let saved = null
  try {
    saved = localStorage.getItem(SPEED_KEY)
  } catch (e) {
    saved = null
  }
  if (saved == null) return // nothing saved → keep the 1.0 default
  const rate = parseFloat(saved)
  if (isNaN(rate) || !SPEEDS.includes(rate)) return // invalid → ignore
  playbackRate.value = rate
  // Apply to all currently-rendered video elements.
  document.querySelectorAll('.feed-video').forEach((v) => { v.playbackRate = rate })
  showToast('已恢复 ' + rate.toFixed(1) + '倍速')
}

// ===================== Feature: Video wallpaper mode (视频壁纸模式) =====================
// A 🖼️ button in the top tabs enters "wallpaper/screensaver" mode: every UI
// overlay (top tabs, action rail, bottom info, progress bar, quality/speed
// toggles) is hidden via v-if, so the video plays fullscreen with no controls
// and loops continuously. Only a small "退出" button remains in the top-right
// corner to return to normal mode.
const wallpaperMode = ref(false)
function enterWallpaper() {
  wallpaperMode.value = true
  // Ensure the active video keeps looping (it already does by default) and
  // resumes if it was paused, so the wallpaper always animates.
  nextTick(() => {
    const vids = document.querySelectorAll('.feed-video')
    const v = vids[index.value]
    if (v) { v.loop = true; v.play().catch(() => {}) }
  })
}
function exitWallpaper() {
  wallpaperMode.value = false
}

// ===================== Feature: Video sleep timer (视频睡眠定时) =====================
// A 😴 button in the top tabs opens a small popup with options 15分钟/30分钟/
// 60分钟/关闭. After the chosen duration, every feed video is paused and a
// toast "定时已到，已暂停播放" is shown. While active, a countdown badge
// "剩余 N分钟" is rendered next to the 😴 button and updates every minute.
// Selecting 关闭 (or leaving the page) cancels the pending timer.
const SLEEP_OPTIONS = [15, 30, 60] // minutes
const sleepMenu = ref(false)        // options popup visibility
const sleepActive = ref(false)      // true while a timer is running
const sleepRemaining = ref(0)       // whole minutes left, shown in the badge
let sleepTimer = null               // the setTimeout id for the final pause
let sleepTickTimer = null           // the setInterval that decrements the badge

// pauseAllVideos stops every loaded feed video. Used when the timer fires.
function pauseAllVideos() {
  document.querySelectorAll('.feed-video').forEach((v) => v.pause())
}

// startSleep arms the timer for the given number of minutes: schedules the
// pause callback, sets up the per-minute countdown tick, and closes the menu.
function startSleep(minutes) {
  sleepMenu.value = false
  clearSleep()
  sleepActive.value = true
  sleepRemaining.value = minutes
  // Decrement the "剩余 N分钟" badge every minute so it stays accurate.
  sleepTickTimer = setInterval(() => {
    if (sleepRemaining.value > 0) sleepRemaining.value--
  }, 60 * 1000)
  // After the full duration, pause everything and surface a confirmation.
  sleepTimer = setTimeout(() => {
    pauseAllVideos()
    clearSleep()
    sleepActive.value = false
    sleepRemaining.value = 0
    showToast('定时已到，已暂停播放')
  }, minutes * 60 * 1000)
  showSuccessToast(`已开启睡眠定时 ${minutes} 分钟`)
}

// cancelSleep is the 关闭 option: clears the running timer + badge.
function cancelSleep() {
  sleepMenu.value = false
  clearSleep()
  sleepActive.value = false
  sleepRemaining.value = 0
  showToast('已关闭睡眠定时')
}

// clearSleep tears down both timers without touching the public flags. It is
// also called from onUnmounted so the timer never outlives the component.
function clearSleep() {
  if (sleepTimer) {
    clearTimeout(sleepTimer)
    sleepTimer = null
  }
  if (sleepTickTimer) {
    clearInterval(sleepTickTimer)
    sleepTickTimer = null
  }
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

// ===================== Feature 1: Video stats overlay (视频数据浮层) =====================
// A long-press (touchstart held > 500ms) or a right-click on the video opens
// a semi-transparent card centered on the video showing plays/likes/comments/
// shares/music/tags. It auto-dismisses after 3s or on a tap, and does not fire
// for a swipe — the page-level onTouchStart handler continues to own swipe
// detection; the press is cancelled if the finger drifts or lifts early.
const showStats = ref(false)        // overlay visibility
const statsIndex = ref(-1)          // slide the overlay is bound to
let pressTimer = null               // long-press timer armed on touchstart
let pressStartX = 0                 // touchstart X, to cancel on large move
let pressStartY = 0                 // touchstart Y, to cancel on large move
let statsDismissTimer = null        // 3s auto-dismiss timer
const STATS_MOVE_TOLERANCE = 10     // px; cancel the press if the finger drifts

// The video currently targeted by the overlay (resolved reactively so the
// template can read its fields). Stays in sync with statsIndex.
const statsVideo = computed(() => {
  if (!showStats.value || statsIndex.value < 0) return null
  return videos.value[statsIndex.value] || null
})

// armPress starts the 500ms timer that opens the stats overlay. If the finger
// moves beyond the tolerance or lifts before the timer fires, the press is a
// no-op (treated as an ordinary tap/swipe instead).
function armPress(i, e) {
  cancelPress()
  const t = (e.touches && e.touches[0]) || e
  pressStartX = t.clientX
  pressStartY = t.clientY
  pressTimer = setTimeout(() => {
    pressTimer = null
    openStats(i)
  }, 500)
}

// cancelPress clears a pending long-press (e.g. the finger lifted early or
// drifted). It does not close an already-open overlay.
function cancelPress() {
  if (pressTimer) {
    clearTimeout(pressTimer)
    pressTimer = null
  }
}

// onVideoTouchMove cancels the press when the finger drifts past the tolerance,
// so a swipe never accidentally triggers the overlay.
function onVideoTouchMove(e) {
  if (!pressTimer) return
  const t = (e.touches && e.touches[0]) || e
  if (Math.abs(t.clientX - pressStartX) > STATS_MOVE_TOLERANCE ||
      Math.abs(t.clientY - pressStartY) > STATS_MOVE_TOLERANCE) {
    cancelPress()
  }
}

// openStats binds the overlay to a slide and arms the 3s auto-dismiss timer.
function openStats(i) {
  statsIndex.value = i
  showStats.value = true
  if (statsDismissTimer) clearTimeout(statsDismissTimer)
  statsDismissTimer = setTimeout(closeStats, 3000)
}

// closeStats hides the overlay and clears the dismiss timer.
function closeStats() {
  showStats.value = false
  if (statsDismissTimer) {
    clearTimeout(statsDismissTimer)
    statsDismissTimer = null
  }
}

// onVideoContextMenu opens the overlay on right-click (desktop) and suppresses
// the native browser context menu.
function onVideoContextMenu(i, e) {
  e.preventDefault()
  cancelPress()
  openStats(i)
}

// ===================== Feature 2: Mood ring indicator (心情光环) =====================
// The avatar in the action rail is wrapped in a rotating conic-gradient ring
// whose colors reflect the video's "mood", derived from the like ratio
// (likes/plays). Returns a class used to pick the gradient palette.
//   ratio > 0.3   → mood-hot     (warm red/orange) = 热门
//   0.1–0.3       → mood-popular (cool blue/cyan)  = 受欢迎
//   < 0.1         → mood-normal  (subtle gray)      = 普通
function moodClass(v) {
  if (!v) return 'mood-normal'
  const plays = v.plays || 0
  const ratio = plays > 0 ? (v.likes || 0) / plays : 0
  if (ratio > 0.3) return 'mood-hot'
  if (ratio >= 0.1) return 'mood-popular'
  return 'mood-normal'
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

// ===================== Feature: Feed vertical scroll hint (上滑提示) =====================
// A small floating "👆上滑" chip shown once to first-time users with a bounce
// animation. It is distinct from the full guide overlay (dy_guide_shown): this
// is a lightweight persistent hint that only dismisses on the user's first
// actual swipe. The dismissed state persists in localStorage 'dy_scroll_hint'.
const SCROLL_HINT_KEY = 'dy_scroll_hint'
const showScrollHint = ref(false)

// loadScrollHint shows the hint on mount unless it has already been dismissed.
function loadScrollHint() {
  try {
    if (localStorage.getItem(SCROLL_HINT_KEY) !== '1') {
      showScrollHint.value = true
    }
  } catch (e) {
    // localStorage unavailable — show by default.
    showScrollHint.value = true
  }
}

// dismissScrollHint hides the chip and persists the dismissed flag so it never
// reappears for this user. Called on the first detected swipe.
function dismissScrollHint() {
  if (!showScrollHint.value) return
  showScrollHint.value = false
  try {
    localStorage.setItem(SCROLL_HINT_KEY, '1')
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
//
// ===================== Feature: Comment pin by creator (作者置顶评论) =====================
// On top of the auto-pinning, if the *current* logged-in user is the video
// author they see a "📌置顶" button on their own top-level comments. Tapping it
// manually pins that comment; only one manual pin is allowed per video, so
// pinning a new comment replaces the previous one. The manual pin is stored
// per-video in localStorage 'dy_pin_<videoId>' and takes precedence over the
// automatic pin. Clearing the pin removes the stored id.
const PIN_PREFIX = 'dy_pin_'
const manualPinId = ref(null) // comment id the author manually pinned for the open video
const pinnedComment = ref(null)
const regularComments = ref([])

// The id of the currently logged-in user (resolved once from the profile API),
// used to decide whether to show the "📌置顶" button on the author's own comments.
const currentUserId = ref(null)

// Returns true when the current user is the author of the active video — only
// then do we render the pin buttons on their own comments.
function canPinAsAuthor() {
  const v = videos.value[index.value]
  const authorId = v ? v.author_id : null
  return authorId != null && currentUserId.value != null && String(currentUserId.value) === String(authorId)
}

// Whether a given comment belongs to the current user (so the pin button shows).
function isOwnComment(c) {
  return c && c.user_id != null && currentUserId.value != null && String(c.user_id) === String(currentUserId.value)
}

// Whether a given (own, top-level) comment is currently the manual pin.
function isManualPin(c) {
  return c && manualPinId.value != null && String(c.id) === String(manualPinId.value)
}

// pinComment stores the chosen comment as the manual pin for this video (one
// only), re-renders the pinned section, and surfaces a toast.
function pinComment(c) {
  if (!canPinAsAuthor() || !isOwnComment(c)) return
  // Child comments cannot be pinned.
  if (c.parent_id && c.parent_id !== 0) { showToast('只能置顶顶级评论'); return }
  manualPinId.value = c.id
  const v = videos.value[index.value]
  if (v) {
    try { localStorage.setItem(PIN_PREFIX + v.id, String(c.id)) } catch (e) {}
  }
  showSuccessToast('已置顶评论')
  recomputePinned()
}

// unpinComment clears the manual pin for this video.
function unpinComment() {
  manualPinId.value = null
  const v = videos.value[index.value]
  if (v) {
    try { localStorage.removeItem(PIN_PREFIX + v.id) } catch (e) {}
  }
  showToast('已取消置顶')
  recomputePinned()
}

// loadManualPin reads any saved manual pin for the open video. Called when the
// comment popup opens for a video so the author's prior pin is restored.
function loadManualPin() {
  const v = videos.value[index.value]
  if (!v) { manualPinId.value = null; return }
  try {
    manualPinId.value = localStorage.getItem(PIN_PREFIX + v.id) || null
  } catch (e) {
    manualPinId.value = null
  }
}

// recomputePinned splits commentList into a pinned entry + the rest. The author's
// manually-pinned comment wins; otherwise the author's own top-level comment; and
// finally the top-level comment with the most likes. Child comments are never pinned.
function recomputePinned() {
  const list = commentList.value || []
  const current = videos.value[index.value]
  const authorId = current ? current.author_id : null
  const topLevel = list.filter((c) => !c.parent_id || c.parent_id === 0)

  let pinned = null
  // 1) The author's manual pin takes precedence (if it still exists in the list).
  if (manualPinId.value != null) {
    pinned = list.find((c) => String(c.id) === String(manualPinId.value)) || null
  }
  // 2) Otherwise the author's own top-level comment.
  if (!pinned && authorId != null) {
    pinned = topLevel.find((c) => c.user_id === authorId) || null
  }
  // 3) Otherwise the most-liked top-level comment.
  if (!pinned && topLevel.length) {
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

// ---- Feature: Comment sort options (评论排序) ----
// sortedComments applies the selected sort to the regular comment list. To keep
// reply nesting intact, we order by the top-level comments and re-attach each
// child (parent_id != 0) right after its parent. 'default' preserves server
// order, 'hot' sorts top-level by likes desc, 'new' by id desc.
const sortedComments = computed(() => {
  const sort = commentSort.value
  const list = regularComments.value
  if (sort === 'default') return list
  const tops = list.filter((c) => !c.parent_id || c.parent_id === 0)
  const children = list.filter((c) => c.parent_id && c.parent_id !== 0)
  if (sort === 'hot') {
    tops.sort((a, b) => (b.likes || 0) - (a.likes || 0))
  } else if (sort === 'new') {
    tops.sort((a, b) => (b.id || 0) - (a.id || 0))
  }
  // Re-attach each child after its parent; children keep their relative order.
  const out = []
  for (const t of tops) {
    out.push(t)
    const kids = children.filter((c) => c.parent_id === t.id)
    out.push(...kids)
  }
  // Orphans (child whose parent was the pinned comment) stay at the end.
  const placed = new Set(out.map((c) => c.id))
  children.filter((c) => !placed.has(c.id)).forEach((c) => out.push(c))
  return out
})

// ===================== Feature: Auto-pause on scroll away (滑出自动暂停) =====================
// When the tab/page becomes hidden (user switched tabs, minimized, etc.) the
// current video is paused automatically via the Page Visibility API. On return
// we surface a small "已为你暂停，点击继续" overlay; tapping it resumes playback.
const pausedByVisibility = ref(false)
function handleVisibilityChange() {
  const vids = document.querySelectorAll('.feed-video')
  const v = vids[index.value]
  if (document.visibilityState === 'hidden') {
    // Pause the active video and remember that we auto-paused it so we know
    // to offer the resume overlay when the user comes back.
    if (v && !v.paused) {
      v.pause()
      pausedByVisibility.value = true
    }
  } else if (document.visibilityState === 'visible') {
    // Back to the tab: if we auto-paused, keep it paused and show the
    // "tap to continue" overlay (resume happens in resumeAfterVisibility()).
    // If nothing was auto-paused there's nothing to do.
  }
}
// resumeAfterVisibility resumes the active video and hides the overlay.
function resumeAfterVisibility() {
  if (!pausedByVisibility.value) return
  pausedByVisibility.value = false
  const vids = document.querySelectorAll('.feed-video')
  const v = vids[index.value]
  if (v) v.play().catch(() => {})
}

onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
})
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
// Feature: 上滑提示 — show the floating scroll hint to first-time users on mount.
onMounted(() => loadScrollHint())
// Restore the user's sound-effects preference on mount.
onMounted(() => loadSoundPref())
// Feature: 播放速度记忆 — restore the saved playback speed + toast on mount.
onMounted(() => restorePlaybackSpeed())
// Feature: 视频循环播放模式 — restore the saved loop-mode preference on mount.
onMounted(() => loadLoopPref())
// Feature: 作者置顶评论 — resolve the current user id once on mount so we can
// show the pin button on the author's own comments.
onMounted(() => {
  if (!localStorage.getItem('dy_token')) return
  getProfile()
    .then((u) => { currentUserId.value = u && (u.id != null ? u.id : u.user_id) })
    .catch(() => { /* not logged in or transient — pin buttons stay hidden */ })
})
// Feature: 策展模式 — restore the saved curator-mode preference on mount.
onMounted(() => loadCuratorPref())
// Feature: 评论时间戳 — refresh relative times every minute.
onMounted(() => {
  commentTimeTimer = setInterval(() => { nowTick.value = Date.now() }, 60000)
})

onUnmounted(() => {
  stopFollowCheck()
  // Feature: 滑出自动暂停 — clean up the visibility listener.
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  pausedByVisibility.value = false
  if (singleTapTimer) clearTimeout(singleTapTimer)
  if (guideTimer) clearTimeout(guideTimer)
  // Feature 1: clear any pending long-press / stats-dismiss timers.
  cancelPress()
  if (statsDismissTimer) clearTimeout(statsDismissTimer)
  // Feature: 视频睡眠定时 — tear down the sleep timer + tick so they don't
  // fire or leak after the component unmounts.
  clearSleep()
  // Feature: 视频书签时间点 — clear any pending long-press timer.
  cancelBookmarkPress()
  // Feature: 评论草稿自动保存 — clear the debounce timer so it doesn't fire post-unmount.
  if (draftDebounce) clearTimeout(draftDebounce)
  // Feature: 评论时间戳 — clear the per-minute refresh timer.
  if (commentTimeTimer) { clearInterval(commentTimeTimer); commentTimeTimer = null }
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
      // Feature: 顺序播放 — disable native looping so the 'ended' event fires
      // and we can auto-advance. Re-enable looping when sequential play is off
      // so the active video keeps looping as before.
      // Feature: 视频循环播放模式 (loop mode) — when ON, force looping and
      // suppress auto-advance so the current video plays on repeat.
      v.loop = loopMode.value || !sequentialPlay.value
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
        // Feature: 顺序播放 (sequential play) — when the toggle is ON, the
        // video plays through once instead of looping, and on end we auto-
        // advance to the next loaded video. When OFF the original reportPlay
        // behavior is preserved. We remove the `loop` attribute while
        // sequential play is active so 'ended' actually fires.
        v.onended = () => {
          reportPlay(vid.id, 1.0)
          // Feature: 视频循环播放模式 — never auto-advance while loop mode is on
          // (the video keeps replaying via video.loop = true).
          if (!loopMode.value && sequentialPlay.value) {
            // Advance to the next video if one is available.
            if (index.value < videos.value.length - 1) {
              index.value++
              playCurrent()
            }
          }
        }
      }
      // Reset + wire the progress bar.
      progress.value = 0
      // Feature: 视频书签时间点 — load this video's saved bookmarks when it becomes active.
      loadBookmarks()
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
    // Feature: 上滑提示 — the first real swipe dismisses the floating hint.
    dismissScrollHint()
    index.value++
    playCurrent()
  } else if (dy > 50 && index.value > 0) {
    // Feature: 上滑提示 — any swipe (including up-to-go-back) counts as dismissal.
    dismissScrollHint()
    index.value--
    playCurrent()
  }
}

async function doLike(v) {
  try {
    const res = await toggleLike(v.id)
    v.liked = res.liked
    v.likes = res.likes
    // Feature: 操作音效反馈 — ascending on like, descending on unlike.
    playSound(res.liked ? playLike : playUnlike)
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
    // Feature: 操作音效反馈 — ascending three-tone on follow.
    if (res.following) playSound(playFollow)
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}

async function openComments(v) {
  currentVideoId.value = v.id
  replyTo.value = null
  replyText.value = ''
  // Reset the sort to default each time the popup is opened for a new video.
  commentSort.value = 'default'
  showComment.value = true
  // Feature: 评论草稿自动保存 — restore any saved draft for this video.
  restoreDraft()
  // Feature: 作者置顶评论 — restore any saved manual pin for this video.
  loadManualPin()
  try {
    commentList.value = await getComments(v.id)
  } catch (e) {
    commentList.value = []
  }
  // Recompute the pinned comment for the now-open video's comment list.
  recomputePinned()
}
// Reset emoji reactions when the comment popup closes (frontend-only state).
watch(showComment, (open) => {
  if (!open) {
    commentReactions.value = {}
    userReactions.value = {}
    // Feature: 评论投票 — clear vote state when the popup closes.
    commentVotes.value = {}
    userVote.value = {}
    // Feature: 评论翻译指示 — clear demo translation state when the popup closes.
    translatedIds.value = new Set()
    translatingList.value = []
  }
})
// Start composing a reply to a specific comment (or cancel if already replying).
function startReply(c) {
  if (replyTo.value && replyTo.value.id === c.id) {
    replyTo.value = null
  } else {
    replyTo.value = c
    replyText.value = ''
  }
}
// Feature: 快捷表情栏 — append a tapped emoji to the comment text.
function appendEmoji(emoji) {
  commentText.value += emoji
}
async function sendComment() {
  if (!commentText.value.trim()) return
  // Feature: 评论字数限制 — block submission while over the 200-char cap.
  if (commentOverLimit.value) { showToast('评论超出字数限制'); return }
  try {
    const cm = await createComment({ video_id: currentVideoId.value, content: commentText.value })
    commentList.value.unshift(cm)
    commentText.value = ''
    const v = videos.value[index.value]
    if (v) v.comments_count++
    // Feature: 评论草稿自动保存 — clear the draft once the comment is sent.
    clearDraft()
    recomputePinned()
    // Feature: 操作音效反馈 — single tone when a comment is sent.
    playSound(playComment)
  } catch (e) {
    showToast('请先登录')
  }
}
// Submit a reply to the comment currently held in replyTo; passes parent_id so
// the backend stores it as a nested child comment.
async function sendReply() {
  if (!replyText.value.trim() || !replyTo.value) return
  // Feature: 评论字数限制 — block reply submission while over the cap.
  if (replyOverLimit.value) { showToast('评论超出字数限制'); return }
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

// ===================== Feature: Comment emoji reactions (评论表情回应) =====================
// toggleReaction flips an emoji on/off for a comment. Selecting an already-active
// emoji removes it; selecting a different one moves the user's reaction (so the
// previous emoji count drops by 1 and the new one rises by 1). Both maps are
// mutated reactively via shallow re-assignment so Vue picks up nested changes.
function toggleReaction(c, emoji) {
  const cid = c.id
  if (!commentReactions.value[cid]) commentReactions.value[cid] = {}
  if (!userReactions.value[cid]) userReactions.value[cid] = null
  const counts = commentReactions.value[cid]
  const current = userReactions.value[cid]
  if (current === emoji) {
    // Toggle off the active emoji.
    counts[emoji] = Math.max(0, (counts[emoji] || 0) - 1)
    if (counts[emoji] === 0) delete counts[emoji]
    userReactions.value[cid] = null
  } else {
    // If the user had a different emoji, decrement it first.
    if (current) {
      counts[current] = Math.max(0, (counts[current] || 0) - 1)
      if (counts[current] === 0) delete counts[current]
    }
    counts[emoji] = (counts[emoji] || 0) + 1
    userReactions.value[cid] = emoji
  }
  // Trigger reactivity for nested object mutations.
  commentReactions.value = { ...commentReactions.value }
  userReactions.value = { ...userReactions.value }
}

// ===================== Feature: Comment translation indicator (评论翻译指示) =====================
// A pure-frontend demo translation toggle on each comment. Tapping 🌐翻译 shows a
// "翻译中..." state for 500ms, then flips to an "已翻译" tag and appends
// "(translated)" to the rendered text. The Set tracks which comment ids are
// translated; toggling again removes the state. Translating IDs are held in a
// separate Set so the loading spinner renders per-comment.
const translatedIds = ref(new Set())      // comment ids currently showing the translated state
const translatingIds = ref(new Set())     // comment ids mid-"翻译中..." (500ms)
// translatingList drives reactivity since Set mutations aren't tracked on their own.
const translatingList = ref([])

function isTranslating(id) { return translatingList.value.includes(id) }
function isTranslated(id) { return translatedIds.value.has(id) }

// toggleTranslate flips a comment's translation state. Turning it on fakes a
// 500ms "翻译中..." delay before marking it translated; turning it off clears it.
function toggleTranslate(c) {
  const id = c.id
  if (isTranslating(id)) return // already mid-translation; ignore re-clicks
  if (translatedIds.value.has(id)) {
    // Toggle off: remove from the translated set (and the appended marker).
    const next = new Set(translatedIds.value)
    next.delete(id)
    translatedIds.value = next
    return
  }
  // Turn on: show the loading spinner for 500ms, then mark translated.
  translatingList.value = [...translatingList.value, id]
  setTimeout(() => {
    translatingList.value = translatingList.value.filter((x) => x !== id)
    const next = new Set(translatedIds.value)
    next.add(id)
    translatedIds.value = next
  }, 500)
}

// translatedText appends the demo marker to a comment's text when translated.
function translatedText(content, id) {
  if (translatedIds.value.has(id)) {
    return content + ' (translated)'
  }
  return content
}

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}

// ===================== Feature: Video collection playlist (视频合集播放列表) =====================

// fmtDuration converts a duration in seconds (from the backend) into a
// m:ss / h:mm:ss string for the playlist rows.
function fmtDuration(s) {
  if (!s || isNaN(s) || s <= 0) return '00:00'
  const total = Math.floor(s)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const sec = total % 60
  if (h > 0) {
    return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
  }
  return `${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
}

// jumpToPlaylist sets the current index to the chosen video, closes the popup,
// and starts playback. Reusing index.value keeps swipe/progress state in sync.
function jumpToPlaylist(i) {
  if (i < 0 || i >= videos.value.length) return
  index.value = i
  showPlaylist.value = false
  nextTick(() => playCurrent())
}

// ===================== Feature: Video thumbnail timeline jump =====================
// Clicking a dot jumps to that video. Marked watched via the index watcher.
function jumpToTimeline(i) {
  if (i < 0 || i >= videos.value.length) return
  if (i === index.value) return
  index.value = i
  nextTick(() => playCurrent())
}

// Track every index that has ever been active so the timeline can mark it as
// watched. We use a Set and reassign the ref so Vue detects the change. When a
// new feed loads (index resets to 0) we also reset the set.
watch(index, (cur) => {
  if (!watchedSet.value.has(cur)) {
    const next = new Set(watchedSet.value)
    next.add(cur)
    watchedSet.value = next
  }
})
// Reset the watched set whenever the feed is reloaded (new tab / refresh).
watch(videos, () => {
  watchedSet.value = new Set([0])
})

// ===================== Feature 1: @mention (评论区at提及) =====================

// onCommentInput watches the comment field. When the last typed char is "@",
// open the suggestion popup and lazily load the suggested-follows list once.
function onCommentInput() {
  // Feature: 评论草稿自动保存 — debounce-save the draft on each input. Called
  // first so an empty input also triggers a (debounced) draft clear.
  onCommentDraftInput()
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

// ===================== Feature: Video frame capture (视频截图) =====================
// Captures the current video frame to a JPEG and triggers a download named
// "screenshot.jpg". Uses an off-DOM canvas + drawImage + toDataURL. Errors
// (e.g. tainted canvas from cross-origin video) are caught and surfaced as a
// toast rather than crashing the feed.
function captureScreenshot() {
  try {
    const vids = document.querySelectorAll('.feed-video')
    const video = vids[index.value]
    if (!video) {
      showToast('视频未就绪')
      return
    }
    // Need a frame that has actually loaded to draw.
    if (!video.videoWidth && !video.videoHeight) {
      showToast('视频未就绪')
      return
    }
    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth || video.clientWidth || 640
    canvas.height = video.videoHeight || video.clientHeight || 360
    const ctx = canvas.getContext('2d')
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
    const dataURL = canvas.toDataURL('image/jpeg', 0.9)
    const a = document.createElement('a')
    a.href = dataURL
    a.download = 'screenshot.jpg'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    showSuccessToast('截图已保存')
  } catch (e) {
    // A cross-origin video without CORS headers taints the canvas, which makes
    // toDataURL throw a SecurityError. Surface a friendly message in that case.
    showToast('截图失败')
  }
}

// ===================== Feature: Video bookmark timestamps (视频书签时间点) =====================
// Users can bookmark specific timestamps while watching a video. A 🔖 button is
// revealed by long-pressing the progress bar; tapping it saves the current
// playback time as a bookmark. Saved bookmarks render as small dots on the
// progress bar; tapping a dot seeks to that time. Bookmarks persist per-video
// in localStorage keyed by video id (max 3 per video).
const BOOKMARK_PREFIX = 'dy_bookmark_'
const bookmarks = ref([])          // [{ time, label }] for the active video
const showBookmarkBtn = ref(false) // 🔖 button visibility (long-press the bar)
let bookmarkPressTimer = null      // arms the 🔖 button on long-press
const MAX_BOOKMARKS = 3

// loadBookmarks reads the saved bookmarks for the active video from localStorage.
function loadBookmarks() {
  const v = videos.value[index.value]
  if (!v) { bookmarks.value = []; return }
  try {
    const raw = localStorage.getItem(BOOKMARK_PREFIX + v.id)
    bookmarks.value = raw ? JSON.parse(raw) : []
  } catch (e) {
    bookmarks.value = []
  }
}

// saveBookmarks persists the current bookmarks array for the active video.
function saveBookmarks() {
  const v = videos.value[index.value]
  if (!v) return
  try {
    localStorage.setItem(BOOKMARK_PREFIX + v.id, JSON.stringify(bookmarks.value))
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
}

// addBookmarkFromCurrent grabs the active video element's current time and
// stores it as a bookmark (capped at MAX_BOOKMARKS). Duplicate timestamps
// (within 1s) are ignored. Then hides the 🔖 button.
function addBookmarkFromCurrent() {
  showBookmarkBtn.value = false
  const vids = document.querySelectorAll('.feed-video')
  const video = vids[index.value]
  if (!video || !video.duration) {
    showToast('视频未就绪')
    return
  }
  const t = Math.floor(video.currentTime)
  if (bookmarks.value.some((b) => Math.abs(b.time - t) < 1)) {
    showToast('该时间点已添加')
    return
  }
  if (bookmarks.value.length >= MAX_BOOKMARKS) {
    showToast('最多添加 ' + MAX_BOOKMARKS + ' 个书签')
    return
  }
  bookmarks.value.push({ time: t, label: fmtTime(t) })
  saveBookmarks()
  showSuccessToast('已添加书签 ' + fmtTime(t))
}

// bookmarkPct returns the left% of a bookmark dot on the progress bar.
function bookmarkPct(b) {
  const vids = document.querySelectorAll('.feed-video')
  const video = vids[index.value]
  const dur = video && video.duration ? video.duration : 0
  if (dur <= 0) return 0
  return Math.min(100, Math.max(0, (b.time / dur) * 100))
}

// seekToBookmark jumps the active video to the bookmark's time.
function seekToBookmark(b) {
  const vids = document.querySelectorAll('.feed-video')
  const video = vids[index.value]
  if (!video) return
  video.currentTime = b.time
  if (video.paused) video.play().catch(() => {})
}

// armBookmarkBtn reveals the 🔖 button after a 500ms long-press on the bar.
function armBookmarkBtn() {
  cancelBookmarkPress()
  bookmarkPressTimer = setTimeout(() => {
    bookmarkPressTimer = null
    showBookmarkBtn.value = true
  }, 500)
}
function cancelBookmarkPress() {
  if (bookmarkPressTimer) {
    clearTimeout(bookmarkPressTimer)
    bookmarkPressTimer = null
  }
}
// hide the 🔖 button when the user taps elsewhere on the bar
function hideBookmarkBtn() {
  showBookmarkBtn.value = false
}

// ===================== Feature: Comment draft auto-save (评论草稿自动保存) =====================
// When the user types in the comment input, the draft is auto-saved to
// localStorage after a 1s debounce. On opening comments for a video, a saved
// draft is restored and a "📝草稿已恢复" toast is shown. A small "草稿" badge
// renders in the input while a draft exists. Sending the comment clears it.
const DRAFT_PREFIX = 'dy_comment_draft_'
const hasDraft = ref(false) // true when a draft exists for the open video
let draftDebounce = null    // 1s debounce timer for auto-save

function draftKey() {
  return DRAFT_PREFIX + currentVideoId.value
}

// onCommentDraftInput is bound to the comment input; it debounces a 1s save and
// updates the "草稿" indicator. An empty input clears any saved draft.
function onCommentDraftInput() {
  if (draftDebounce) clearTimeout(draftDebounce)
  draftDebounce = setTimeout(() => {
    const text = commentText.value
    if (text && text.trim()) {
      try { localStorage.setItem(draftKey(), text) } catch (e) {}
      hasDraft.value = true
    } else {
      clearDraft()
    }
  }, 1000)
}

// restoreDraft loads a saved draft for the open video (if any) and surfaces the
// "📝草稿已恢复" toast + indicator. Called from openComments().
function restoreDraft() {
  let saved = null
  try { saved = localStorage.getItem(draftKey()) } catch (e) {}
  if (saved) {
    commentText.value = saved
    hasDraft.value = true
    showToast('📝草稿已恢复')
  } else {
    commentText.value = ''
    hasDraft.value = false
  }
}

// clearDraft removes the saved draft for the open video and hides the indicator.
function clearDraft() {
  try { localStorage.removeItem(draftKey()) } catch (e) {}
  hasDraft.value = false
}

// ===================== Feature: Video collection curator mode (策展模式) =====================
// An editor's-pick overlay. When enabled, every video shows a "推荐" badge and a
// quality score derived deterministically from its likes/plays ratio (so the
// same video always reports the same score). High-quality videos (score > 0.3)
// get a golden border frame, and an "编辑推荐" watermark sits in the bottom-right
// of every slide. The toggle persists in localStorage 'dy_curator_mode'.
const CURATOR_KEY = 'dy_curator_mode'
const curatorMode = ref(false)

// loadCuratorPref restores the saved curator-mode preference on mount.
function loadCuratorPref() {
  try {
    curatorMode.value = localStorage.getItem(CURATOR_KEY) === '1'
  } catch (e) {
    curatorMode.value = false
  }
}

// toggleCurator flips the mode and persists the choice.
function toggleCurator() {
  curatorMode.value = !curatorMode.value
  try {
    localStorage.setItem(CURATOR_KEY, curatorMode.value ? '1' : '0')
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
}

// qualityScore returns a deterministic 0–1 score for a video from its
// likes/plays ratio. Guards divide-by-zero and clamps to [0,1].
function qualityScore(v) {
  if (!v) return 0
  const plays = v.plays || 0
  const likes = v.likes || 0
  if (plays <= 0) return likes > 0 ? 1 : 0
  return Math.min(1, Math.max(0, likes / plays))
}

// fmtScore renders the score as a percentage like "42%".
function fmtScore(v) {
  return Math.round(qualityScore(v) * 100) + '%'
}

// ===================== Feature: Comment timestamps (评论时间戳) =====================
// Show a relative time ("刚刚" / "3分钟前" / "2小时前" / "昨天" / "3天前") under
// each comment's username, derived from its created_at field. The rendered times
// refresh every minute via a reactive "now" tick, which is cleared on unmount.
const nowTick = ref(Date.now())
let commentTimeTimer = null

// relTime maps a created_at value to a Chinese relative-time string.
//   < 1 min  → 刚刚
//   < 1 h    → X分钟前
//   < 1 day  → X小时前
//   1 day    → 昨天
//   < 30 d   → X天前
//   older    → X月前 (falls back to a stable formatted date)
function relTime(createdAt) {
  if (!createdAt) return ''
  const t = new Date(createdAt)
  if (isNaN(t.getTime())) return ''
  const diff = Math.max(0, nowTick.value - t.getTime())
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return min + '分钟前'
  const hr = Math.floor(min / 60)
  if (hr < 24) return hr + '小时前'
  const day = Math.floor(hr / 24)
  if (day === 1) return '昨天'
  if (day < 30) return day + '天前'
  const mon = Math.floor(day / 30)
  return mon + '个月前'
}
</script>

<template>
  <div class="feed-page" @touchstart="onTouchStart" @touchend="onTouchEnd">
    <!-- Feature: 视频壁纸模式 — the only UI shown in wallpaper mode is a small
         "退出" button in the top-right corner; everything else is hidden. -->
    <div v-if="wallpaperMode" class="wallpaper-exit" @click="exitWallpaper">退出</div>

    <!-- Top tabs — hidden while wallpaper mode is active -->
    <div class="top-tabs" v-if="!wallpaperMode">
      <span class="tab-follow" :class="{ active: activeTab === 'follow' }" @click="activeTab = 'follow'">
        关注
        <!-- Feature 2: red dot badge shown when the following feed has new videos -->
        <i v-if="followHasNew && activeTab !== 'follow'" class="follow-dot"></i>
      </span>
      <span class="sep">|</span>
      <span :class="{ active: activeTab === 'recommend' }" @click="activeTab = 'recommend'">推荐</span>
      <!-- Feature: 操作音效反馈 — sound toggle, persisted in localStorage 'dy_sound'. -->
      <span class="sound-btn" @click="toggleSound">{{ soundOn ? '🔊' : '🔇' }}</span>
      <!-- Feature: 视频壁纸模式 — 🖼️ button enters fullscreen wallpaper/screensaver mode. -->
      <span class="wallpaper-btn" @click="enterWallpaper">🖼️</span>
      <!-- ===================== Feature: 策展模式 (curator mode) =====================
           Toggles the editor's-pick overlay; preference persists in 'dy_curator_mode'. -->
      <span class="curator-btn" :class="{ on: curatorMode }" @click="toggleCurator">🎯 策展模式</span>
      <!-- ===================== Feature: 视频睡眠定时 (sleep timer) =====================
           😴 opens a small options popup; an active timer shows a "剩余 N分钟" badge. -->
      <span class="sleep-btn" @click="sleepMenu = true">
        😴
        <span v-if="sleepActive" class="sleep-badge">剩余 {{ sleepRemaining }}分钟</span>
      </span>
      <van-icon name="search" class="search-btn" size="22" @click="router.push('/discover')" />
    </div>

    <!-- ===================== Feature: Feed mini progress dots (视频进度小圆点) =====================
         Segmented dots (one per loaded video) replace the old thin top progress
         bar. The current video's dot is enlarged and themed (#fe2c55); dots the
         user has already passed through get a subtle fill; tapping any dot jumps
         to that video. When 12 or fewer videos are loaded we overlay each dot's
         cover thumbnail so the row doubles as a visual scrubber. -->
    <div v-if="!wallpaperMode && videos.length" class="video-timeline" :class="{ many: videos.length > 12 }">
      <span
        v-for="(v, i) in videos"
        :key="v.id"
        class="vt-dot"
        :class="{ current: i === index, watched: watchedSet.has(i) }"
        @click.stop="jumpToTimeline(i)"
      >
        <img v-if="videos.length <= 12" class="vt-thumb" :src="v.cover_url" alt="" />
      </span>
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
          @touchstart.passive="armPress(i, $event)"
          @touchmove.passive="onVideoTouchMove"
          @touchend.passive="cancelPress"
          @touchcancel="cancelPress"
          @contextmenu="onVideoContextMenu(i, $event)"
        ></video>
        <!-- ===================== Feature 1: Video stats overlay (视频数据浮层) =====================
             Shown for the active slide on long-press / right-click. A tap anywhere on
             it dismisses it (the whole card is clickable). Auto-dismisses after 3s. -->
        <div
          v-if="i === index && showStats && statsIndex === i && statsVideo"
          class="stats-overlay"
          @click.stop="closeStats"
          @touchstart.stop.prevent="closeStats"
        >
          <div class="stats-card">
            <div class="stats-card-title">视频数据</div>
            <div class="stats-grid">
              <div class="stats-cell"><span class="stats-val">{{ fmtCount(statsVideo.plays) }}</span><span class="stats-lbl">播放</span></div>
              <div class="stats-cell"><span class="stats-val">{{ fmtCount(statsVideo.likes) }}</span><span class="stats-lbl">点赞</span></div>
              <div class="stats-cell"><span class="stats-val">{{ fmtCount(statsVideo.comments_count) }}</span><span class="stats-lbl">评论</span></div>
              <div class="stats-cell"><span class="stats-val">{{ fmtCount(statsVideo.shares) }}</span><span class="stats-lbl">分享</span></div>
            </div>
            <div class="stats-row">
              <span class="stats-row-lbl">🎵 音乐</span>
              <span class="stats-row-val van-ellipsis">{{ statsVideo.music || '原声' }}</span>
            </div>
            <div v-if="(statsVideo.tags || '').split(',').filter(Boolean).length" class="stats-row">
              <span class="stats-row-lbl"># 标签</span>
              <span class="stats-tags">
                <span v-for="t in statsVideo.tags.split(',').filter(Boolean)" :key="t" class="stats-tag">#{{ t }}</span>
              </span>
            </div>
            <div class="stats-hint">轻点关闭</div>
          </div>
        </div>
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
        <div v-if="i === index && !wallpaperMode" class="quality-toggle" @click.stop="toggleQuality">
          {{ quality === 'hd' ? '高清' : '标清' }}
        </div>
        <!-- HD badge shown only while 高清 is selected -->
        <div v-if="i === index && quality === 'hd' && !wallpaperMode" class="hd-badge">HD</div>
        <!-- Feature 4: Playback speed toggle (视频慢放/倍速) — cycles 0.5/1.0/1.5/2.0 -->
        <div v-if="i === index && !wallpaperMode" class="speed-toggle" @click.stop="cycleSpeed">
          {{ playbackRate }}x
        </div>
        <!-- Floating speed badge — only shown when not at normal speed -->
        <div v-if="i === index && playbackRate !== 1.0 && !wallpaperMode" class="speed-badge">
          {{ playbackRate }}x
        </div>
        <!-- ===================== Feature: 视频循环播放模式 (loop mode) =====================
             🔁 toggle next to the speed/quality buttons. When ON the current video
             loops continuously; a "单曲循环" badge confirms the active state. -->
        <div
          v-if="i === index && !wallpaperMode"
          class="loop-toggle"
          :class="{ on: loopMode }"
          @click.stop="toggleLoop"
        >🔁</div>
        <div v-if="i === index && loopMode && !wallpaperMode" class="loop-badge">单曲循环</div>
        <!-- ===================== Feature: 策展模式 (curator mode) per-slide overlay =====================
             Shown on the active slide while curator mode is on: a "推荐" badge with the
             quality score in the top-left, a golden border frame for high-quality
             videos (score > 0.3), and an "编辑推荐" watermark in the bottom-right. -->
        <template v-if="curatorMode && !wallpaperMode">
          <!-- Top-left 推荐 badge + deterministic quality score -->
          <div class="curator-badge">
            <span class="cb-tag">推荐</span>
            <span class="cb-score">质量 {{ fmtScore(v) }}</span>
          </div>
          <!-- Golden border frame for high-quality videos -->
          <div v-if="qualityScore(v) > 0.3" class="curator-frame"></div>
          <!-- Bottom-right editor's-pick watermark -->
          <div class="curator-watermark">✦ 编辑推荐</div>
        </template>
        <!-- ===================== Feature: Auto-pause on scroll away (滑出自动暂停) =====================
             Shown only on the active slide after the tab was hidden then returned
             to. The video was auto-paused on hide; tapping this overlay resumes it. -->
        <div
          v-if="i === index && pausedByVisibility"
          class="paused-overlay"
          @click.stop="resumeAfterVisibility"
          @touchstart.stop.prevent="resumeAfterVisibility"
        >
          <div class="paused-play">▶</div>
          <div class="paused-text">已为你暂停，点击继续</div>
        </div>
        <!-- Right action rail -->
        <div v-if="!wallpaperMode" class="action-rail">
          <div class="avatar-wrap" @click="router.push('/user/' + v.author_id)">
            <!-- ===================== Feature 2: Mood ring (心情光环) =====================
                 A rotating conic-gradient ring around the avatar. The gradient lives on a
                 ::before pseudo-element of .mood-ring so only the ring spins, not the
                 photo. The palette is chosen by moodClass(v) from the like ratio. -->
            <div class="mood-ring" :class="moodClass(v)">
              <img class="avatar" :src="v.author_avatar || 'https://via.placeholder.com/48'" />
            </div>
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
          <!-- ===================== Feature: Video frame capture (视频截图) =====================
               Captures the current frame to a JPEG download. -->
          <div class="action-item" @click="captureScreenshot">
            <span class="shot-icon">📸</span>
            <span>截图</span>
          </div>
          <div class="disc"><van-icon name="music-o" /></div>
        </div>
        <!-- Bottom info -->
        <!-- Video progress bar (only for the active slide) -->
        <div
          v-if="i === index && !wallpaperMode"
          class="progress-bar"
          @touchstart.passive="armBookmarkBtn"
          @touchmove.passive="cancelBookmarkPress"
          @touchend.passive="cancelBookmarkPress"
          @mousedown="armBookmarkBtn"
          @mouseup="cancelBookmarkPress"
          @mouseleave="cancelBookmarkPress"
        >
          <div class="pb-track" @click="hideBookmarkBtn">
            <div class="pb-fill" :style="{ width: progress + '%' }"></div>
            <!-- ===================== Feature: 视频书签时间点 (bookmark dots) =====================
                 Saved bookmarks render as small dots on the track; tapping a dot seeks. -->
            <span
              v-for="(b, bi) in bookmarks"
              :key="bi"
              class="pb-bookmark-dot"
              :style="{ left: bookmarkPct(b) + '%' }"
              @click.stop="seekToBookmark(b)"
            ></span>
          </div>
          <span class="pb-time">{{ currentTime }} / {{ duration }}</span>
          <!-- 🔖 bookmark button — revealed by long-pressing the progress bar -->
          <span
            v-if="showBookmarkBtn"
            class="pb-bookmark-btn"
            @click.stop="addBookmarkFromCurrent"
            @touchstart.stop.prevent="addBookmarkFromCurrent"
          >🔖</span>
        </div>
        <div v-if="!wallpaperMode" class="bottom-info">
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
            <!-- Feature: 合集 button — opens the sequential playlist popup -->
            <span class="tag playlist-tag" @click.stop="showPlaylist = true">📋 合集 {{ videos.length }}</span>
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

    <!-- ===================== Feature: Feed vertical scroll hint (上滑提示) =====================
         A lightweight floating "👆上滑" chip shown once to first-time users with a
         bounce animation. Dismissed on the first actual swipe; tracked via
         localStorage 'dy_scroll_hint'. Lower z-index than the guide overlay so the
         full guide takes precedence if both are present. -->
    <div v-if="showScrollHint" class="scroll-hint" @click.stop="dismissScrollHint">
      <span class="sh-icon">👆</span>
      <span class="sh-text">上滑</span>
    </div>

    <!-- Comment popup -->
    <van-popup v-model:show="showComment" position="bottom" round :style="{ height: '50%' }">
      <div class="comment-panel">
        <div class="cp-head">{{ commentList.length }} 条评论</div>
        <!-- ===================== Feature: Comment sort options (评论排序) =====================
             Three small tabs that re-sort the (non-pinned) comment list. Default
             keeps server order; 最热 sorts by likes desc; 最新 by id desc. -->
        <div class="cp-sort">
          <span
            v-for="opt in [{ k: 'default', t: '默认' }, { k: 'hot', t: '最热' }, { k: 'new', t: '最新' }]"
            :key="opt.k"
            class="cp-sort-tab"
            :class="{ active: commentSort === opt.k }"
            @click="commentSort = opt.k"
          >{{ opt.t }}</span>
        </div>
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
              <!-- Feature: 评论时间戳 — small gray relative time below the username -->
              <div v-if="relTime(pinnedComment.created_at)" class="cp-time">{{ relTime(pinnedComment.created_at) }}</div>
              <div class="cp-content">
                <span v-if="!isTranslating(pinnedComment.id)">
                  <template v-for="(seg, si) in parseMentions(translatedText(pinnedComment.content, pinnedComment.id))" :key="si">
                    <span v-if="seg.type === 'mention'" class="cp-mention">{{ seg.value }}</span>
                    <span v-else>{{ seg.value }}</span>
                  </template>
                </span>
                <span v-if="isTranslating(pinnedComment.id)" class="cp-translating">翻译中...</span>
                <span v-if="isTranslated(pinnedComment.id)" class="cp-translated-tag">已翻译</span>
              </div>
              <!-- Feature: 评论翻译指示 — 🌐翻译 toggle (demo translation) -->
              <div class="cp-translate-row">
                <span class="cp-translate-btn" :class="{ active: isTranslated(pinnedComment.id) }" @click="toggleTranslate(pinnedComment)">🌐{{ isTranslated(pinnedComment.id) ? '取消翻译' : '翻译' }}</span>
              </div>
              <div class="cp-reply-btn" :class="{ active: replyTo && replyTo.id === pinnedComment.id }" @click="startReply(pinnedComment)">回复</div>
              <!-- ===================== Feature: 作者置顶评论 (comment pin by creator) =====================
                   If the current user is the video author and this pinned comment is
                   their manual pin, show a "取消置顶" action to remove it. -->
              <div v-if="canPinAsAuthor() && isManualPin(pinnedComment)" class="cp-pin-btn active" @click="unpinComment">📌 取消置顶</div>
              <div v-if="replyTo && replyTo.id === pinnedComment.id" class="cp-sub-input">
                <van-field
                  v-model="replyText"
                  :placeholder="'回复 @' + pinnedComment.username"
                  class="cp-field"
                  @keyup.enter="sendReply"
                />
                <van-button size="mini" type="primary" color="#fe2c55" @click="sendReply">发送</van-button>
              </div>
              <!-- Emoji reactions (评论表情回应): a row of 5 emojis, toggle on tap,
                   counts shown when > 0, the user's pick is highlighted. -->
              <div class="cp-reactions">
                <span
                  v-for="emoji in REACTION_EMOJIS"
                  :key="emoji"
                  class="cp-rx-btn"
                  :class="{ active: userReactions[pinnedComment.id] === emoji }"
                  @click="toggleReaction(pinnedComment, emoji)"
                >
                  <span class="cp-rx-emoji">{{ emoji }}</span>
                  <span v-if="commentReactions[pinnedComment.id] && commentReactions[pinnedComment.id][emoji]" class="cp-rx-count">{{ commentReactions[pinnedComment.id][emoji] }}</span>
                </span>
              </div>
              <!-- ===================== Feature: Comment poll voting (评论投票) =====================
                   Upvote / downvote buttons, separate from emoji reactions. Only one
                   vote per comment at a time; the score = upvotes - downvotes. -->
              <div class="cp-vote">
                <span class="cp-vote-btn up" :class="{ active: userVote[pinnedComment.id] === 'up' }" @click="voteUp(pinnedComment)">👍</span>
                <span class="cp-vote-score">{{ voteScore(pinnedComment) }}</span>
                <span class="cp-vote-btn down" :class="{ active: userVote[pinnedComment.id] === 'down' }" @click="voteDown(pinnedComment)">👎</span>
              </div>
            </div>
            <div class="cp-like" :class="{ active: pinnedComment.liked }" @click="doCommentLike(pinnedComment)">
              <van-icon :name="pinnedComment.liked ? 'like' : 'like-o'" size="16" :color="pinnedComment.liked ? '#fe2c55' : '#999'" /><span>{{ pinnedComment.likes }}</span>
            </div>
          </div>
          <div
            v-for="c in sortedComments"
            :key="c.id"
            class="cp-item"
            :class="{ 'cp-item-child': c.parent_id && c.parent_id !== 0 }"
          >
            <img class="cp-avatar" :src="c.avatar || 'https://via.placeholder.com/36'" />
            <div class="cp-body">
              <div class="cp-user">{{ c.username }}</div>
              <!-- Feature: 评论时间戳 — small gray relative time below the username -->
              <div v-if="relTime(c.created_at)" class="cp-time">{{ relTime(c.created_at) }}</div>
              <div class="cp-content">
                <span v-if="!isTranslating(c.id)">
                  <template v-for="(seg, si) in parseMentions(translatedText(c.content, c.id))" :key="si">
                    <span v-if="seg.type === 'mention'" class="cp-mention">{{ seg.value }}</span>
                    <span v-else>{{ seg.value }}</span>
                  </template>
                </span>
                <span v-if="isTranslating(c.id)" class="cp-translating">翻译中...</span>
                <span v-if="isTranslated(c.id)" class="cp-translated-tag">已翻译</span>
              </div>
              <!-- Feature: 评论翻译指示 — 🌐翻译 toggle (demo translation) -->
              <div class="cp-translate-row">
                <span class="cp-translate-btn" :class="{ active: isTranslated(c.id) }" @click="toggleTranslate(c)">🌐{{ isTranslated(c.id) ? '取消翻译' : '翻译' }}</span>
              </div>
              <div class="cp-reply-btn" :class="{ active: replyTo && replyTo.id === c.id }" @click="startReply(c)">回复</div>
              <!-- ===================== Feature: 作者置顶评论 (comment pin by creator) =====================
                   When the current user is the video author, show a "📌置顶" button on
                   their own top-level comments so they can pin one per video. The
                   button is hidden on child comments (parent_id != 0). -->
              <div
                v-if="canPinAsAuthor() && isOwnComment(c) && !(c.parent_id && c.parent_id !== 0)"
                class="cp-pin-btn"
                @click="pinComment(c)"
              >📌置顶</div>
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
              <!-- Emoji reactions (评论表情回应): a row of 5 emojis, toggle on tap,
                   counts shown when > 0, the user's pick is highlighted. -->
              <div class="cp-reactions">
                <span
                  v-for="emoji in REACTION_EMOJIS"
                  :key="emoji"
                  class="cp-rx-btn"
                  :class="{ active: userReactions[c.id] === emoji }"
                  @click="toggleReaction(c, emoji)"
                >
                  <span class="cp-rx-emoji">{{ emoji }}</span>
                  <span v-if="commentReactions[c.id] && commentReactions[c.id][emoji]" class="cp-rx-count">{{ commentReactions[c.id][emoji] }}</span>
                </span>
              </div>
              <!-- ===================== Feature: Comment poll voting (评论投票) =====================
                   Upvote / downvote buttons, separate from emoji reactions. Only one
                   vote per comment at a time; the score = upvotes - downvotes. -->
              <div class="cp-vote">
                <span class="cp-vote-btn up" :class="{ active: userVote[c.id] === 'up' }" @click="voteUp(c)">👍</span>
                <span class="cp-vote-score" :class="{ neg: voteScore(c) < 0 }">{{ voteScore(c) }}</span>
                <span class="cp-vote-btn down" :class="{ active: userVote[c.id] === 'down' }" @click="voteDown(c)">👎</span>
              </div>
            </div>
            <div class="cp-like" :class="{ active: c.liked }" @click="doCommentLike(c)">
              <van-icon :name="c.liked ? 'like' : 'like-o'" size="16" :color="c.liked ? '#fe2c55' : '#999'" /><span>{{ c.likes }}</span>
            </div>
          </div>
          <div v-if="!commentList.length" class="cp-empty">暂无评论，来说点什么吧</div>
        </div>
        <!-- ===================== Feature: Comment quick emoji bar (快捷表情栏) =====================
             A horizontal row of 8 quick emojis above the input. Tapping one appends
             it to the comment text. "😊更多" expands a second row of 16 more emojis. -->
        <div class="cp-emoji-bar">
          <div class="cp-emoji-row">
            <span
              v-for="(e, ei) in QUICK_EMOJIS"
              :key="'q' + ei"
              class="cp-emoji-btn"
              @click="appendEmoji(e)"
            >{{ e }}</span>
            <span class="cp-emoji-more" @click="moreExpanded = !moreExpanded">😊更多</span>
          </div>
          <div v-if="moreExpanded" class="cp-emoji-row cp-emoji-more-row">
            <span
              v-for="(e, ei) in MORE_EMOJIS"
              :key="'m' + ei"
              class="cp-emoji-btn"
              @click="appendEmoji(e)"
            >{{ e }}</span>
          </div>
        </div>
        <div class="cp-input">
          <div class="cp-input-field-wrap">
            <!-- Feature: 评论草稿自动保存 — "草稿" badge shown while a saved draft exists -->
            <span v-if="hasDraft" class="cp-draft-badge">草稿</span>
            <!-- ===================== Feature: 评论字数限制 (character limit) =====================
                 Live counter on the right of the field. Turns red with "超出字数限制"
                 once the 200-char cap is exceeded. -->
            <span
              v-if="commentText.length > 0"
              class="cp-char-counter"
              :class="{ over: commentOverLimit }"
            >{{ commentOverLimit ? '超出字数限制 ' : '' }}{{ commentText.length }}/{{ COMMENT_MAX }}</span>
            <van-field
              v-model="commentText"
              placeholder="说点什么，用 @ 提及好友"
              class="cp-field"
              @input="onCommentInput"
              @keyup.enter="sendComment"
              @blur="() => setTimeout(closeMention, 150)"
            />
          </div>
          <van-button
            size="small"
            type="primary"
            color="#fe2c55"
            :disabled="commentOverLimit"
            @click="sendComment"
          >发送</van-button>
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

    <!-- ===================== Feature: Video collection playlist (视频合集播放列表) =====================
         Bottom-sheet popup listing every loaded video as a sequential playlist.
         Each row shows cover thumbnail + title + duration; the currently-playing
         video is highlighted and tapping a row jumps to it. The 顺序播放 toggle
         auto-advances to the next video when the current one ends. -->
    <van-popup v-model:show="showPlaylist" position="bottom" round :style="{ height: '55%' }">
      <div class="playlist-panel">
        <div class="pl-head">
          <span class="pl-title">📋 合集播放列表</span>
          <span class="pl-count">{{ videos.length }} 个视频</span>
        </div>
        <div class="pl-seq" @click="sequentialPlay = !sequentialPlay">
          <span class="pl-seq-switch" :class="{ on: sequentialPlay }">
            <span class="pl-seq-knob"></span>
          </span>
          <span class="pl-seq-label" :class="{ on: sequentialPlay }">顺序播放</span>
          <span class="pl-seq-hint">{{ sequentialPlay ? '播完自动播放下一个' : '已关闭' }}</span>
        </div>
        <div class="pl-list">
          <div
            v-for="(pv, pi) in videos"
            :key="pv.id"
            class="pl-item"
            :class="{ current: pi === index }"
            @click="jumpToPlaylist(pi)"
          >
            <div class="pl-cover-wrap">
              <img class="pl-cover" :src="pv.cover_url || 'https://via.placeholder.com/80x110'" />
              <span class="pl-duration">{{ fmtDuration(pv.duration) }}</span>
              <span v-if="pi === index" class="pl-playing">▶ 播放中</span>
            </div>
            <div class="pl-info">
              <div class="pl-name van-ellipsis">{{ pv.title }}</div>
              <div class="pl-author">@{{ pv.author_name }}</div>
              <div class="pl-meta">
                <span>❤ {{ fmtCount(pv.likes) }}</span>
                <span>💬 {{ fmtCount(pv.comments_count) }}</span>
              </div>
            </div>
            <div v-if="pi === index" class="pl-now-bar"></div>
          </div>
          <div v-if="!videos.length" class="pl-empty">暂无视频</div>
        </div>
      </div>
    </van-popup>

    <!-- ===================== Feature: Video sleep timer (视频睡眠定时) =====================
         A centered popup with 15分钟 / 30分钟 / 60分钟 / 关闭 options. Selecting a
         duration arms a timer that auto-pauses all videos when it fires; 关闭
         cancels a running timer. The active countdown shows as a badge on 😴. -->
    <van-popup v-model:show="sleepMenu" round :style="{ width: '78%', maxWidth: '340px', padding: '0' }">
      <div class="sleep-panel">
        <div class="sleep-head">😴 睡眠定时</div>
        <div class="sleep-sub">选定时间后自动暂停播放</div>
        <div class="sleep-opts">
          <div
            v-for="m in SLEEP_OPTIONS"
            :key="m"
            class="sleep-opt"
            :class="{ active: sleepActive && sleepRemaining > 0 }"
            @click="startSleep(m)"
          >{{ m }}分钟</div>
          <div class="sleep-opt sleep-opt-off" @click="cancelSleep">关闭</div>
        </div>
        <div v-if="sleepActive" class="sleep-status">定时中 · 剩余 {{ sleepRemaining }} 分钟</div>
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

/* ===================== Feature: Video thumbnail timeline (视频缩略图时间轴) =====================
   A thin horizontal strip fixed at the very top, just below the tabs. Each
   loaded video renders as a dot; the current one is enlarged + themed, watched
   videos get a subtle fill, and each dot is clickable to jump to that video.
   When the stack has 12 or fewer videos we overlay the cover thumbnail on each
   dot so it doubles as a visual scrubber; beyond that we shrink to plain dots
   so they all fit on screen. */
.video-timeline {
  position: fixed;
  top: 48px;
  left: 0;
  right: 0;
  z-index: 19;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 6px 12px;
  pointer-events: auto;
  overflow-x: auto;
  scrollbar-width: none;
}
.video-timeline::-webkit-scrollbar { display: none; }
.video-timeline.many { justify-content: flex-start; }
.vt-dot {
  position: relative;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.35);
  cursor: pointer;
  flex: 0 0 auto;
  transition: transform 0.18s ease, background 0.18s ease, border-color 0.18s ease;
  overflow: hidden;
}
/* Watched dots get a subtle theme fill. */
.vt-dot.watched { background: rgba(254, 44, 85, 0.45); border-color: rgba(254, 44, 85, 0.6); }
/* The current dot is enlarged and fully themed. */
.vt-dot.current {
  transform: scale(1.55);
  background: #fe2c55;
  border-color: #fff;
  box-shadow: 0 0 8px rgba(254, 44, 85, 0.8);
}
.vt-dot:active { transform: scale(1.35); }
.vt-dot.current:active { transform: scale(1.45); }
/* Cover thumbnail shown when the stack is small enough to be legible. */
.vt-thumb {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.85;
}
.vt-dot.current .vt-thumb { opacity: 1; }
/* Feature: 操作音效反馈 — sound toggle sits just left of the search icon. */
.sound-btn {
  position: absolute;
  right: 52px;
  font-size: 20px;
  cursor: pointer;
  user-select: none;
  line-height: 1;
}
/* Feature: 视频壁纸模式 — 🖼️ button sits just left of the sound toggle. */
.wallpaper-btn {
  position: absolute;
  right: 88px;
  font-size: 20px;
  cursor: pointer;
  user-select: none;
  line-height: 1;
}
.wallpaper-btn:active { opacity: 0.7; }
/* ===================== Feature: 视频睡眠定时 (sleep timer) =====================
   😴 button sits just left of the 🖼️ wallpaper button; carries a small badge
   with the remaining minutes while a timer is active. */
.sleep-btn {
  position: absolute;
  right: 124px;
  font-size: 20px;
  cursor: pointer;
  user-select: none;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.sleep-btn:active { opacity: 0.7; }
.sleep-badge {
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  background: #fe2c55;
  padding: 1px 6px;
  border-radius: 8px;
  white-space: nowrap;
}
/* Centered options popup */
.sleep-panel { padding: 20px 18px 22px; background: #161616; }
.sleep-head { text-align: center; color: #fff; font-size: 17px; font-weight: bold; }
.sleep-sub { text-align: center; color: #888; font-size: 12px; margin-top: 4px; }
.sleep-opts { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; margin-top: 16px; }
.sleep-opt {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 46px;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  background: #222;
  border: 1px solid #2a2a2a;
  border-radius: 12px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, border-color 0.15s;
}
.sleep-opt:active { background: #2c2c2c; }
.sleep-opt.active { border-color: #fe2c55; }
.sleep-opt-off { grid-column: 1 / -1; color: #fe2c55; }
.sleep-status {
  text-align: center;
  color: #25f4ee;
  font-size: 12px;
  margin-top: 14px;
}
/* The small "退出" button shown only in wallpaper mode (top-right corner). */
.wallpaper-exit {
  position: fixed;
  top: 14px;
  right: 16px;
  z-index: 50;
  padding: 6px 14px;
  font-size: 13px;
  color: #fff;
  background: rgba(0, 0, 0, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 16px;
  backdrop-filter: blur(6px);
  cursor: pointer;
  user-select: none;
}
.wallpaper-exit:active { background: rgba(254, 44, 85, 0.7); }
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
/* Feature: 视频截图 — emoji glyph sized to match the 32px van-icons in the rail. */
.action-item .shot-icon { font-size: 30px; line-height: 1; }
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
/* Feature: 评论排序 — row of three small sort tabs under the popup header. */
.cp-sort { display: flex; gap: 18px; padding: 8px 16px; }
.cp-sort-tab {
  color: #888;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
}
.cp-sort-tab.active { color: #fe2c55; font-weight: 600; }
.cp-list { flex: 1; overflow-y: auto; padding: 8px 12px; }
.cp-item { display: flex; gap: 10px; padding: 12px 0; }
/* Child comments (parent_id != 0) are indented to reflect the reply nesting. */
.cp-item-child { margin-left: 40px; }
.cp-item-child .cp-avatar { width: 28px; height: 28px; }
.cp-avatar { width: 36px; height: 36px; border-radius: 50%; flex-shrink: 0; }
.cp-body { flex: 1; }
.cp-user { color: #888; font-size: 13px; }
/* Feature: 评论时间戳 — small gray relative time below the username */
.cp-time { color: #666; font-size: 11px; margin-top: 2px; }
.cp-content { color: #fff; font-size: 14px; margin-top: 3px; }
.cp-reply-btn { color: #888; font-size: 12px; margin-top: 6px; display: inline-block; cursor: pointer; }
.cp-reply-btn.active { color: #fe2c55; }
/* ===================== Feature: 作者置顶评论 (comment pin by creator) =====================
   A small "📌置顶" / "📌 取消置顶" action shown next to the reply button. The
   inactive pin button uses a muted style; the active (cancel) state is themed. */
.cp-pin-btn {
  display: inline-block;
  margin-top: 6px;
  margin-left: 12px;
  font-size: 12px;
  color: #fe2c55;
  cursor: pointer;
  user-select: none;
  padding: 2px 8px;
  border: 1px solid rgba(254, 44, 85, 0.4);
  border-radius: 10px;
  background: rgba(254, 44, 85, 0.08);
  transition: background 0.15s, border-color 0.15s;
}
.cp-pin-btn:active { background: rgba(254, 44, 85, 0.25); }
.cp-pin-btn.active { background: rgba(254, 44, 85, 0.2); border-color: #fe2c55; font-weight: 600; }
/* ===================== Feature: Comment translation indicator (评论翻译指示) =====================
   🌐翻译 button sits below the comment text; an active translation shows an
   "已翻译" tag and a "(translated)" marker appended to the text. While the
   500ms fake translation runs, a "翻译中..." placeholder is shown. */
.cp-translate-row { margin-top: 6px; }
.cp-translate-btn {
  display: inline-block;
  color: #25f4ee;
  font-size: 12px;
  padding: 2px 8px;
  background: rgba(37, 244, 238, 0.1);
  border: 1px solid rgba(37, 244, 238, 0.35);
  border-radius: 10px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, border-color 0.15s;
}
.cp-translate-btn:active { opacity: 0.7; }
.cp-translate-btn.active { color: #fff; background: rgba(37, 244, 238, 0.25); border-color: #25f4ee; }
.cp-translated-tag {
  display: inline-block;
  margin-left: 6px;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
  background: #25f4ee;
  padding: 1px 6px;
  border-radius: 4px;
  vertical-align: middle;
}
.cp-translating {
  color: #25f4ee;
  font-size: 13px;
}
.cp-sub-input { display: flex; gap: 8px; align-items: center; margin-top: 8px; }
.cp-like { display: flex; flex-direction: column; align-items: center; color: #888; font-size: 11px; cursor: pointer; }
.cp-like.active { color: #fe2c55; }
.cp-empty { text-align: center; color: #666; padding: 40px; }
.cp-input { display: flex; gap: 8px; padding: 10px; border-top: 1px solid #222; }
.cp-field { background: #222; border-radius: 18px; }

/* ===================== Feature: Comment quick emoji bar (快捷表情栏) ===================== */
/* Horizontal strip of quick emojis shown above the input. */
.cp-emoji-bar { border-top: 1px solid #222; background: #121212; }
.cp-emoji-row { display: flex; align-items: center; gap: 4px; padding: 8px 10px; overflow-x: auto; scrollbar-width: none; }
.cp-emoji-row::-webkit-scrollbar { display: none; }
.cp-emoji-btn {
  flex-shrink: 0;
  font-size: 22px;
  line-height: 1;
  padding: 4px 6px;
  border-radius: 8px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, transform 0.1s;
}
.cp-emoji-btn:active { background: #2a2a2a; transform: scale(1.2); }
.cp-emoji-more {
  flex-shrink: 0;
  font-size: 12px;
  color: #fe2c55;
  padding: 6px 10px;
  border-radius: 14px;
  background: rgba(254,44,85,0.12);
  cursor: pointer;
  white-space: nowrap;
  margin-left: 2px;
}
.cp-emoji-more:active { background: rgba(254,44,85,0.3); }
.cp-emoji-more-row { border-top: 1px solid #1c1c1c; }

/* ===================== Feature: Comment emoji reactions (评论表情回应) ===================== */
/* A horizontal row of emoji buttons below each comment. The active emoji is
   highlighted with the theme color; counts appear when greater than 0. */
.cp-reactions { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
.cp-rx-btn {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px 8px;
  background: #222;
  border: 1px solid #2a2a2a;
  border-radius: 14px;
  font-size: 14px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, border-color 0.15s;
}
.cp-rx-btn:active { background: #2c2c2c; }
.cp-rx-btn.active { background: rgba(254,44,85,0.18); border-color: #fe2c55; }
.cp-rx-emoji { line-height: 1; }
.cp-rx-count { color: #fe2c55; font-size: 11px; font-weight: bold; }

/* ===================== Feature: Comment poll voting (评论投票) =====================
   A compact upvote / score / downvote row beneath the emoji reactions. The
   active vote button takes the theme color; a negative score is shown in red. */
.cp-vote { display: inline-flex; align-items: center; gap: 8px; margin-top: 8px; padding: 4px 4px; }
.cp-vote-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #222;
  border: 1px solid #2a2a2a;
  font-size: 15px;
  cursor: pointer;
  user-select: none;
  line-height: 1;
  transition: background 0.15s, border-color 0.15s, transform 0.15s;
}
.cp-vote-btn:active { transform: scale(0.92); }
.cp-vote-btn.up.active { background: rgba(254, 44, 85, 0.2); border-color: #fe2c55; }
.cp-vote-btn.down.active { background: rgba(80, 160, 255, 0.2); border-color: #50a0ff; }
.cp-vote-score {
  min-width: 22px;
  text-align: center;
  color: #fff;
  font-size: 13px;
  font-weight: bold;
  padding: 0 2px;
}
.cp-vote-score.neg { color: #ff6b6b; }

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

/* ===================== Feature: 视频循环播放模式 (loop mode) styles =====================
   🔁 toggle button, positioned just below the speed toggle. Turns themed-red
   while active. The "单曲循环" badge floats beside it as confirmation. */
.loop-toggle {
  position: absolute;
  top: 116px;
  right: 14px;
  z-index: 12;
  padding: 4px 10px;
  font-size: 14px;
  line-height: 1;
  color: #fff;
  background: rgba(0,0,0,0.45);
  border: 1px solid rgba(255,255,255,0.35);
  border-radius: 12px;
  backdrop-filter: blur(4px);
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, border-color 0.15s;
}
.loop-toggle.on {
  background: rgba(254,44,85,0.85);
  border-color: #fe2c55;
  box-shadow: 0 0 10px rgba(254,44,85,0.5);
}
.loop-toggle:active { background: rgba(254,44,85,0.7); }
.loop-badge {
  position: absolute;
  top: 116px;
  right: 52px;
  z-index: 12;
  padding: 3px 8px;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
  background: rgba(254,44,85,0.9);
  border-radius: 10px;
  white-space: nowrap;
  pointer-events: none;
  box-shadow: 0 2px 8px rgba(254,44,85,0.4);
}

/* ===================== Feature: Auto-pause on scroll away (滑出自动暂停) =====================
   Centered overlay shown on the active slide after returning from a hidden tab.
   The video was paused on hide; tapping the overlay resumes it. */
.paused-overlay {
  position: absolute;
  inset: 0;
  z-index: 14;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  background: rgba(0, 0, 0, 0.35);
}
.paused-play {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: rgba(254, 44, 85, 0.92);
  color: #fff;
  font-size: 30px;
  line-height: 72px;
  text-align: center;
  box-shadow: 0 4px 20px rgba(254, 44, 85, 0.5);
  animation: pausedPulse 1.6s ease-in-out infinite;
}
.paused-text {
  color: #fff;
  font-size: 14px;
  padding: 6px 16px;
  background: rgba(0, 0, 0, 0.55);
  border-radius: 16px;
}
@keyframes pausedPulse { 0%, 100% { transform: scale(1); opacity: 0.92; } 50% { transform: scale(1.08); opacity: 1; } }

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

/* ===================== Feature: Feed vertical scroll hint (上滑提示) =====================
   A small floating chip near the bottom-center with a bounce animation. The
   whole chip bounces upward repeatedly to suggest a swipe-up gesture; tapping it
   also dismisses it. Lower z-index than the guide overlay so the full guide takes
   precedence if both are present. */
.scroll-hint {
  position: fixed;
  left: 50%;
  bottom: 150px;
  transform: translateX(-50%);
  z-index: 80;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: rgba(254, 44, 85, 0.9);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  border-radius: 20px;
  box-shadow: 0 4px 18px rgba(254, 44, 85, 0.5);
  cursor: pointer;
  user-select: none;
  animation: scrollHintBounce 1.2s ease-in-out infinite;
}
.scroll-hint:active { opacity: 0.8; }
.sh-icon { font-size: 18px; line-height: 1; }
.sh-text { letter-spacing: 1px; }
/* The chip lifts up then settles, echoing the swipe-up gesture. */
@keyframes scrollHintBounce {
  0%   { transform: translate(-50%, 0); }
  45%  { transform: translate(-50%, -14px); }
  60%  { transform: translate(-50%, -14px); }
  100% { transform: translate(-50%, 0); }
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

/* ===================== Feature: Video collection playlist (视频合集播放列表) ===================== */
/* The 合集 chip in the tag row — slightly accent color to stand out. */
.playlist-tag {
  background: rgba(37, 244, 238, 0.75);
  cursor: pointer;
}
.playlist-tag:active { opacity: 0.7; }
/* Bottom-sheet panel */
.playlist-panel { display: flex; flex-direction: column; height: 100%; background: #161616; }
.pl-head { display: flex; align-items: center; justify-content: space-between; padding: 14px 16px 8px; }
.pl-title { color: #fff; font-size: 16px; font-weight: bold; }
.pl-count { color: #fe2c55; font-size: 12px; }
/* 顺序播放 toggle row */
.pl-seq {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px 12px;
  border-bottom: 1px solid #222;
  cursor: pointer;
}
.pl-seq-switch {
  position: relative;
  width: 40px;
  height: 22px;
  border-radius: 11px;
  background: #444;
  transition: background 0.2s;
  flex-shrink: 0;
}
.pl-seq-switch.on { background: #fe2c55; }
.pl-seq-knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.2s;
}
.pl-seq-switch.on .pl-seq-knob { transform: translateX(18px); }
.pl-seq-label { color: #fff; font-size: 14px; font-weight: 500; }
.pl-seq-label.on { color: #fe2c55; }
.pl-seq-hint { color: #888; font-size: 11px; margin-left: auto; }
/* Scrollable list */
.pl-list { flex: 1; overflow-y: auto; padding: 4px 0; }
.pl-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  cursor: pointer;
  position: relative;
}
.pl-item:active { background: #1f1f1f; }
.pl-item.current { background: rgba(254, 44, 85, 0.12); }
.pl-cover-wrap { position: relative; flex-shrink: 0; width: 64px; height: 88px; border-radius: 6px; overflow: hidden; background: #222; }
.pl-cover { width: 100%; height: 100%; object-fit: cover; }
.pl-duration {
  position: absolute;
  right: 3px;
  bottom: 3px;
  color: #fff;
  font-size: 10px;
  background: rgba(0, 0, 0, 0.65);
  padding: 1px 4px;
  border-radius: 3px;
  font-variant-numeric: tabular-nums;
}
.pl-playing {
  position: absolute;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 11px;
  background: rgba(0, 0, 0, 0.45);
}
.pl-info { flex: 1; min-width: 0; }
.pl-name { color: #fff; font-size: 14px; line-height: 18px; font-weight: 500; }
.pl-author { color: #888; font-size: 12px; margin-top: 4px; }
.pl-meta { display: flex; gap: 12px; color: #666; font-size: 11px; margin-top: 4px; }
.pl-now-bar { width: 3px; align-self: stretch; background: #fe2c55; border-radius: 2px; margin-left: 4px; }
.pl-empty { text-align: center; color: #666; padding: 40px; }

/* ===================== Feature 1: Video stats overlay (视频数据浮层) ===================== */
/* A full-slide dimmed backdrop that centers the dark card and absorbs the
   dismiss tap. The backdrop covers one slide (absolute), not the whole page. */
.stats-overlay {
  position: absolute;
  inset: 0;
  z-index: 16;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(1px);
  cursor: pointer;
  animation: statsFadeIn 0.18s ease-out;
}
@keyframes statsFadeIn { from { opacity: 0; } to { opacity: 1; } }
/* Dark rounded card holding the stats. Width is capped so it reads as a card
   rather than filling the screen. */
.stats-card {
  width: 86%;
  max-width: 320px;
  padding: 18px 18px 14px;
  background: rgba(20, 20, 20, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 18px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.55);
  color: #fff;
  animation: statsCardIn 0.2s ease-out;
}
@keyframes statsCardIn { from { transform: scale(0.92); opacity: 0; } to { transform: scale(1); opacity: 1; } }
.stats-card-title {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
  text-align: center;
  margin-bottom: 14px;
  letter-spacing: 0.5px;
}
/* 2x2 grid of headline counts (plays / likes / comments / shares). */
.stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 14px;
}
.stats-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 6px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 12px;
}
.stats-val {
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}
.stats-lbl {
  margin-top: 4px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.6);
}
/* Music + tags rows. */
.stats-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  min-width: 0;
}
.stats-row-lbl {
  flex-shrink: 0;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.55);
}
.stats-row-val {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: #fff;
}
.stats-tags {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}
.stats-tag {
  font-size: 12px;
  color: #fff;
  background: rgba(254, 44, 85, 0.7);
  padding: 2px 8px;
  border-radius: 10px;
  line-height: 16px;
}
.stats-hint {
  margin-top: 12px;
  text-align: center;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.4);
}

/* ===================== Feature 2: Mood ring (心情光环) ===================== */
/* A rotating conic-gradient ring drawn on .mood-ring::before sits behind a
   static circular avatar, so only the ring spins (Instagram-stories style).
   The .mood-ring wrapper is sized to match the original avatar (48px) and
   positioned relative so the ::before layer can fill it. The palette is chosen
   by the mood-* modifier classes. */
.mood-ring {
  position: relative;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  padding: 3px; /* gap between the rotating gradient and the photo */
  background: #000; /* inner gap color between ring and photo */
}
/* The gradient ring layer: an inset circular fill that rotates. sits behind
   the avatar (z-index 0; avatar is z-index 1). */
.mood-ring::before {
  content: '';
  position: absolute;
  inset: -3px; /* extend past the padding so the ring shows around the photo */
  border-radius: 50%;
  z-index: 0;
  background: conic-gradient(from 0deg, #555, #888, #555, #555);
}
/* The avatar sits above the gradient, clipped to a circle smaller than the
   ring so the gradient shows as a border around it. */
.mood-ring .avatar {
  position: relative;
  z-index: 1;
  width: 100%;
  height: 100%;
  border: 2px solid #000; /* dark inner edge separating photo from ring */
  box-sizing: border-box;
}
/* 热门 — warm gradient (red/orange), for like ratio > 0.3. */
.mood-ring.mood-hot::before {
  background: conic-gradient(from 0deg, #fe2c55, #ff6a00, #ffd200, #fe2c55);
  animation: moodRingSpin 3s linear infinite;
}
/* 受欢迎 — cool gradient (blue/cyan), for ratio 0.1–0.3. */
.mood-ring.mood-popular::before {
  background: conic-gradient(from 0deg, #25f4ee, #4d8bff, #6a5cff, #25f4ee);
  animation: moodRingSpin 3s linear infinite;
}
/* 普通 — subtle gray, for ratio < 0.1. Static (no spin) so it reads as muted. */
.mood-ring.mood-normal::before {
  background: conic-gradient(from 0deg, #555, #888, #555, #555);
  animation: none;
}
@keyframes moodRingSpin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* ===================== Feature: Video bookmark timestamps (视频书签时间点) ===================== */
/* The progress bar needs a higher z-index so its long-press + dots sit above
   the bottom-info overlay, and the track becomes relative so dots can position. */
.progress-bar { user-select: none; }
.progress-bar .pb-track { position: relative; }
/* Bookmark dots — small accent dots layered on the track; tappable to seek. */
.pb-bookmark-dot {
  position: absolute;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: #ffd700;
  border: 2px solid rgba(255, 255, 255, 0.9);
  box-shadow: 0 0 6px rgba(255, 215, 0, 0.7);
  cursor: pointer;
  z-index: 2;
  transition: transform 0.1s;
}
.pb-bookmark-dot:active { transform: translate(-50%, -50%) scale(1.3); }
/* The 🔖 button revealed by long-pressing the bar; sits just above the track. */
.pb-bookmark-btn {
  font-size: 20px;
  line-height: 1;
  cursor: pointer;
  user-select: none;
  animation: bookmarkPop 0.18s ease-out;
}
@keyframes bookmarkPop {
  from { opacity: 0; transform: scale(0.5); }
  to { opacity: 1; transform: scale(1); }
}

/* ===================== Feature: Comment draft auto-save (评论草稿自动保存) ===================== */
/* The input field + draft badge share a row; the badge sits at the field's right edge. */
.cp-input-field-wrap { position: relative; flex: 1; }
.cp-draft-badge {
  position: absolute;
  top: 50%;
  right: 12px;
  transform: translateY(-50%);
  z-index: 3;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  background: #fe2c55;
  padding: 1px 7px;
  border-radius: 8px;
  pointer-events: none;
}

/* ===================== Feature: Comment character limit (评论字数限制) =====================
   A small live "X/200" counter pinned to the right of the comment field. It
   turns red and shows "超出字数限制" once the 200-char cap is exceeded. */
.cp-char-counter {
  position: absolute;
  top: 50%;
  right: 12px;
  transform: translateY(-50%);
  z-index: 4;
  font-size: 10px;
  font-weight: 600;
  color: #888;
  background: rgba(0, 0, 0, 0.55);
  padding: 1px 7px;
  border-radius: 8px;
  pointer-events: none;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.cp-char-counter.over {
  color: #ff4d4f;
  background: rgba(255, 77, 79, 0.15);
  border: 1px solid rgba(255, 77, 79, 0.6);
}

/* ===================== Feature: 策展模式 (curator mode) ===================== */
/* Toggle button — sits just left of the 😴 sleep button. Lights up gold when on. */
.curator-btn {
  position: absolute;
  right: 160px;
  font-size: 13px;
  font-weight: 600;
  color: rgba(255,255,255,0.75);
  cursor: pointer;
  user-select: none;
  line-height: 1;
  padding: 4px 9px;
  border-radius: 12px;
  border: 1px solid rgba(255,255,255,0.3);
  background: rgba(0,0,0,0.35);
  white-space: nowrap;
  transition: color 0.15s, background 0.15s, border-color 0.15s;
}
.curator-btn:active { opacity: 0.7; }
.curator-btn.on {
  color: #4a3500;
  background: linear-gradient(135deg, #ffd700, #ffb300);
  border-color: #ffd700;
  box-shadow: 0 2px 10px rgba(255,215,0,0.5);
}
/* Top-left 推荐 badge + quality score, gold-toned pill. */
.curator-badge {
  position: absolute;
  top: 52px;
  left: 14px;
  z-index: 13;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(255,215,0,0.92), rgba(255,179,0,0.92));
  color: #4a3500;
  font-size: 12px;
  font-weight: 600;
  box-shadow: 0 2px 10px rgba(255,215,0,0.45);
  pointer-events: none;
  animation: curatorBadgeIn 0.25s ease-out;
}
@keyframes curatorBadgeIn {
  from { opacity: 0; transform: translateY(-8px); }
  to { opacity: 1; transform: translateY(0); }
}
.cb-tag { font-weight: 700; }
.cb-score { font-variant-numeric: tabular-nums; opacity: 0.9; }
/* Golden border frame overlay for high-quality videos (score > 0.3). */
.curator-frame {
  position: absolute;
  inset: 0;
  z-index: 12;
  border: 4px solid rgba(255,215,0,0.85);
  box-shadow: inset 0 0 24px rgba(255,215,0,0.35);
  pointer-events: none;
  animation: curatorFramePulse 2.4s ease-in-out infinite;
}
@keyframes curatorFramePulse {
  0%, 100% { box-shadow: inset 0 0 24px rgba(255,215,0,0.35); }
  50% { box-shadow: inset 0 0 36px rgba(255,215,0,0.6); }
}
/* Bottom-right editor's-pick watermark. */
.curator-watermark {
  position: absolute;
  right: 14px;
  bottom: 188px;
  z-index: 13;
  color: rgba(255,215,0,0.85);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 1px;
  text-shadow: 0 1px 4px rgba(0,0,0,0.6);
  pointer-events: none;
}
</style>