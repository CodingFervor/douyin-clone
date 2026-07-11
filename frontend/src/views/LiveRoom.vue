<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getLiveRoom, likeLive, sendLiveMessage, getLiveGifts, startPK, getActivePK, scorePK, guardHost, getGuardStatus, dropRedPacket, getActiveRedPacket, grabRedPacket, getContributors, contribute, banUser, getSuggestFollows, getUser, toggleFollow, getLiveSchedules } from '../api'

const route = useRoute()
const router = useRouter()
const room = ref(null)
const products = ref([])
const loading = ref(true)
const likeCount = ref(0)
const showCart = ref(false)
const floatingHearts = ref([])
// Live chat / danmaku
const messages = ref([])
const msgText = ref('')
const msgListRef = ref(null)
let pollTimer = null
// Gifts
const gifts = ref([])
const showGifts = ref(false)
const flyingGifts = ref([])
// ===================== Feature: Full-screen mega gift effect (全屏礼物特效) =====================
// High-value gifts (price >= 100, e.g. 跑车/火箭) trigger a full-screen
// animation: a golden flash, a large emoji flying diagonally across the screen,
// and an announcement banner at the top center with a gold gradient. Each entry
// is auto-removed after the ~3s animation completes.
const giftEffects = ref([]) // active mega effects [{id, icon, name, username, is_mega}]
let giftEffectSeq = 0
// PK battle
const pk = ref(null)
// Fan guard
const guardCount = ref(0)
const isGuarding = ref(false)
// Red packets (红包雨)
const redPacket = ref(null)
const fallingPackets = ref([])
// Contribution board (贡献榜)
const contributors = ref([])
const showContributors = ref(false)
const showDanmaku = ref(true)
// ---- Highlight moments (直播高光时刻) ----
// Demo data: notable moments in this live session, shown in a timeline popup.
const highlights = ref([
  { time: '5分钟前', text: '直播间人数突破1万！' },
  { time: '12分钟前', text: '收到火箭礼物🚀' },
  { time: '20分钟前', text: 'PK大获全胜！' },
])
const showHighlights = ref(false)

// ===================== Feature: Live room theme (直播间主题装扮) =====================
// A color picker that re-tints the room's background and accent UI. Each theme
// defines a background color and a complementary accent; the selection persists
// in localStorage 'dy_live_theme' and is restored on mount.
const THEMES = [
  { key: 'default', label: '默认', bg: '#000000', accent: '#fe2c55' },
  { key: 'warm', label: '暖橙', bg: '#1a0a00', accent: '#ff9500' },
  { key: 'blue', label: '深蓝', bg: '#001a1a', accent: '#25f4ee' },
  { key: 'pink', label: '樱粉', bg: '#1a0010', accent: '#ff5c8a' },
  { key: 'green', label: '森绿', bg: '#001a00', accent: '#34c759' },
  { key: 'purple', label: '暗紫', bg: '#0a001a', accent: '#9c27b0' },
]
const THEME_KEY = 'dy_live_theme'
const themeBg = ref('#000000')
const themeAccent = ref('#fe2c55')
const showTheme = ref(false)
// Apply a theme object: set reactive colors + persist the key.
function applyTheme(t) {
  themeBg.value = t.bg
  themeAccent.value = t.accent
  try { localStorage.setItem(THEME_KEY, t.key) } catch (e) {}
  showTheme.value = false
}
// Restore the saved theme from localStorage; defaults to 默认.
function restoreTheme() {
  try {
    const key = localStorage.getItem(THEME_KEY)
    if (key) {
      const found = THEMES.find((t) => t.key === key)
      if (found) {
        themeBg.value = found.bg
        themeAccent.value = found.accent
      }
    }
  } catch (e) {
    // localStorage unavailable — keep defaults.
  }
}

// ===================== Feature: Live room background music (直播间背景音乐) =====================
// A "🎵 BGM" button near the theme entry opens a popup with 3 ambient tracks.
// Selecting a track sets the active BGM (visual only — no actual audio plays),
// shows a "已切换: <name>" toast, and persists the choice to localStorage
// 'dy_live_bgm'. When a track is active, a pulsing 🎵 shows in the corner.
const BGM_KEY = 'dy_live_bgm'
const BGM_TRACKS = [
  { key: 'melody', name: '轻松旋律', icon: '🎵' },
  { key: 'electro', name: '节奏电音', icon: '🎧' },
  { key: 'nature', name: '自然白噪音', icon: '🌿' },
]
const showBgm = ref(false)       // popup visibility
const bgmTrack = ref(null)       // the active track object ({key,name,icon}) or null
// Derived: the active track's name (for the toast), '' when none.
function bgmName() {
  return bgmTrack.value ? bgmTrack.value.name : ''
}
// Selecting a track activates it, persists the choice, and toasts the switch.
function selectBgm(t) {
  bgmTrack.value = t
  try { localStorage.setItem(BGM_KEY, t.key) } catch (e) {}
  showBgm.value = false
  showToast('已切换: ' + t.name)
}
// Turn the BGM off: clear the active track + remove the persisted choice.
function clearBgm() {
  bgmTrack.value = null
  try { localStorage.removeItem(BGM_KEY) } catch (e) {}
  showBgm.value = false
  showToast('已关闭背景音乐')
}
// restoreBgm reads the saved track key on mount and reactivates it (no toast).
function restoreBgm() {
  try {
    const key = localStorage.getItem(BGM_KEY)
    if (key) {
      const found = BGM_TRACKS.find((t) => t.key === key)
      if (found) bgmTrack.value = found
    }
  } catch (e) {
    // localStorage unavailable — keep default (off).
  }
}

onMounted(async () => {
  try {
    const res = await getLiveRoom(route.params.id)
    room.value = res.room
    products.value = res.products || []
    likeCount.value = res.room?.likes || 0
    messages.value = res.messages || []
  } catch (e) {
    showToast('直播间不存在')
  } finally {
    loading.value = false
  }
  // Load the gift catalog (best-effort).
  getLiveGifts().then((data) => { gifts.value = data || [] }).catch(() => {})
  // Load any active PK + guard status for this room.
  getActivePK(route.params.id).then((d) => { pk.value = d || null }).catch(() => {})
  getGuardStatus(route.params.id).then((d) => { guardCount.value = d.count || 0; isGuarding.value = !!d.guarding }).catch(() => {})
  getActiveRedPacket(route.params.id).then((d) => { if (d) { redPacket.value = d; startRain() } }).catch(() => {})
  loadContributors()
  // Feature: 直播间主题装扮 — restore the saved theme on mount.
  restoreTheme()
  // Feature: 直播间背景音乐 — restore the saved BGM track on mount (no toast).
  restoreBgm()
  // Feature: 直播间幸运数字 — load today's lucky points on mount.
  loadLuckyPoints()
  // Feature: 直播间幸运转盘 — load accumulated wheel points on mount.
  loadWheelPoints()
  // Feature: 主播开播提醒 — load the armed state + next schedule on mount.
  loadReminder()
  loadSchedules()
  pollTimer = setInterval(pollMessages, 3000)
  // Feature: 直播间热度计 — initial compute + refresh every 10s.
  computeHeat()
  heatTimer = setInterval(() => {
    loadContributors()
    computeHeat()
  }, 10000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  // Clean up dice game timers to avoid leaks when leaving the room.
  if (diceSpinTimer) clearInterval(diceSpinTimer)
  if (diceStopTimer) clearTimeout(diceStopTimer)
  // Feature: 直播间幸运转盘 — clear the spin completion timer.
  if (wheelStopTimer) { clearTimeout(wheelStopTimer); wheelStopTimer = null }
  // Feature: 排行榜更新动画 — clear the flash-class reset timer.
  if (changeClassTimer) clearTimeout(changeClassTimer)
  // Feature: 直播间热度计 — clear the 10s refresh timer.
  if (heatTimer) { clearInterval(heatTimer); heatTimer = null }
  // Feature: 直播间网络质量 — clear the tooltip auto-hide timer.
  if (qualityTipTimer) { clearTimeout(qualityTipTimer); qualityTipTimer = null }
})

async function pollMessages() {
  // Lightweight: reserved for real-time chat polling.
}

function doLike() {
  likeCount.value++
  const id = Date.now()
  floatingHearts.value.push(id)
  setTimeout(() => {
    floatingHearts.value = floatingHearts.value.filter((i) => i !== id)
  }, 1500)
  likeLive(route.params.id).catch(() => {})
  // Feature: 直播间热度计 — keep heat in sync with likes.
  computeHeat()
}

async function sendMessage() {
  const text = msgText.value.trim()
  if (!text) return
  try {
    const m = await sendLiveMessage(route.params.id, text)
    messages.value.push(m)
    msgText.value = ''
    await nextTick()
    scrollMsgs()
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}

// Ban a user from the room (禁言)
async function doBan(m) {
  if (!m.user_id || m.username === '我') return
  try {
    showDialog({ title: '禁言', message: `确定禁言 ${m.username}？`, showCancelButton: true }).then(async () => {
      await banUser(route.params.id, m.user_id)
      showSuccessToast('已禁言')
    })
  } catch (e) {
    showToast('操作失败')
  }
}

// Send a gift: high-value gifts (price >= 100) trigger a full-screen mega
// effect; normal gifts keep the existing flying-heart style animation. Both
// echo a notice into the chat.
function sendGift(g) {
  const id = Date.now() + Math.random()
  const isMega = (g.price || 0) >= 100
  if (isMega) {
    // Full-screen mega gift effect (全屏礼物特效).
    const eid = ++giftEffectSeq
    giftEffects.value.push({ id: eid, icon: g.icon, name: g.name, username: '我', is_mega: true })
    // Auto-remove after the ~3s animation completes.
    setTimeout(() => {
      giftEffects.value = giftEffects.value.filter((f) => f.id !== eid)
    }, 3000)
  } else {
    // Normal gift — keep the existing flying animation.
    flyingGifts.value.push({ id, icon: g.icon, name: g.name })
    setTimeout(() => {
      flyingGifts.value = flyingGifts.value.filter((f) => f.id !== id)
    }, 2500)
  }
  // Echo a notice into the chat.
  messages.value.push({ id, username: '我', content: `送出 ${g.icon} ${g.name}`, is_gift: true })
  nextTick(scrollMsgs)
  showGifts.value = false
  showSuccessToast(`送出 ${g.icon} ${g.name}`)
}

// ---- PK ----
async function doStartPK() {
  try {
    pk.value = await startPK(route.params.id)
    showSuccessToast('PK已开始！为你的主播加油')
  } catch (e) {
    showToast(e.response?.data?.error || 'PK失败')
  }
}
async function cheer(side) {
  if (!pk.value) return
  try {
    pk.value = await scorePK(pk.value.id, side, 10)
  } catch (e) {
    showToast('加油失败')
  }
}
function pkPercent() {
  if (!pk.value) return 50
  const total = pk.value.score_a + pk.value.score_b
  if (total === 0) return 50
  return Math.round((pk.value.score_a / total) * 100)
}

// ---- Fan guard ----
async function doGuard() {
  try {
    const res = await guardHost(route.params.id)
    isGuarding.value = res.guarding
    guardCount.value = res.count
    showSuccessToast(res.guarding ? '守护成功！' : '已取消守护')
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}

// ---- Red packets (红包雨) ----
async function doDropPacket() {
  try {
    const p = await dropRedPacket(route.params.id, 10, 10)
    redPacket.value = p
    startRain()
    showSuccessToast('红包已发出！')
  } catch (e) {
    showToast('发送失败')
  }
}
// Animate falling red packets across the screen.
function startRain() {
  let count = 0
  const rain = setInterval(() => {
    const id = Date.now() + Math.random()
    fallingPackets.value.push({ id, left: Math.random() * 90 })
    setTimeout(() => { fallingPackets.value = fallingPackets.value.filter((f) => f.id !== id) }, 3000)
    count++
    if (count > 12) clearInterval(rain)
  }, 250)
}
async function grab() {
  try {
    const res = await grabRedPacket(route.params.id)
    showSuccessToast(res.message)
    redPacket.value = await getActiveRedPacket(route.params.id)
    if (!redPacket.value) fallingPackets.value = []
  } catch (e) {
    showToast(e.response?.data?.error || '手慢了')
  }
}

// ---- Contribution board (贡献榜) ----
async function loadContributors() {
  try {
    const fresh = await getContributors(route.params.id)
    computeContributorChanges(fresh)
    contributors.value = fresh
  } catch (e) {
    computeContributorChanges([])
    contributors.value = []
  }
}

// ===================== Feature: Leaderboard update animation (排行榜更新动画) =====================
// When the contributor list refreshes (after contribute/ban actions), animate
// the changed rows:
//   - items that moved up (rank improved) get a green flash
//   - items that moved down (rank dropped) get a brief red flash
//   - newly appeared items slide in from the right (handled by <transition-group>)
//
// prevRank is a map of { [user_id]: rankIndex } captured after each refresh, so
// the next refresh can compare old vs new positions. changeClass holds the
// per-user flash class applied for ~1.2s after a refresh.
const prevRank = ref({})                 // { [user_id]: previousIndex }
const changeClass = ref({})              // { [user_id]: 'flash-up' | 'flash-down' }
let changeClassTimer = null

// computeContributorChanges compares the incoming list against the previously
// stored positions, populates changeClass, then records the new baseline. New
// items get no flash class here — <transition-group> handles their slide-in.
function computeContributorChanges(fresh) {
  const newRank = {}
  const classes = {}
  fresh.forEach((c, i) => {
    const uid = c.user_id
    newRank[uid] = i
    if (prevRank.value[uid] != null) {
      const old = prevRank.value[uid]
      // A lower index = a higher rank = moved up. Only flag real changes so an
      // unchanged list doesn't flash.
      if (i < old) classes[uid] = 'flash-up'
      else if (i > old) classes[uid] = 'flash-down'
    }
    // Items not in prevRank are new — they animate via transition-group instead.
  })
  changeClass.value = classes
  prevRank.value = newRank
  // Clear the flash classes after the animation so a subsequent identical
  // refresh (same positions) flashes again correctly.
  if (changeClassTimer) clearTimeout(changeClassTimer)
  changeClassTimer = setTimeout(() => { changeClass.value = {} }, 1200)
}

// contributorClass returns the per-item animation class (if any) for a row.
function contributorClass(c) {
  return changeClass.value[c.user_id] || ''
}
async function doContribute() {
  try {
    await contribute(route.params.id, 10)
    showSuccessToast('已打榜 +10')
    await loadContributors()
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}

// ===================== Feature: 幸运骰子小游戏 (lucky dice mini-game) =====================
// A pure-frontend dice game in the live room. Rolling animates a spinning dice
// for ~1.5s then reveals a random value 1-6. We track the last 5 results and
// the best (highest) roll for the session, with a fun result message per value.
const showDice = ref(false)
const diceValue = ref(1)        // currently displayed pip value
const diceRolling = ref(false)  // spinning state
const diceHistory = ref([])     // last 5 results (newest last)
const diceFace = ref('🎲')      // emoji shown while idle/rolling
let diceSpinTimer = null
let diceStopTimer = null

// Fun message based on the rolled value.
function diceMessage(v) {
  switch (v) {
    case 6: return '大吉！运气爆棚 🎉'
    case 5: return '大吉大利 🍗'
    case 4: return '稳中向好 👍'
    case 3: return '中规中矩 🙂'
    case 2: return '差一点点 😅'
    default: return '再接再厉 💪'
  }
}

// Best (highest) roll across the session history.
const diceBest = computed(() => {
  if (!diceHistory.value.length) return null
  return Math.max(...diceHistory.value)
})

function rollDice() {
  if (diceRolling.value) return
  diceRolling.value = true
  // Rapidly cycle faces + the spinning animation class drives the CSS spin.
  if (diceSpinTimer) clearInterval(diceSpinTimer)
  const faces = ['⚀', '⚁', '⚂', '⚃', '⚄', '⚅']
  diceSpinTimer = setInterval(() => {
    diceFace.value = faces[Math.floor(Math.random() * 6)]
  }, 90)
  // Stop after ~1.5s and lock in a random result.
  if (diceStopTimer) clearTimeout(diceStopTimer)
  diceStopTimer = setTimeout(() => {
    clearInterval(diceSpinTimer)
    diceSpinTimer = null
    const result = Math.floor(Math.random() * 6) + 1
    diceValue.value = result
    diceFace.value = faces[result - 1]
    diceRolling.value = false
    // Keep only the last 5 results.
    diceHistory.value.push(result)
    if (diceHistory.value.length > 5) diceHistory.value.shift()
    showToast(diceMessage(result))
  }, 1500)
}

// ===================== Feature: Live room lucky number (直播间幸运数字) =====================
// A pure-frontend guessing game. The user picks a number 1-9; the system rolls
// a random 1-9. An exact match awards +100 lucky points and triggers a golden
// explosion animation; a near miss (±1) awards +10; otherwise nothing. Lucky
// points persist per-day in localStorage (keyed by YYYY-MM-DD) so "今日幸运值"
// resets each day.
const LUCKY_KEY = 'dy_lucky_points'
const showLucky = ref(false)        // popup visibility
const luckyPick = ref(null)         // the number the user selected (1-9)
const luckyResult = ref(null)       // { pick, roll, points, msg, hit } from the last guess
const luckyPoints = ref(0)          // today's accumulated lucky points
const luckyExplosions = ref([])     // active golden explosion particles [{id}]
let luckyExplosionSeq = 0

// todayKey returns the current date as YYYY-MM-DD for per-day point storage.
function todayKey() {
  const d = new Date()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}

// loadLuckyPoints restores today's points from localStorage. The stored value is
// an object keyed by date, so points from a previous day are ignored (reset).
function loadLuckyPoints() {
  try {
    const raw = localStorage.getItem(LUCKY_KEY)
    if (!raw) { luckyPoints.value = 0; return }
    const map = JSON.parse(raw)
    luckyPoints.value = map[todayKey()] || 0
  } catch (e) {
    luckyPoints.value = 0
  }
}

// saveLuckyPoints persists today's points back to the date-keyed map.
function saveLuckyPoints() {
  try {
    const raw = localStorage.getItem(LUCKY_KEY)
    const map = raw ? JSON.parse(raw) : {}
    map[todayKey()] = luckyPoints.value
    localStorage.setItem(LUCKY_KEY, JSON.stringify(map))
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
}

// playLucky resolves a guess: rolls 1-9, computes points + message, triggers the
// golden explosion on a hit, and accumulates points into today's total.
function playLucky() {
  if (luckyPick.value == null) return
  const pick = luckyPick.value
  const roll = Math.floor(Math.random() * 9) + 1
  let points = 0
  let msg = ''
  let hit = false
  if (pick === roll) {
    points = 100
    msg = '🎉 恭喜！幸运值+100'
    hit = true
    triggerLuckyExplosion()
  } else if (Math.abs(pick - roll) === 1) {
    points = 10
    msg = '差一点点！幸运值+10'
  } else {
    msg = '再试试吧~'
  }
  luckyPoints.value += points
  if (points > 0) saveLuckyPoints()
  luckyResult.value = { pick, roll, points, msg, hit }
}

// resetLuckyRound clears the selection + result so the user can guess again.
function resetLuckyRound() {
  luckyPick.value = null
  luckyResult.value = null
}

// triggerLuckyExplosion spawns a burst of golden particles that animate outward,
// each auto-removed after the ~1s animation finishes.
function triggerLuckyExplosion() {
  for (let i = 0; i < 24; i++) {
    const id = ++luckyExplosionSeq
    luckyExplosions.value.push({
      id,
      // Spread the burst across the top half of the popup centered on the result.
      x: 50 + (Math.random() - 0.5) * 60,
      y: 30 + (Math.random() - 0.5) * 40,
      angle: Math.random() * Math.PI * 2,
    })
    setTimeout(() => {
      luckyExplosions.value = luckyExplosions.value.filter((p) => p.id !== id)
    }, 1000)
  }
}

// ===================== Feature: 直播间幸运转盘 (lucky wheel mini-game) =====================
// A spinning 8-segment wheel. One free spin per live room visit is granted
// (tracked in localStorage keyed by room ID). Spinning animates a CSS rotate
// for 3s with an ease-out curve, lands on a random segment, and shows either a
// confetti burst (for prize segments) or a plain toast (for 谢谢参与). Prize
// points accumulate into localStorage 'dy_wheel_points'.
const WHEEL_KEY = 'dy_wheel_points'         // total accumulated wheel points
const WHEEL_SPUN_PREFIX = 'dy_wheel_spun_'  // + roomId → '1' once the free spin is used
const WHEEL_PRIZES = [
  { label: '积分x10',  color: '#fe2c55', points: 10,  kind: 'points' },
  { label: '红包',     color: '#ff6b9d', points: 5,   kind: 'prize'  },
  { label: '积分x20',  color: '#25f4ee', points: 20,  kind: 'points' },
  { label: '谢谢参与', color: '#888888', points: 0,   kind: 'none'   },
  { label: '积分x50',  color: '#fe2c55', points: 50,  kind: 'points' },
  { label: '铁粉+1',   color: '#ff6b9d', points: 8,   kind: 'prize'  },
  { label: '守护徽章', color: '#25f4ee', points: 15,  kind: 'prize'  },
  { label: '再转一次', color: '#ffb400', points: 0,   kind: 'respin' },
]
const showWheel = ref(false)             // popup visibility
const wheelSpinning = ref(false)         // true while a spin is animating
const wheelRotation = ref(0)             // cumulative degrees applied to the wheel
const wheelPoints = ref(0)               // total points accumulated from the wheel
const wheelConfetti = ref([])            // active confetti particles [{id}]
let wheelConfettiSeq = 0
let wheelStopTimer = null

// The 8 segments are 45° each. To draw them we expose each prize as a slice
// with its angular bounds so the template can build conic-gradient stops.
const wheelSlices = computed(() => {
  return WHEEL_PRIZES.map((p, i) => ({
    ...p,
    start: i * 45,
    end: (i + 1) * 45,
    mid: i * 45 + 22.5,
  }))
})
// conicGradient builds the wheel face as a single conic-gradient string.
const conicGradient = computed(() => {
  const stops = wheelSlices.value.map((s) => `${s.color} ${s.start}deg ${s.end}deg`)
  return `conic-gradient(${stops.join(', ')})`
})

// loadWheelPoints restores the accumulated wheel total from localStorage.
function loadWheelPoints() {
  try {
    wheelPoints.value = parseInt(localStorage.getItem(WHEEL_KEY) || '0', 10) || 0
  } catch (e) {
    wheelPoints.value = 0
  }
}
// saveWheelPoints persists the running total.
function saveWheelPoints() {
  try {
    localStorage.setItem(WHEEL_KEY, String(wheelPoints.value))
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
}
// hasFreeSpin returns true until the user has used their one free spin for the
// current room visit. The flag is keyed by room ID so each room grants one spin.
function hasFreeSpin() {
  try {
    return localStorage.getItem(WHEEL_SPUN_PREFIX + route.params.id) !== '1'
  } catch (e) {
    return true
  }
}
function markSpun() {
  try {
    localStorage.setItem(WHEEL_SPUN_PREFIX + route.params.id, '1')
  } catch (e) {
    // ignore
  }
}

// spinWheel performs a spin: picks a random segment, rotates the wheel so the
// pointer (at the top, 0°) lands within that segment, animates over 3s with an
// ease-out curve, then resolves the prize. "再转一次" refunds the free spin.
function spinWheel() {
  if (wheelSpinning.value) return
  if (!hasFreeSpin()) {
    showToast('本次直播免费转盘已用完')
    return
  }
  markSpun()
  wheelSpinning.value = true
  // Pick the winning segment, then aim the wheel so that segment's center lands
  // under the pointer at 0° (top). Add a little random jitter within the slice
  // so the pointer doesn't always sit dead-center.
  const segIndex = Math.floor(Math.random() * WHEEL_PRIZES.length)
  const seg = WHEEL_PRIZES[segIndex]
  const sliceWidth = 360 / WHEEL_PRIZES.length
  const jitter = (Math.random() - 0.5) * (sliceWidth - 8)
  const targetWithinSlice = segIndex * sliceWidth + sliceWidth / 2 + jitter
  // The pointer is at 0° (top). To bring the chosen slice center to the top we
  // rotate the wheel by -targetWithinSlice (mod 360). Add at least 5 full turns
  // so the spin feels substantial, and always rotate forward (cumulative).
  const fullTurns = 5 + Math.floor(Math.random() * 3) // 5–7 turns
  const current = wheelRotation.value
  // Normalize current rotation to its 0–360 remainder so the delta math stays
  // bounded, then add the full turns + the offset to reach the target slice.
  const currentMod = ((current % 360) + 360) % 360
  let desiredMod = (360 - targetWithinSlice) % 360
  if (desiredMod < 0) desiredMod += 360
  let delta = desiredMod - currentMod
  if (delta <= 0) delta += 360
  wheelRotation.value = current + fullTurns * 360 + delta

  if (wheelStopTimer) clearTimeout(wheelStopTimer)
  wheelStopTimer = setTimeout(() => {
    wheelSpinning.value = false
    resolveWheelPrize(seg)
  }, 3000)
}

// resolveWheelPrize surfaces the result. Prize/points segments fire a confetti
// burst and add points; 谢谢参与 shows a plain toast; 再转一次 refunds the free
// spin so the user can spin again.
function resolveWheelPrize(seg) {
  if (seg.kind === 'none') {
    showToast('谢谢参与，下次加油！')
    return
  }
  if (seg.kind === 'respin') {
    // Refund the free spin so the user gets another go.
    try { localStorage.removeItem(WHEEL_SPUN_PREFIX + route.params.id) } catch (e) {}
    triggerWheelConfetti()
    showToast('🎁 再转一次！可免费再玩一次')
    return
  }
  // Points / prize segment.
  wheelPoints.value += seg.points
  saveWheelPoints()
  triggerWheelConfetti()
  showSuccessToast(`🎉 恭喜中奖：${seg.label}（+${seg.points}积分）`)
}

// triggerWheelConfetti spawns a burst of colored particles that animate outward
// from the wheel center, each auto-removed after the animation finishes.
function triggerWheelConfetti() {
  const colors = ['#fe2c55', '#ff6b9d', '#25f4ee', '#ffb400', '#ffd700', '#ffffff']
  for (let i = 0; i < 36; i++) {
    const id = ++wheelConfettiSeq
    wheelConfetti.value.push({
      id,
      x: 50 + (Math.random() - 0.5) * 20,
      y: 45 + (Math.random() - 0.5) * 20,
      angle: Math.random() * Math.PI * 2,
      color: colors[Math.floor(Math.random() * colors.length)],
      size: 6 + Math.random() * 6,
      delay: Math.random() * 0.15,
    })
    setTimeout(() => {
      wheelConfetti.value = wheelConfetti.value.filter((p) => p.id !== id)
    }, 1400)
  }
}


// There is no dedicated viewer API, so we compose the list from:
//   1. recent contributors (already loaded — treated as "recent viewers")
//   2. suggested-follow users fetched lazily on first open ("also watching")
// The total viewer count comes from room.viewers; the popup lists a sample.
const showViewers = ref(false)
const viewers = ref([])
const viewersLoaded = ref(false)
// Derived total: the room's viewer count, falling back to the sample size.
function viewerCount() {
  const base = room.value?.viewers || viewers.value.length || 0
  return base
}
function openViewers() {
  showViewers.value = true
  // Start with contributors (already loaded) as recent viewers, then enrich
  // with suggested users the first time the popup is opened.
  if (!viewersLoaded.value) {
    viewersLoaded.value = true
    viewers.value = contributors.value.map((c) => ({
      id: c.user_id,
      nickname: c.nickname,
      avatar: c.avatar,
      label: '贡献观众',
    }))
    // Fetch a few suggested users and merge them in as "also watching".
    getSuggestFollows()
      .then((data) => {
        const sample = (data || []).slice(0, 6).map((u) => ({
          id: u.id,
          nickname: u.nickname,
          avatar: u.avatar,
          label: '也在看',
        }))
        // Deduplicate by id against the existing viewers.
        const seen = new Set(viewers.value.map((v) => v.id))
        sample.forEach((s) => { if (!seen.has(s.id)) viewers.value.push(s) })
      })
      .catch(() => {})
  }
}

function scrollMsgs() {
  if (msgListRef.value) msgListRef.value.scrollTop = msgListRef.value.scrollHeight
}

function fmt(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}

// ===================== Feature: Fan badge tier colors (粉丝勋章等级配色) =====================
// Contributors are bucketed by total contribution amount into colored tiers.
// The tier drives a small pill rendered next to each name in the contribution
// board, and an aggregate breakdown is shown in the guard count display.
//   >= 1000  -> 红色 铁粉 (iron fan)
//   >= 500   -> 橙色 忠粉 (loyal fan)
//   >= 100   -> 黄色 老粉 (old fan)
//   >= 10    -> 绿色 新粉 (new fan)
//   else     -> 灰色 路人 (passerby)
function fanTier(amount) {
  const a = amount || 0
  if (a >= 1000) return { label: '铁粉', cls: 'tier-iron', color: '#ff3b30' }
  if (a >= 500) return { label: '忠粉', cls: 'tier-loyal', color: '#ff9500' }
  if (a >= 100) return { label: '老粉', cls: 'tier-old', color: '#ffcc00' }
  if (a >= 10) return { label: '新粉', cls: 'tier-new', color: '#34c759' }
  return { label: '路人', cls: 'tier-passer', color: '#999' }
}

// Count how many contributors fall into the 铁粉 (>=1000) and 忠粉 (>=500)
// tiers, for the guard count breakdown. The guard count itself is the number
// of guarding users; the tier breakdown is computed from contributors as a
// proxy for fan loyalty.
const tierBreakdown = computed(() => {
  let iron = 0
  let loyal = 0
  for (const c of contributors.value) {
    const a = c.amount || 0
    if (a >= 1000) iron++
    else if (a >= 500) loyal++
  }
  return { iron, loyal }
})

// ===================== Feature: 主播名片 (live anchor intro card) =====================
// A bottom-sheet "business card" for the host. Tapping the host info in the top
// bar opens it. It fetches the host's full profile via getUser(host_id) and shows
// avatar, stats, bio, tags, and action buttons (follow / DM / homepage).
//   - follow uses the existing toggleFollow API with the host_id
//   - 私信 navigates to /chat/:host_id
//   - 查看主页 navigates to /user/:host_id
// Followers/likes come from the profile; "直播时长(小时)" is demo data.
const showAnchor = ref(false)
const anchor = ref(null)        // full host profile from getUser()
const anchorLoading = ref(false)
// Demo-only stat: total live hours is not tracked by the backend, so we derive a
// stable-ish demo number from the host id so it doesn't reshuffle every open.
const anchorLiveHours = computed(() => {
  const base = (room.value?.host_id || 1) % 200
  return base + 128
})

// Open the anchor card: show the sheet immediately with the room's basic host
// info, then enrich with the full profile from getUser.
function openAnchorCard() {
  showAnchor.value = true
  anchorLoading.value = true
  // Seed with the room's host info so the card isn't empty during the fetch.
  if (room.value) {
    anchor.value = {
      id: room.value.host_id,
      nickname: room.value.host_name,
      avatar: room.value.host_avatar,
      bio: '',
      followers_count: 0,
      likes_count: 0,
      is_following: false,
    }
  }
  const hid = room.value?.host_id
  if (!hid) { anchorLoading.value = false; return }
  getUser(hid)
    .then((u) => { anchor.value = u })
    .catch(() => {})
    .finally(() => { anchorLoading.value = false })
}

// Follow / unfollow the host from within the card.
async function doAnchorFollow() {
  if (!localStorage.getItem('dy_token')) { router.push('/login'); return }
  const hid = anchor.value?.id || room.value?.host_id
  if (!hid) return
  try {
    const res = await toggleFollow(hid)
    if (anchor.value) anchor.value.is_following = res.following
    showSuccessToast(res.following ? '关注成功' : '已取消关注')
  } catch (e) {
    showToast('操作失败')
  }
}

// ===================== Feature: Anchor schedule reminder (主播开播提醒) =====================
// A "🔔 开播提醒" button in the anchor intro card lets the user opt in to a
// reminder for the host's next scheduled live. The reminder is stored in
// localStorage keyed by host_id. When already set, the button shows an "已设置"
// state with a cancel option. The next scheduled time (from live_schedules) is
// shown as "下次直播: X月X日" below the button.
const REMINDER_PREFIX = 'dy_live_reminder_'
const reminderSet = ref(false)     // true when a reminder is armed for this host
const nextSchedule = ref(null)     // the host's next upcoming schedule (if any)
const schedules = ref([])          // all upcoming schedules

function reminderKey() {
  return REMINDER_PREFIX + (room.value?.host_id || '')
}

// loadReminder reads the armed state from localStorage for this room's host.
function loadReminder() {
  try {
    reminderSet.value = localStorage.getItem(reminderKey()) === '1'
  } catch (e) {
    reminderSet.value = false
  }
}

// loadSchedules fetches upcoming live schedules and resolves the host's next one.
async function loadSchedules() {
  try {
    const data = await getLiveSchedules()
    schedules.value = data || []
    const hid = room.value?.host_id
    nextSchedule.value = hid
      ? (data || []).find((s) => s.host_id === hid) || null
      : null
  } catch (e) {
    schedules.value = []
    nextSchedule.value = null
  }
}

// toggleReminder arms or cancels the reminder, persisting to localStorage.
function toggleReminder() {
  if (reminderSet.value) {
    try { localStorage.removeItem(reminderKey()) } catch (e) {}
    reminderSet.value = false
    showToast('已取消开播提醒')
  } else {
    try { localStorage.setItem(reminderKey(), '1') } catch (e) {}
    reminderSet.value = true
    showSuccessToast('已开启开播提醒')
  }
}

// formatScheduleDate renders "X月X日" from a schedule's scheduled_time.
function formatScheduleDate(s) {
  if (!s || !s.scheduled_time) return ''
  const d = new Date(s.scheduled_time)
  if (isNaN(d.getTime())) return ''
  return `${d.getMonth() + 1}月${d.getDate()}日`
}

// ===================== Feature: Live room heat meter (直播间热度计) =====================
// A vertical thermometer gauge on the right side of the screen. "热度" is the
// sum of viewers + likes + the total contribution amount. The fill level + color
// tier is keyed off the viewer count:
//   < 1000      → blue   (冷)
//   1000–5000   → orange (温)
//   5000–10000  → red    (热)
//   > 10000     → purple (爆)
// The fill animates and bubbles at the top; a "🔥 热度: X" label sits beside it.
// The value recomputes every 10 seconds (and re-reads the latest contributors).
const heatValue = ref(0)
let heatTimer = null

// computeHeat sums viewers + likes + total contributor amount into heatValue.
function computeHeat() {
  const viewers = room.value?.viewers || 0
  const likes = likeCount.value || 0
  const contrib = contributors.value.reduce((s, c) => s + (c.amount || 0), 0)
  heatValue.value = viewers + likes + contrib
}

// heatTier resolves the current color tier from the viewer count.
// Returns { key, color, label } used to drive the fill color + bubble tint.
function heatTier() {
  const v = room.value?.viewers || 0
  if (v > 10000) return { key: 'boom', color: '#9c27b0', label: '爆' }
  if (v >= 5000) return { key: 'hot', color: '#ff3b30', label: '热' }
  if (v >= 1000) return { key: 'warm', color: '#ff9500', label: '温' }
  return { key: 'cold', color: '#4facfe', label: '冷' }
}

// heatFillPct maps the viewer count to a 0–100 fill level (log-scaled so small
// rooms still show a visible fill and very large rooms don't overflow).
function heatFillPct() {
  const v = room.value?.viewers || 0
  if (v <= 0) return 4
  // log1p keeps a 0–10000 spread readable; cap at 100.
  const pct = Math.round((Math.log1p(v) / Math.log1p(12000)) * 100)
  return Math.max(4, Math.min(100, pct))
}

// ===================== Feature: Live room connection quality (直播间网络质量) =====================
// A signal-bars icon (4 bars) in the top-right corner. The quality level is
// derived deterministically from a hash of the room id, so the same room always
// shows the same quality:
//   bars 4 → excellent → green   → "网络极佳 4G"
//   bars 3 → good      → light-green → "网络良好"
//   bars 2 → fair      → orange  → "网络一般"
//   bars 1 → poor      → red     → "网络较差"
// Tapping the icon surfaces the matching tooltip text.
const showQualityTip = ref(false)
let qualityTipTimer = null

// hashRoomId turns the room id string into a small non-negative integer. Used so
// the quality is deterministic per room rather than random per visit.
function hashRoomId(id) {
  const s = String(id == null ? '' : id)
  let h = 0
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) >>> 0
  }
  return h
}

// qualityInfo derives { bars, color, tip } deterministically from the room id.
function qualityInfo() {
  const id = route.params.id
  const level = (hashRoomId(id) % 4) + 1 // 1..4
  if (level >= 4) return { bars: 4, color: '#34c759', tip: '网络极佳 4G' }       // green
  if (level === 3) return { bars: 3, color: '#8ed98e', tip: '网络良好' }          // light-green
  if (level === 2) return { bars: 2, color: '#ff9500', tip: '网络一般' }          // orange
  return { bars: 1, color: '#ff3b30', tip: '网络较差' }                            // red
}

// tapQuality shows the tooltip text for ~2s, then hides it.
function tapQuality() {
  showQualityTip.value = true
  if (qualityTipTimer) clearTimeout(qualityTipTimer)
  qualityTipTimer = setTimeout(() => { showQualityTip.value = false }, 2000)
}
</script>

<template>
  <div class="room-page" v-if="loading">
    <div class="loading-center"><van-loading color="#fe2c55" /></div>
  </div>
  <div class="room-page" v-else-if="room" :style="{ background: themeBg }">
    <!-- HLS video player -->
    <video class="live-video" :src="room.stream_url" autoplay muted loop playsinline></video>

    <!-- Top bar -->
    <div class="top-bar">
      <van-icon name="arrow-left" size="22" color="#fff" @click="router.back()" />
      <!-- Feature: 主播名片 — tapping the host info opens the anchor intro card -->
      <div class="host-info host-info-tap" @click="openAnchorCard">
        <img class="host-avatar" :style="{ borderColor: themeAccent }" :src="room.host_avatar" />
        <div>
          <div class="host-name">{{ room.host_name }}</div>
          <div class="host-viewers">{{ fmt(room.viewers) }}观看</div>
        </div>
        <van-icon name="arrow-down" size="12" color="rgba(255,255,255,0.7)" class="host-arrow" />
      </div>
      <van-button size="mini" round :color="isGuarding ? '#9c27b0' : '#333'" @click="doGuard">{{ isGuarding ? '已守护' : '守护' }}</van-button>
    </div>

    <!-- ===================== Feature: 直播间网络质量 (connection quality) =====================
         A 4-bar signal icon in the top-right corner. The level + color is derived
         deterministically from the room id hash; tapping it shows a tooltip. -->
    <div class="signal-wrap" @click="tapQuality">
      <div class="signal-bars" :title="qualityInfo().tip">
        <span
          v-for="b in 4"
          :key="b"
          class="signal-bar"
          :class="{ off: b > qualityInfo().bars, anim: b <= qualityInfo().bars }"
          :style="b <= qualityInfo().bars ? { '--bar-i': b, background: qualityInfo().color, animationDelay: (b * 0.15) + 's' } : {}"
        ></span>
      </div>
      <transition name="quality-tip">
        <span v-if="showQualityTip" class="signal-tip">{{ qualityInfo().tip }}</span>
      </transition>
    </div>

    <!-- Title -->
    <div class="room-title">{{ room.title }}</div>

    <!-- PK banner -->
    <div v-if="pk" class="pk-banner">
      <div class="pk-side pk-left" @click="cheer('a')">
        <span class="pk-name">{{ pk.room_a_name }}</span>
        <span class="pk-score">{{ pk.score_a }}</span>
      </div>
      <div class="pk-bar-wrap">
        <div class="pk-vs">PK</div>
        <div class="pk-bar"><div class="pk-fill" :style="{ width: pkPercent() + '%' }"></div></div>
      </div>
      <div class="pk-side pk-right" @click="cheer('b')">
        <span class="pk-score">{{ pk.score_b }}</span>
        <span class="pk-name">{{ pk.room_b_name }}</span>
      </div>
    </div>
    <div v-else class="pk-start" @click="doStartPK">⚔️ 发起PK</div>

    <!-- Guard count — with fan-tier breakdown (粉丝勋章等级配色) -->
    <div v-if="guardCount > 0" class="guard-info">
      🛡️ {{ guardCount }}人守护
      <span v-if="tierBreakdown.iron || tierBreakdown.loyal" class="guard-breakdown">
        (铁粉{{ tierBreakdown.iron }} 忠粉{{ tierBreakdown.loyal }})
      </span>
    </div>
    <!-- Danmaku toggle -->
    <div class="dm-toggle" @click="showDanmaku = !showDanmaku">{{ showDanmaku ? '🙈 隐藏弹幕' : '💬 显示弹幕' }}</div>

    <!-- Highlight moments entry (直播高光时刻) — gold badge button -->
    <div class="highlight-entry" @click="showHighlights = true">
      <span class="hl-badge">✨</span>
      <span>高光时刻</span>
    </div>

    <!-- Contribution board entry (贡献榜) -->
    <div class="contrib-entry" @click="showContributors = true">
      🏆 贡献榜
      <div v-if="contributors.length" class="ce-avatars">
        <img v-for="(c, i) in contributors.slice(0, 3)" :key="i" class="ce-avatar" :class="'rank-' + i" :src="c.avatar" />
      </div>
    </div>

    <!-- ===================== Feature: 幸运骰子小游戏 (lucky dice) entry ===================== -->
    <div class="dice-entry" @click="showDice = true">🎲 骰子</div>

    <!-- ===================== Feature: Live room lucky number (直播间幸运数字) entry ===================== -->
    <div class="lucky-entry" @click="showLucky = true">🔮 幸运数字</div>

    <!-- ===================== Feature: 直播间幸运转盘 (lucky wheel) entry ===================== -->
    <div class="wheel-entry" @click="showWheel = true">🎡 幸运转盘</div>

    <!-- ===================== Feature: 直播间主题装扮 (theme picker) ===================== -->
    <div class="theme-entry" @click="showTheme = true">
      <span class="theme-swatch" :style="{ background: themeAccent }"></span>
      🎨 主题
    </div>

    <!-- ===================== Feature: 直播间背景音乐 (BGM) ===================== -->
    <div class="bgm-entry" :class="{ active: bgmTrack }" @click="showBgm = true">
      🎵 BGM<span v-if="bgmTrack" class="bgm-entry-name">{{ bgmName() }}</span>
    </div>
    <!-- Pulsing music-note indicator shown in the corner while BGM is on. -->
    <div v-if="bgmTrack" class="bgm-indicator">🎵</div>
    <van-popup v-model:show="showTheme" position="bottom" round>
      <div class="theme-panel">
        <div class="tp-head">🎨 主题装扮</div>
        <div class="tp-sub">选择直播间主题配色</div>
        <div class="tp-grid">
          <div
            v-for="t in THEMES"
            :key="t.key"
            class="tp-item"
            :class="{ active: themeBg === t.bg }"
            @click="applyTheme(t)"
          >
            <span class="tp-swatch" :style="{ background: t.bg }">
              <span class="tp-accent-dot" :style="{ background: t.accent }"></span>
            </span>
            <span class="tp-label">{{ t.label }}</span>
          </div>
        </div>
      </div>
    </van-popup>

    <!-- ===================== Feature: 直播间背景音乐 (BGM) popup =====================
         Three ambient tracks (visual only, no actual audio). Selecting one sets the
         active BGM + toasts the switch; a 关闭 button clears it. -->
    <van-popup v-model:show="showBgm" position="bottom" round>
      <div class="bgm-panel">
        <div class="bgm-head">🎵 背景音乐</div>
        <div class="bgm-sub">为直播间选择氛围音乐（示例）</div>
        <div class="bgm-list">
          <div
            v-for="t in BGM_TRACKS"
            :key="t.key"
            class="bgm-item"
            :class="{ active: bgmTrack && bgmTrack.key === t.key }"
            @click="selectBgm(t)"
          >
            <span class="bgm-icon">{{ t.icon }}</span>
            <span class="bgm-name">{{ t.name }}</span>
            <span v-if="bgmTrack && bgmTrack.key === t.key" class="bgm-check">✓</span>
          </div>
        </div>
        <van-button block round class="bgm-off-btn" @click="clearBgm">关闭背景音乐</van-button>
      </div>
    </van-popup>

    <!-- Live viewer list entry (直播间观众列表) — shows count + opens popup -->
    <div class="viewer-entry" @click="openViewers">
      👥 观众 {{ fmt(viewerCount()) }}
    </div>

    <!-- Viewer list popup (直播间观众列表) -->
    <van-popup v-model:show="showViewers" position="bottom" round :style="{ height: '50%' }">
      <div class="viewer-panel">
        <div class="vp-head">
          👥 观众列表
          <span class="vp-count">{{ fmt(viewerCount()) }} 人在看</span>
        </div>
        <div class="vp-sub">本场直播的观众（示例数据）</div>
        <div class="vp-list">
          <div v-for="(u, i) in viewers" :key="u.id + '-' + i" class="vp-item">
            <img class="vp-avatar" :src="u.avatar || 'https://via.placeholder.com/40'" />
            <div class="vp-info">
              <div class="vp-name van-ellipsis">{{ u.nickname }}</div>
              <div class="vp-label" :class="{ recent: u.label === '贡献观众' }">{{ u.label }}</div>
            </div>
          </div>
          <div v-if="!viewers.length" class="vp-empty">暂无观众数据</div>
        </div>
      </div>
    </van-popup>

    <!-- Contribution board popup -->
    <van-popup v-model:show="showContributors" position="bottom" round>
      <div class="contrib-panel">
        <div class="cp-head">🏆 贡献榜</div>
        <div v-if="!contributors.length" class="cp-empty">暂无贡献，快来打榜吧</div>
        <!-- ===================== Feature: 排行榜更新动画 (leaderboard animation) =====================
             transition-group slides newly-appeared contributors in from the right.
             Per-row changeClass adds a green/red flash when a row moved up/down. -->
        <transition-group name="contrib-slide" tag="div" class="contrib-list">
          <div
            v-for="(c, i) in contributors"
            :key="c.user_id"
            class="cp-item"
            :class="contributorClass(c)"
          >
            <span class="cp-rank" :class="{ top: i < 3 }">{{ i + 1 }}</span>
            <img class="cp-avatar" :src="c.avatar" />
            <span class="cp-name">
              {{ c.nickname }}
              <!-- Feature: 粉丝勋章等级配色 — tier-colored pill next to the name -->
              <span class="fan-badge" :class="fanTier(c.amount).cls">{{ fanTier(c.amount).label }}</span>
            </span>
            <span class="cp-amount">{{ c.amount }}</span>
          </div>
        </transition-group>
        <van-button block round :color="themeAccent" style="margin-top: 16px" @click="doContribute">为TA打榜 +10</van-button>
      </div>
    </van-popup>

    <!-- Highlight moments popup (直播高光时刻) — timeline list -->
    <van-popup v-model:show="showHighlights" position="bottom" round :style="{ height: '50%' }">
      <div class="highlight-panel">
        <div class="hp-head"><span class="hp-icon">✨</span> 高光时刻</div>
        <div class="hp-sub">本场直播的精彩瞬间</div>
        <div class="hp-list">
          <div v-for="(h, i) in highlights" :key="i" class="hp-item">
            <div class="hp-dot-wrap">
              <span class="hp-dot" :class="{ first: i === 0 }"></span>
              <span v-if="i < highlights.length - 1" class="hp-line"></span>
            </div>
            <div class="hp-content">
              <div class="hp-time">{{ h.time }}</div>
              <div class="hp-text">{{ h.text }}</div>
            </div>
          </div>
        </div>
      </div>
    </van-popup>

    <!-- Red packet drop button + rain -->
    <div v-if="!redPacket" class="rp-drop" @click="doDropPacket">🧧 发红包</div>
    <div v-if="redPacket" class="rp-banner" @click="grab">
      🧧 红包雨进行中 · 剩余 {{ redPacket.remaining }}/{{ redPacket.total }} · 点击抢
    </div>
    <div class="rp-rain">
      <div v-for="f in fallingPackets" :key="f.id" class="rp-fall" :style="{ left: f.left + '%' }">🧧</div>
    </div>

    <!-- Danmaku / chat list -->
    <div v-if="showDanmaku" class="danmaku-layer" ref="msgListRef">
      <div v-for="m in messages" :key="m.id" class="dm-item" @longpress="doBan(m)">
        <span class="dm-user">{{ m.username }}:</span>
        <span class="dm-text">{{ m.content }}</span>
        <span v-if="m.user_id && m.username !== '我'" class="dm-ban" @click.stop="doBan(m)">🚫</span>
      </div>
    </div>

    <!-- Floating hearts -->
    <div class="hearts-layer">
      <div v-for="id in floatingHearts" :key="id" class="floating-heart">❤</div>
    </div>

    <!-- Right action rail -->
    <div class="action-rail">
      <div class="action-item" @click="doLike">
        <van-icon name="like" :color="themeAccent" size="32" />
        <span>{{ fmt(likeCount) }}</span>
      </div>
      <div class="action-item" @click="showGifts = true">
        <van-icon name="gift-o" color="#ffd700" size="32" />
        <span>礼物</span>
      </div>
      <div class="action-item" @click="showCart = true">
        <van-icon name="shopping-cart-o" color="#fff" size="32" />
        <span>{{ products.length }}件商品</span>
      </div>
      <div class="action-item" @click="router.push('/feed')">
        <van-icon name="cross" color="#fff" size="32" />
        <span>关闭</span>
      </div>
    </div>

    <!-- ===================== Feature: 直播间热度计 (heat meter) =====================
         A vertical thermometer gauge on the right side. The fill level + color tier
         (cold/warm/hot/boom) is keyed off the viewer count; heat = viewers +
         likes + contribution. Bubbles animate at the top of the fill. -->
    <div class="heat-meter" :class="'heat-' + heatTier().key">
      <div class="heat-bulb" :style="{ background: heatTier().color }"></div>
      <div class="heat-tube">
        <div class="heat-fill" :style="{ height: heatFillPct() + '%', background: heatTier().color }">
          <span class="heat-bubble heat-bubble-1"></span>
          <span class="heat-bubble heat-bubble-2"></span>
          <span class="heat-bubble heat-bubble-3"></span>
        </div>
      </div>
      <div class="heat-label">🔥 热度</div>
      <div class="heat-value">{{ fmt(heatValue) }}</div>
    </div>

    <!-- Flying gifts animation layer -->
    <div class="gift-layer">
      <div v-for="g in flyingGifts" :key="g.id" class="flying-gift">
        <span class="fg-icon">{{ g.icon }}</span>
      </div>
    </div>

    <!-- ===================== Feature: Full-screen mega gift effect (全屏礼物特效) =====================
         Triggered by high-value gifts (price >= 100). Each effect renders:
         - a golden flash overlay (0.5s) at the start,
         - a large gift emoji flying diagonally across the screen (~3s),
         - an announcement banner at top center with a gold gradient.
         Auto-removed after the animation completes. -->
    <div class="mega-gift-layer">
      <template v-for="fx in giftEffects" :key="fx.id">
        <!-- Golden flash overlay — quick 0.5s burst when the effect starts. -->
        <div class="mega-flash"></div>
        <!-- Large gift emoji flying diagonally across the screen. -->
        <div class="mega-fly">{{ fx.icon }}</div>
        <!-- Announcement banner at top center. -->
        <div class="mega-banner">
          <span class="mb-user">{{ fx.username }}</span> 送出了 <span class="mb-name">{{ fx.name }}</span><span class="mb-icon">{{ fx.icon }}</span>
        </div>
      </template>
    </div>

    <!-- Bottom: chat input + product teaser (小黄车入口) -->
    <div class="bottom-bar">
      <div class="chat-input">
        <input v-model="msgText" placeholder="说点什么..." @keyup.enter="sendMessage" />
        <van-icon name="smile-comment-o" :color="themeAccent" size="22" @click="sendMessage" />
      </div>
      <div class="cart-teaser" @click="showCart = true" v-if="products.length">
        <img class="teaser-img" :src="products[0].image" />
        <div class="teaser-info">
          <div class="teaser-name van-ellipsis">{{ products[0].name }}</div>
          <div class="teaser-price">¥{{ products[0].price.toFixed(2) }}</div>
        </div>
        <div class="teaser-btn">{{ products.length }}</div>
      </div>
    </div>

    <!-- Cart popup (小黄车) -->
    <van-popup v-model:show="showCart" position="bottom" round :style="{ height: '50%' }">
      <div class="cart-panel">
        <div class="cp-head">购物车 ({{ products.length }}件好物)</div>
        <div class="cp-list">
          <div v-for="p in products" :key="p.id" class="cp-item">
            <img class="cp-img" :src="p.image" />
            <div class="cp-body">
              <div class="cp-name van-ellipsis">{{ p.name }}</div>
              <div class="cp-meta">
                <span class="cp-price">¥{{ p.price.toFixed(2) }}</span>
                <span class="cp-sales">已售{{ p.sales }}件</span>
              </div>
            </div>
            <van-button size="mini" round color="#fe2c55" @click="showSuccessToast('已加入购物车')">抢购</van-button>
          </div>
        </div>
      </div>
    </van-popup>

    <!-- ===================== Feature: 幸运骰子小游戏 (lucky dice) popup ===================== -->
    <van-popup v-model:show="showDice" position="bottom" round :style="{ height: '46%' }">
      <div class="dice-panel">
        <div class="dice-head">🎲 幸运骰子</div>
        <div class="dice-sub">纯前端小游戏 · 摇一摇试试手气</div>

        <!-- Animated dice -->
        <div class="dice-stage">
          <div class="dice-cube" :class="{ rolling: diceRolling }">{{ diceFace }}</div>
        </div>

        <!-- Current result + message -->
        <div class="dice-result">
          <span v-if="!diceRolling" class="dr-val">{{ diceValue }} 点</span>
          <span v-else class="dr-val">摇骰中…</span>
        </div>
        <div v-if="!diceRolling && diceHistory.length" class="dice-msg">{{ diceMessage(diceValue) }}</div>

        <!-- Roll history (last 5) -->
        <div v-if="diceHistory.length" class="dice-history">
          <span class="dh-label">最近：</span>
          <span v-for="(h, i) in diceHistory" :key="i" class="dh-num">{{ h }}</span>
        </div>
        <!-- Best roll -->
        <div v-if="diceBest" class="dice-best">手气最佳：{{ diceBest }} 点</div>

        <van-button
          block
          round
          :color="themeAccent"
          :loading="diceRolling"
          class="dice-roll-btn"
          @click="rollDice"
        >{{ diceRolling ? '摇骰中…' : '摇骰子' }}</van-button>
      </div>
    </van-popup>

    <!-- ===================== Feature: Live room lucky number (直播间幸运数字) popup =====================
         Pick 1-9, system rolls 1-9. Match → +100 + golden explosion; ±1 → +10;
         else → 再试试吧. Lucky points persist per-day in localStorage. -->
    <van-popup v-model:show="showLucky" position="bottom" round :style="{ height: '50%' }">
      <div class="lucky-panel">
        <div class="lucky-head">🔮 幸运数字</div>
        <div class="lucky-sub">猜中1-9，幸运值+100 · 差一点+10</div>

        <!-- Today's lucky points -->
        <div class="lucky-points">今日幸运值: <span>{{ luckyPoints }}</span></div>

        <!-- Number picker 1-9 -->
        <div v-if="!luckyResult" class="lucky-pick">
          <div class="lp-prompt">请选择一个数字 (1-9)</div>
          <div class="lp-grid">
            <div
              v-for="n in 9"
              :key="n"
              class="lp-num"
              :class="{ picked: luckyPick === n }"
              @click="luckyPick = n"
            >{{ n }}</div>
          </div>
          <van-button
            block
            round
            :color="themeAccent"
            :disabled="luckyPick == null"
            class="lp-go-btn"
            @click="playLucky"
          >开始猜数字</van-button>
        </div>

        <!-- Result view -->
        <div v-else class="lucky-result" :class="{ hit: luckyResult.hit }">
          <!-- Golden explosion particles on a match -->
          <div class="lp-burst-layer">
            <div
              v-for="p in luckyExplosions"
              :key="p.id"
              class="lp-particle"
              :style="{ left: p.x + '%', top: p.y + '%', '--ang': p.angle + 'rad' }"
            >✨</div>
          </div>
          <div class="lr-roll">
            <span class="lr-label">你选的</span>
            <span class="lr-num">{{ luckyResult.pick }}</span>
            <span class="lr-vs">VS</span>
            <span class="lr-label">系统</span>
            <span class="lr-num">{{ luckyResult.roll }}</span>
          </div>
          <div class="lr-msg" :class="{ hit: luckyResult.hit }">{{ luckyResult.msg }}</div>
          <van-button block round :color="themeAccent" class="lr-again-btn" @click="resetLuckyRound">再来一次</van-button>
        </div>
      </div>
    </van-popup>

    <!-- ===================== Feature: 直播间幸运转盘 (lucky wheel) popup =====================
         An 8-segment conic-gradient wheel. One free spin per room visit; spinning
         animates a 3s ease-out rotate to a random segment. Prizes fire confetti;
         谢谢参与 shows a plain toast; 再转一次 refunds the free spin. -->
    <van-popup v-model:show="showWheel" position="bottom" round :style="{ height: '62%' }">
      <div class="wheel-panel">
        <div class="wheel-head">🎡 幸运转盘</div>
        <div class="wheel-sub">每次直播 1 次免费机会 · 累计积分 <span>{{ wheelPoints }}</span></div>

        <div class="wheel-stage">
          <!-- Pointer fixed at the top (12 o'clock). -->
          <div class="wheel-pointer">▼</div>
          <!-- The wheel itself. transition handles the 3s ease-out spin. -->
          <div
            class="wheel-disc"
            :class="{ spinning: wheelSpinning }"
            :style="{
              background: conicGradient,
              transform: `rotate(${wheelRotation}deg)`,
              transition: wheelSpinning ? 'transform 3s cubic-bezier(0.16, 1, 0.3, 1)' : 'none',
            }"
          >
            <!-- Segment labels positioned around the disc. -->
            <span
              v-for="(s, i) in wheelSlices"
              :key="i"
              class="wheel-seg-label"
              :style="{ transform: `rotate(${s.mid}deg) translateY(-78px)` }"
            >{{ s.label }}</span>
          </div>
          <!-- Confetti layer overlays the wheel during a prize burst. -->
          <div class="wheel-confetti-layer">
            <span
              v-for="p in wheelConfetti"
              :key="p.id"
              class="wheel-confetti"
              :style="{
                left: p.x + '%',
                top: p.y + '%',
                background: p.color,
                width: p.size + 'px',
                height: p.size + 'px',
                '--angle': p.angle + 'rad',
                animationDelay: p.delay + 's',
              }"
            ></span>
          </div>
        </div>

        <van-button
          block
          round
          :color="themeAccent"
          :loading="wheelSpinning"
          :disabled="!hasFreeSpin()"
          class="wheel-spin-btn"
          @click="spinWheel"
        >{{ wheelSpinning ? '转动中…' : (hasFreeSpin() ? '🎡 免费转一次' : '本次机会已用完') }}</van-button>
        <div class="wheel-hint">中奖积分已累计到本地（dy_wheel_points）</div>
      </div>
    </van-popup>

    <!-- Gift tray popup -->
    <van-popup v-model:show="showGifts" position="bottom" round :style="{ height: '40%' }">
      <div class="gift-panel">
        <div class="gp-head">送礼</div>
        <div class="gp-grid">
          <div v-for="g in gifts" :key="g.id" class="gp-item" @click="sendGift(g)">
            <div class="gp-icon">{{ g.icon }}</div>
            <div class="gp-name">{{ g.name }}</div>
            <div class="gp-price">{{ g.price }}</div>
          </div>
          <van-empty v-if="!gifts.length" description="暂无礼物" image="search" />
        </div>
      </div>
    </van-popup>

    <!-- ===================== Feature: 主播名片 (anchor intro card) ===================== -->
    <van-popup v-model:show="showAnchor" position="bottom" round closeable :style="{ height: '62%' }">
      <div class="anchor-card">
        <!-- Gradient header with large avatar + name -->
        <div class="ac-header">
          <div class="ac-gradient"></div>
          <img class="ac-avatar" :src="anchor?.avatar || 'https://via.placeholder.com/80'" />
          <div class="ac-name van-ellipsis">{{ anchor?.nickname || room?.host_name }}</div>
          <!-- Tags / badges -->
          <div class="ac-tags">
            <span class="ac-tag tag-creator">优质创作者</span>
            <span class="ac-tag tag-active">活跃主播</span>
          </div>
        </div>

        <div v-if="anchorLoading" class="ac-loading"><van-loading color="#fe2c55" /></div>

        <template v-else>
          <!-- Stats row: followers / likes / live hours (demo) -->
          <div class="ac-stats">
            <div class="ac-stat">
              <div class="ac-stat-val">{{ fmt(anchor?.followers_count || 0) }}</div>
              <div class="ac-stat-label">粉丝</div>
            </div>
            <div class="ac-divider"></div>
            <div class="ac-stat">
              <div class="ac-stat-val">{{ fmt(anchor?.likes_count || 0) }}</div>
              <div class="ac-stat-label">获赞</div>
            </div>
            <div class="ac-divider"></div>
            <div class="ac-stat">
              <div class="ac-stat-val">{{ fmt(anchorLiveHours) }}</div>
              <div class="ac-stat-label">直播时长(时)</div>
            </div>
          </div>

          <!-- Bio -->
          <div class="ac-bio">
            {{ anchor?.bio || '这位主播很懒，还没有填写简介～' }}
          </div>

          <!-- Action buttons -->
          <div class="ac-actions">
            <!-- ===================== Feature: 主播开播提醒 (schedule reminder) =====================
                 Toggles a reminder stored in localStorage keyed by host_id. Shows the
                 next scheduled live time (from live_schedules) below the button. -->
            <div class="ac-reminder-block">
              <van-button
                round
                block
                class="ac-btn"
                :color="reminderSet ? '#444' : '#fe2c55'"
                @click="toggleReminder"
              >{{ reminderSet ? '🔔 已设置提醒' : '🔔 开播提醒' }}</van-button>
              <div v-if="reminderSet && nextSchedule" class="ac-next-live">
                下次直播: {{ formatScheduleDate(nextSchedule) }}
              </div>
              <div v-else-if="nextSchedule" class="ac-next-live">下次直播: {{ formatScheduleDate(nextSchedule) }}</div>
            </div>
            <van-button
              round
              block
              :color="anchor?.is_following ? '#333' : '#fe2c55'"
              class="ac-btn"
              @click="doAnchorFollow"
            >{{ anchor?.is_following ? '已关注' : '+ 关注' }}</van-button>
            <van-button round block class="ac-btn ac-btn-outline" @click="router.push('/chat/' + (anchor?.id || room?.host_id))">私信</van-button>
            <van-button round block class="ac-btn ac-btn-outline" @click="router.push('/user/' + (anchor?.id || room?.host_id))">查看主页</van-button>
          </div>
        </template>
      </div>
    </van-popup>
  </div>
  <div v-else class="room-page"><van-empty description="直播间不存在" /></div>
</template>

<style scoped>
.room-page { height: 100vh; background: #000; position: relative; overflow: hidden; transition: background-color 0.4s ease; }
.loading-center { display: flex; align-items: center; justify-content: center; height: 100%; }
.live-video { width: 100%; height: 100%; object-fit: cover; }
.top-bar { position: absolute; top: 0; left: 0; right: 0; display: flex; align-items: center; gap: 10px; padding: 16px; z-index: 10; background: linear-gradient(to bottom, rgba(0,0,0,0.4), transparent); }
.host-info { display: flex; align-items: center; gap: 8px; flex: 1; }
.host-avatar { width: 36px; height: 36px; border-radius: 50%; border: 2px solid #fe2c55; }
.host-name { color: #fff; font-size: 14px; font-weight: bold; }
.host-viewers { color: rgba(255,255,255,0.7); font-size: 11px; }

/* ===================== Feature: 直播间网络质量 (connection quality) =====================
   A 4-bar signal icon fixed in the top-right corner. Each lit bar is colored by
   the quality tier (green/light-green/orange/red) and gently oscillates in
   height. Tapping the icon reveals a tooltip with the textual quality. */
.signal-wrap {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 12;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 12px;
  background: rgba(0,0,0,0.35);
  backdrop-filter: blur(4px);
}
.signal-bars {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 16px;
}
.signal-bar {
  width: 3px;
  background: rgba(255,255,255,0.3);
  border-radius: 1.5px;
  /* bar heights: 1=shortest, 4=tallest */
  height: calc(var(--bar-h, 4px));
}
/* Off bars render at a minimal height with the muted color. */
.signal-bar.off { --bar-h: 4px; }
.signal-bar.anim { animation: signalOsc 1.2s ease-in-out infinite; }
/* Per-bar base heights via nth-child so the staircase shape is preserved. */
.signal-bar:nth-child(1) { --bar-h: 5px; }
.signal-bar:nth-child(2) { --bar-h: 8px; }
.signal-bar:nth-child(3) { --bar-h: 12px; }
.signal-bar:nth-child(4) { --bar-h: 16px; }
@keyframes signalOsc {
  0%, 100% { transform: scaleY(1); }
  50%      { transform: scaleY(0.6); }
}
/* Tooltip text — appears on tap, fades in/out. */
.signal-tip {
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
  text-shadow: 0 1px 3px rgba(0,0,0,0.6);
}
.quality-tip-enter-active, .quality-tip-leave-active { transition: opacity 0.2s; }
.quality-tip-enter-from, .quality-tip-leave-to { opacity: 0; }
.room-title { position: absolute; top: 70px; left: 16px; right: 60px; color: #fff; font-size: 15px; font-weight: bold; text-shadow: 0 1px 3px rgba(0,0,0,0.5); z-index: 10; }
.pk-start { position: absolute; top: 100px; left: 16px; z-index: 10; background: rgba(254,44,85,0.8); color: #fff; font-size: 12px; padding: 4px 12px; border-radius: 12px; }
.pk-banner { position: absolute; top: 100px; left: 12px; right: 12px; z-index: 10; display: flex; align-items: center; gap: 8px; background: rgba(0,0,0,0.6); border-radius: 20px; padding: 6px 12px; }
.pk-side { display: flex; align-items: center; gap: 6px; cursor: pointer; flex: 0 0 auto; }
.pk-name { color: #fff; font-size: 11px; max-width: 60px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pk-score { color: #fe2c55; font-size: 14px; font-weight: bold; }
.pk-bar-wrap { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 3px; }
.pk-vs { color: #ffd700; font-size: 11px; font-weight: bold; }
.pk-bar { width: 100%; height: 6px; background: #fe2c55; border-radius: 3px; overflow: hidden; }
.pk-fill { height: 100%; background: #25f4ee; transition: width 0.3s; }
.guard-info { position: absolute; top: 140px; left: 16px; z-index: 10; color: #9c27b0; font-size: 11px; background: rgba(0,0,0,0.5); padding: 2px 8px; border-radius: 10px; }
.dm-toggle { position: absolute; top: 168px; left: 16px; z-index: 10; background: rgba(0,0,0,0.5); color: #25f4ee; font-size: 11px; padding: 2px 8px; border-radius: 10px; cursor: pointer; }
.contrib-entry { position: absolute; top: 140px; right: 16px; z-index: 10; background: rgba(0,0,0,0.5); color: #ffd700; font-size: 11px; padding: 4px 10px; border-radius: 12px; display: flex; align-items: center; gap: 4px; }
.ce-avatars { display: flex; }
.ce-avatar { width: 18px; height: 18px; border-radius: 50%; border: 1px solid #fff; margin-left: -6px; }
.ce-avatar.rank-0 { border-color: #ffd700; }
.ce-avatar.rank-1 { border-color: #c0c0c0; }
.ce-avatar.rank-2 { border-color: #cd7f32; }

/* ===================== Feature: Live viewer list (直播间观众列表) styles ===================== */
/* Viewer entry button — placed below the contribution board entry */
.viewer-entry {
  position: absolute;
  top: 196px;
  right: 16px;
  z-index: 10;
  background: rgba(0,0,0,0.5);
  color: #fff;
  font-size: 11px;
  padding: 4px 10px;
  border-radius: 12px;
  cursor: pointer;
  border: 1px solid rgba(254,44,85,0.5);
}
.viewer-entry:active { background: rgba(254,44,85,0.7); }
/* Viewer popup panel */
.viewer-panel { background: #161616; height: 100%; display: flex; flex-direction: column; padding: 16px; }
.vp-head { display: flex; align-items: center; justify-content: space-between; color: #fff; font-size: 16px; font-weight: bold; }
.vp-count { color: #fe2c55; font-size: 13px; font-weight: normal; }
.vp-sub { color: #888; font-size: 12px; margin: 4px 0 12px; }
.vp-list { flex: 1; overflow-y: auto; display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; align-content: start; }
.vp-item { display: flex; align-items: center; gap: 10px; padding: 8px; background: #1f1f1f; border-radius: 10px; }
.vp-avatar { width: 40px; height: 40px; border-radius: 50%; flex-shrink: 0; border: 1px solid #333; }
.vp-info { flex: 1; min-width: 0; }
.vp-name { color: #fff; font-size: 13px; }
.vp-label { color: #888; font-size: 11px; margin-top: 2px; }
.vp-label.recent { color: #ffd700; }
.vp-empty { grid-column: 1 / -1; text-align: center; color: #666; padding: 40px; }
.contrib-panel { padding: 16px; background: #161616; }
.cp-head { text-align: center; color: #ffd700; font-size: 16px; font-weight: bold; margin-bottom: 16px; }
.cp-empty { text-align: center; color: #666; padding: 30px; }
.cp-item { display: flex; align-items: center; gap: 12px; padding: 10px 0; border-bottom: 1px solid #222; }
.cp-rank { width: 24px; text-align: center; color: #666; font-weight: bold; font-size: 15px; }
.cp-rank.top { color: #ffd700; }
.cp-avatar { width: 36px; height: 36px; border-radius: 50%; }
.cp-name { flex: 1; color: #fff; font-size: 14px; }
.cp-amount { color: #fe2c55; font-weight: bold; font-size: 14px; }

/* ===================== Feature: Leaderboard update animation (排行榜更新动画) =====================
   When the contributor list refreshes, rows that changed rank get a brief flash:
   moved up → green flash, moved down → red flash. New rows slide in from the
   right via Vue's <transition-group>. */
.contrib-list { display: flex; flex-direction: column; position: relative; }
/* Green flash for a row whose rank improved (moved up the board). */
.cp-item.flash-up {
  animation: contribFlashUp 1.2s ease-out;
}
/* Red flash for a row whose rank dropped (moved down the board). */
.cp-item.flash-down {
  animation: contribFlashDown 1.2s ease-out;
}
@keyframes contribFlashUp {
  0%   { background: rgba(52, 199, 89, 0.45); box-shadow: inset 3px 0 0 #34c759; }
  100% { background: transparent; box-shadow: inset 3px 0 0 transparent; }
}
@keyframes contribFlashDown {
  0%   { background: rgba(255, 59, 48, 0.45); box-shadow: inset 3px 0 0 #ff3b30; }
  100% { background: transparent; box-shadow: inset 3px 0 0 transparent; }
}
/* transition-group: newly-added contributors slide in from the right. */
.contrib-slide-enter-active {
  transition: transform 0.4s ease-out, opacity 0.4s ease-out;
}
.contrib-slide-leave-active {
  transition: transform 0.3s ease-in, opacity 0.3s ease-in;
  position: absolute;
  width: 100%;
}
.contrib-slide-enter-from {
  transform: translateX(60px);
  opacity: 0;
}
.contrib-slide-leave-to {
  transform: translateX(-30px);
  opacity: 0;
}
/* Smooth repositioning of existing rows when the order changes. */
.contrib-slide-move {
  transition: transform 0.4s ease;
}
/* ===================== Highlight moments (直播高光时刻) ===================== */
/* Floating gold badge button near the PK/guard area */
.highlight-entry {
  position: absolute;
  top: 168px;
  right: 16px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 4px;
  background: linear-gradient(135deg, rgba(255,215,0,0.95), rgba(255,180,0,0.95));
  color: #4a2c00;
  font-size: 11px;
  font-weight: bold;
  padding: 4px 10px;
  border-radius: 12px;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(255,215,0,0.4);
}
.highlight-entry:active { transform: scale(0.95); }
.hl-badge { font-size: 13px; animation: hlSparkle 1.6s ease-in-out infinite; }
@keyframes hlSparkle { 0%,100% { transform: scale(1); } 50% { transform: scale(1.25); } }
/* Highlight popup — timeline-style list */
.highlight-panel { background: #161616; height: 100%; display: flex; flex-direction: column; padding: 16px; }
.hp-head { text-align: center; color: #ffd700; font-size: 17px; font-weight: bold; }
.hp-icon { margin-right: 4px; }
.hp-sub { text-align: center; color: #888; font-size: 12px; margin-top: 4px; margin-bottom: 16px; }
.hp-list { flex: 1; overflow-y: auto; padding: 0 4px; }
.hp-item { display: flex; gap: 12px; }
.hp-dot-wrap { display: flex; flex-direction: column; align-items: center; flex-shrink: 0; padding-top: 2px; }
.hp-dot { width: 12px; height: 12px; border-radius: 50%; background: #ffd700; border: 2px solid rgba(255,215,0,0.3); box-shadow: 0 0 8px rgba(255,215,0,0.6); }
.hp-dot.first { animation: hlPulse 1.5s ease-in-out infinite; }
@keyframes hlPulse { 0%,100% { box-shadow: 0 0 8px rgba(255,215,0,0.6); } 50% { box-shadow: 0 0 16px rgba(255,215,0,1); } }
.hp-line { flex: 1; width: 2px; background: rgba(255,215,0,0.3); margin: 4px 0; min-height: 28px; }
.hp-content { flex: 1; padding-bottom: 18px; }
.hp-time { color: #ffd700; font-size: 12px; font-weight: 500; margin-bottom: 3px; }
.hp-text { color: #fff; font-size: 14px; line-height: 20px; }
.rp-drop { position: absolute; top: 140px; right: 16px; z-index: 10; background: rgba(255,0,54,0.85); color: #fff; font-size: 12px; padding: 4px 10px; border-radius: 12px; }
.rp-banner { position: absolute; top: 168px; left: 12px; right: 12px; z-index: 10; background: linear-gradient(90deg, #ff0036, #ff9800); color: #fff; text-align: center; font-size: 12px; padding: 6px; border-radius: 16px; }
.rp-rain { position: absolute; top: 0; left: 0; right: 0; bottom: 0; z-index: 14; pointer-events: none; }
.rp-fall { position: absolute; top: -40px; font-size: 28px; animation: rpFall 3s linear forwards; }
@keyframes rpFall { 0% { transform: translateY(0); opacity: 1; } 100% { transform: translateY(100vh); opacity: 0.6; } }
.hearts-layer { position: absolute; bottom: 100px; right: 24px; z-index: 15; pointer-events: none; }
.floating-heart { font-size: 28px; color: #fe2c55; animation: floatUp 1.5s ease-out forwards; position: absolute; bottom: 0; right: 0; }
@keyframes floatUp { 0% { transform: translateY(0) scale(0.8); opacity: 1; } 100% { transform: translateY(-200px) scale(1.2); opacity: 0; } }
.action-rail { position: absolute; right: 10px; bottom: 100px; display: flex; flex-direction: column; align-items: center; gap: 16px; z-index: 10; }
.action-item { display: flex; flex-direction: column; align-items: center; gap: 3px; }
.action-item span { color: #fff; font-size: 11px; }
.cart-teaser { position: absolute; bottom: 20px; left: 12px; right: 12px; background: rgba(0,0,0,0.6); border-radius: 8px; padding: 8px; display: flex; align-items: center; gap: 8px; z-index: 10; }
.danmaku-layer { position: absolute; left: 12px; bottom: 80px; width: 70%; max-height: 40%; overflow-y: auto; z-index: 10; display: flex; flex-direction: column; gap: 4px; scrollbar-width: none; }
.danmaku-layer::-webkit-scrollbar { display: none; }
.dm-item { background: rgba(0,0,0,0.35); border-radius: 14px; padding: 4px 10px; color: #fff; font-size: 12px; line-height: 18px; align-self: flex-start; max-width: 100%; }
.dm-user { color: #fe2c55; margin-right: 4px; }
.dm-text { color: #fff; word-break: break-all; }
.dm-ban { margin-left: 6px; font-size: 10px; opacity: 0.6; cursor: pointer; }
.bottom-bar { position: absolute; bottom: 12px; left: 12px; right: 60px; z-index: 10; display: flex; flex-direction: column; gap: 8px; }
.chat-input { display: flex; align-items: center; gap: 8px; background: rgba(0,0,0,0.5); border-radius: 20px; padding: 4px 12px; }
.chat-input input { flex: 1; background: transparent; border: none; outline: none; color: #fff; font-size: 13px; height: 32px; }
.chat-input input::placeholder { color: rgba(255,255,255,0.5); }
.cart-teaser { background: rgba(0,0,0,0.6); border-radius: 8px; padding: 8px; display: flex; align-items: center; gap: 8px; position: static; }
.teaser-img { width: 40px; height: 40px; border-radius: 6px; }
.teaser-info { flex: 1; min-width: 0; }
.teaser-name { color: #fff; font-size: 13px; }
.teaser-price { color: #fe2c55; font-size: 14px; font-weight: bold; }
.teaser-btn { background: #fe2c55; color: #fff; font-size: 12px; padding: 4px 10px; border-radius: 12px; flex-shrink: 0; }
.cart-panel { background: #161616; height: 100%; display: flex; flex-direction: column; }
.cp-head { text-align: center; padding: 14px; color: #fff; font-size: 15px; border-bottom: 1px solid #222; }
.cp-list { flex: 1; overflow-y: auto; padding: 8px 12px; }
.cp-item { display: flex; align-items: center; gap: 10px; padding: 10px 0; border-bottom: 1px solid #1a1a1a; }
.cp-img { width: 50px; height: 50px; border-radius: 6px; flex-shrink: 0; }
.cp-body { flex: 1; min-width: 0; }
.cp-name { color: #fff; font-size: 14px; }
.cp-meta { display: flex; gap: 10px; align-items: baseline; margin-top: 4px; }
.cp-price { color: #fe2c55; font-size: 16px; font-weight: bold; }
.cp-sales { color: #999; font-size: 11px; }
.gift-layer { position: absolute; top: 30%; left: 0; right: 0; z-index: 16; pointer-events: none; display: flex; justify-content: center; }
.flying-gift { animation: flyGift 2.5s ease-out forwards; }
.fg-icon { font-size: 64px; filter: drop-shadow(0 0 12px rgba(255, 215, 0, 0.8)); }
@keyframes flyGift { 0% { transform: translateY(60px) scale(0.4); opacity: 0; } 25% { transform: translateY(0) scale(1.4); opacity: 1; } 80% { transform: translateY(-20px) scale(1.2); opacity: 1; } 100% { transform: translateY(-80px) scale(1); opacity: 0; } }

/* ===================== Feature: Full-screen mega gift effect (全屏礼物特效) ===================== */
/* Layer covering the whole room; holds the flash, flying emoji, and banner.
   High z-index so it renders above all other overlays. */
.mega-gift-layer { position: absolute; inset: 0; z-index: 40; pointer-events: none; }
/* Golden flash overlay — a 0.5s burst that fills the screen then fades. */
.mega-flash {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle, rgba(255, 215, 0, 0.85) 0%, rgba(255, 215, 0, 0.3) 60%, rgba(255, 215, 0, 0) 100%);
  animation: megaFlash 0.5s ease-out forwards;
}
@keyframes megaFlash {
  0%   { opacity: 0; }
  20%  { opacity: 1; }
  100% { opacity: 0; }
}
/* Large gift emoji flying diagonally (bottom-left -> top-right) over ~3s. */
.mega-fly {
  position: absolute;
  top: 0;
  left: 0;
  font-size: 140px;
  line-height: 1;
  filter: drop-shadow(0 0 24px rgba(255, 215, 0, 0.9));
  animation: megaFly 3s ease-in forwards;
}
@keyframes megaFly {
  0%   { transform: translate(-20vw, 90vh) rotate(0deg) scale(0.6); opacity: 0; }
  15%  { opacity: 1; }
  85%  { opacity: 1; }
  100% { transform: translate(90vw, -20vh) rotate(35deg) scale(1.4); opacity: 0; }
}
/* Announcement banner at top center — gold gradient text, fades after ~3s. */
.mega-banner {
  position: absolute;
  top: 80px;
  left: 50%;
  transform: translateX(-50%);
  padding: 10px 24px;
  font-size: 18px;
  font-weight: bold;
  white-space: nowrap;
  border-radius: 24px;
  background: linear-gradient(90deg, rgba(255, 215, 0, 0.25), rgba(255, 180, 0, 0.15));
  border: 1px solid rgba(255, 215, 0, 0.6);
  box-shadow: 0 0 20px rgba(255, 215, 0, 0.5);
  animation: megaBanner 3s ease-out forwards;
}
.mega-banner .mb-user { color: #fff; }
.mega-banner .mb-name {
  background: linear-gradient(90deg, #ffd700, #ffae00);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  color: #ffd700;
  margin: 0 4px;
}
.mega-banner .mb-icon { margin-left: 4px; }
@keyframes megaBanner {
  0%   { opacity: 0; transform: translate(-50%, -20px) scale(0.8); }
  15%  { opacity: 1; transform: translate(-50%, 0) scale(1); }
  80%  { opacity: 1; transform: translate(-50%, 0) scale(1); }
  100% { opacity: 0; transform: translate(-50%, -20px) scale(0.95); }
}
.gift-panel { background: #161616; height: 100%; display: flex; flex-direction: column; }
.gp-head { text-align: center; padding: 14px; color: #fff; font-size: 15px; border-bottom: 1px solid #222; }
.gp-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; padding: 12px; overflow-y: auto; }
.gp-item { display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 8px 4px; border-radius: 8px; }
.gp-item:active { background: #222; }
.gp-icon { font-size: 32px; }
.gp-name { color: #fff; font-size: 12px; }
.gp-price { color: #ffd700; font-size: 11px; }

/* ===================== Feature: Fan badge tier colors (粉丝勋章等级配色) ===================== */
/* Guard count breakdown — smaller, lighter text inside the guard info pill. */
.guard-breakdown { color: rgba(255, 255, 255, 0.75); font-size: 10px; margin-left: 2px; }
/* Tier-colored pill next to each contributor's name in the contribution board. */
.fan-badge {
  display: inline-flex;
  align-items: center;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  padding: 1px 6px;
  border-radius: 8px;
  line-height: 14px;
  margin-left: 6px;
  vertical-align: middle;
  white-space: nowrap;
}
/* >= 1000: 红色 铁粉 */
.fan-badge.tier-iron { background: #ff3b30; box-shadow: 0 0 6px rgba(255, 59, 48, 0.5); }
/* >= 500: 橙色 忠粉 */
.fan-badge.tier-loyal { background: #ff9500; }
/* >= 100: 黄色 老粉 — dark text for readability on yellow */
.fan-badge.tier-old { background: #ffcc00; color: #4a3500; }
/* >= 10: 绿色 新粉 */
.fan-badge.tier-new { background: #34c759; }
/* else: 灰色 路人 */
.fan-badge.tier-passer { background: #888; }

/* ===================== Feature: 幸运骰子小游戏 (lucky dice) ===================== */
/* Entry button — sits below the contribution board entry, themed red. */
.dice-entry {
  position: absolute;
  top: 224px;
  right: 16px;
  z-index: 10;
  background: rgba(254, 44, 85, 0.85);
  color: #fff;
  font-size: 11px;
  padding: 4px 10px;
  border-radius: 12px;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(254, 44, 85, 0.4);
}
.dice-entry:active { transform: scale(0.95); background: rgba(254, 44, 85, 1); }
/* Popup panel */
.dice-panel {
  background: #161616;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 16px;
  gap: 6px;
}
.dice-head { color: #fff; font-size: 17px; font-weight: bold; }
.dice-sub { color: #888; font-size: 12px; margin-bottom: 8px; }
/* Dice stage + animated emoji cube */
.dice-stage { perspective: 600px; margin: 6px 0; }
.dice-cube {
  font-size: 96px;
  line-height: 1;
  color: #fe2c55;
  filter: drop-shadow(0 0 16px rgba(254, 44, 85, 0.5));
  transform-style: preserve-3d;
}
.dice-cube.rolling { animation: diceSpin 0.35s linear infinite; }
@keyframes diceSpin {
  0% { transform: rotateX(0) rotateY(0) scale(1); }
  50% { transform: rotateX(180deg) rotateY(180deg) scale(1.15); }
  100% { transform: rotateX(360deg) rotateY(360deg) scale(1); }
}
.dice-result { margin-top: 8px; }
.dr-val { color: #fff; font-size: 20px; font-weight: bold; }
.dice-msg { color: #ffd700; font-size: 14px; margin-top: 2px; }
/* Roll history */
.dice-history { display: flex; align-items: center; gap: 8px; margin-top: 10px; }
.dh-label { color: #888; font-size: 12px; }
.dh-num {
  color: #fff;
  font-size: 13px;
  font-weight: bold;
  min-width: 24px;
  height: 24px;
  line-height: 24px;
  text-align: center;
  background: #2a2a2a;
  border-radius: 6px;
  padding: 0 6px;
}
/* Best roll */
.dice-best { color: #25f4ee; font-size: 13px; margin-top: 8px; }
.dice-roll-btn { margin-top: 14px; }

/* ===================== Feature: Live room lucky number (直播间幸运数字) ===================== */
/* Entry button — sits below the dice entry, gold-toned to match the lucky theme. */
.lucky-entry {
  position: absolute;
  top: 258px;
  right: 16px;
  z-index: 10;
  background: linear-gradient(135deg, rgba(255,215,0,0.95), rgba(255,180,0,0.95));
  color: #4a2c00;
  font-size: 11px;
  font-weight: bold;
  padding: 4px 10px;
  border-radius: 12px;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(255,215,0,0.4);
}
.lucky-entry:active { transform: scale(0.95); }

/* ===================== Feature: 直播间幸运转盘 (lucky wheel) =====================
   Entry button — sits just below the lucky-number entry, purple-toned to match
   the wheel-of-fortune theme. */
.wheel-entry {
  position: absolute;
  top: 292px;
  right: 16px;
  z-index: 10;
  background: linear-gradient(135deg, rgba(149, 90, 255, 0.95), rgba(254, 44, 85, 0.95));
  color: #fff;
  font-size: 11px;
  font-weight: bold;
  padding: 4px 10px;
  border-radius: 12px;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(149, 90, 255, 0.45);
}
.wheel-entry:active { transform: scale(0.95); }
/* Popup panel */
.wheel-panel {
  position: relative;
  background: #161616;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 16px;
  gap: 6px;
  overflow: hidden;
}
.wheel-head { color: #fff; font-size: 17px; font-weight: bold; }
.wheel-sub { color: #888; font-size: 12px; margin-bottom: 8px; }
.wheel-sub span { color: #ffd700; font-weight: bold; }
/* The wheel stage holds the pointer, the disc, and the confetti overlay. */
.wheel-stage {
  position: relative;
  width: 240px;
  height: 240px;
  margin: 14px 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
/* Fixed pointer at the top (12 o'clock). */
.wheel-pointer {
  position: absolute;
  top: -6px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 5;
  color: #fff;
  font-size: 24px;
  line-height: 1;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.7));
  pointer-events: none;
}
/* The spinning disc. Its background is a conic-gradient built from the prizes;
   its rotation is bound to wheelRotation and eased via the inline transition. */
.wheel-disc {
  position: relative;
  width: 220px;
  height: 220px;
  border-radius: 50%;
  border: 6px solid #fff;
  box-shadow: 0 0 24px rgba(254, 44, 85, 0.5);
}
.wheel-disc.spinning { /* rotation handled inline via transform + transition */ }
/* Segment labels arranged around the disc. Each is rotated to its slice's
   midpoint then pushed outward toward the rim. */
.wheel-seg-label {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 70px;
  margin-left: -35px;
  margin-top: -8px;
  text-align: center;
  color: #fff;
  font-size: 10px;
  font-weight: bold;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.6);
  pointer-events: none;
  transform-origin: center;
}
/* Confetti layer overlays the stage during a prize burst. */
.wheel-confetti-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 6;
}
.wheel-confetti {
  position: absolute;
  border-radius: 2px;
  opacity: 0;
  animation: wheelConfettiBurst 1.3s ease-out forwards;
}
@keyframes wheelConfettiBurst {
  0%   { opacity: 1; transform: translate(0, 0) rotate(0deg) scale(1); }
  100% {
    opacity: 0;
    /* Travel outward along the precomputed angle. */
    transform: translate(calc(cos(var(--angle)) * 110px), calc(sin(var(--angle)) * 110px + 40px)) rotate(540deg) scale(0.4);
  }
}
.wheel-spin-btn { margin-top: 10px; }
.wheel-hint { color: #555; font-size: 11px; margin-top: 6px; }
/* Popup panel */
.lucky-panel {
  position: relative;
  background: #161616;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 16px;
  gap: 6px;
  overflow: hidden;
}
.lucky-head { color: #ffd700; font-size: 17px; font-weight: bold; }
.lucky-sub { color: #888; font-size: 12px; margin-bottom: 4px; }
.lucky-points {
  color: #ffd700;
  font-size: 14px;
  font-weight: 600;
  background: rgba(255,215,0,0.12);
  border: 1px solid rgba(255,215,0,0.4);
  padding: 4px 14px;
  border-radius: 14px;
  margin-bottom: 6px;
}
.lucky-points span { font-size: 18px; font-weight: bold; }

/* Number picker */
.lucky-pick { display: flex; flex-direction: column; align-items: center; width: 100%; }
.lp-prompt { color: #fff; font-size: 14px; margin-bottom: 12px; }
.lp-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  width: 100%;
  max-width: 240px;
  margin-bottom: 18px;
}
.lp-num {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 52px;
  font-size: 22px;
  font-weight: bold;
  color: #fff;
  background: #2a2a2a;
  border: 2px solid #333;
  border-radius: 12px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.1s;
  user-select: none;
}
.lp-num:active { transform: scale(0.94); }
.lp-num.picked {
  background: rgba(255,215,0,0.18);
  border-color: #ffd700;
  color: #ffd700;
}
.lp-go-btn { max-width: 260px; }

/* Result view */
.lucky-result {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
}
.lr-roll {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #888;
  font-size: 12px;
  margin: 10px 0 12px;
}
.lr-num {
  color: #fff;
  font-size: 30px;
  font-weight: bold;
}
.lr-vs { color: #ffd700; font-size: 13px; font-weight: bold; margin: 0 4px; }
.lr-msg {
  color: #fff;
  font-size: 17px;
  font-weight: 600;
  text-align: center;
  margin-bottom: 18px;
}
.lr-msg.hit {
  color: #ffd700;
  font-size: 20px;
  text-shadow: 0 0 16px rgba(255,215,0,0.8);
  animation: lrHitPulse 0.6s ease-out;
}
@keyframes lrHitPulse {
  0% { transform: scale(0.7); opacity: 0; }
  60% { transform: scale(1.15); opacity: 1; }
  100% { transform: scale(1); opacity: 1; }
}
.lr-again-btn { max-width: 260px; }

/* Golden explosion — particles spawn near the result and fly outward. */
.lp-burst-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 5;
}
.lp-particle {
  position: absolute;
  font-size: 20px;
  transform: translate(-50%, -50%);
  animation: lpBurst 1s ease-out forwards;
}
@keyframes lpBurst {
  0% { transform: translate(-50%, -50%) translate(0, 0) scale(0.4) rotate(0deg); opacity: 1; }
  100% {
    transform: translate(-50%, -50%) translate(calc(cos(var(--ang)) * 90px), calc(sin(var(--ang)) * 90px)) scale(1.2) rotate(180deg);
    opacity: 0;
  }
}

/* ===================== Feature: 直播间主题装扮 (theme picker) ===================== */
/* Entry button near the top bar — small accent swatch + label. */
.theme-entry {
  position: absolute;
  top: 252px;
  right: 16px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(0,0,0,0.5);
  color: #fff;
  font-size: 11px;
  padding: 4px 10px;
  border-radius: 12px;
  cursor: pointer;
}
.theme-entry:active { transform: scale(0.95); }
.theme-swatch {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 1px solid rgba(255,255,255,0.6);
}
/* Popup panel */
.theme-panel { background: #161616; padding: 16px; }
.tp-head { text-align: center; color: #fff; font-size: 16px; font-weight: bold; }
.tp-sub { text-align: center; color: #888; font-size: 12px; margin: 4px 0 16px; }
.tp-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.tp-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 10px 6px;
  border-radius: 12px;
  border: 2px solid transparent;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}
.tp-item:active { background: #1f1f1f; }
.tp-item.active { border-color: #fe2c55; background: rgba(254,44,85,0.12); }
.tp-swatch {
  position: relative;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 2px solid rgba(255,255,255,0.25);
  box-shadow: inset 0 0 8px rgba(0,0,0,0.4);
}
.tp-accent-dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid #161616;
}
.tp-label { color: #fff; font-size: 12px; }

/* ===================== Feature: 直播间背景音乐 (BGM) ===================== */
/* Entry button — sits below the theme entry, turns themed-red when a track is on. */
.bgm-entry {
  position: absolute;
  top: 290px;
  right: 16px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(0,0,0,0.5);
  color: #fff;
  font-size: 11px;
  padding: 4px 10px;
  border-radius: 12px;
  cursor: pointer;
}
.bgm-entry.active {
  background: rgba(254,44,85,0.85);
  box-shadow: 0 2px 8px rgba(254,44,85,0.4);
}
.bgm-entry:active { transform: scale(0.95); }
.bgm-entry-name { font-size: 10px; opacity: 0.9; }

/* Pulsing music-note indicator in the corner while BGM is on. */
.bgm-indicator {
  position: absolute;
  bottom: 168px;
  left: 16px;
  z-index: 12;
  font-size: 22px;
  filter: drop-shadow(0 0 8px rgba(254,44,85,0.7));
  animation: bgmPulse 1.2s ease-in-out infinite;
  pointer-events: none;
}
@keyframes bgmPulse {
  0%, 100% { transform: scale(1); opacity: 0.7; }
  50% { transform: scale(1.35); opacity: 1; }
}

/* Popup panel */
.bgm-panel { background: #161616; padding: 16px; }
.bgm-head { text-align: center; color: #fff; font-size: 16px; font-weight: bold; }
.bgm-sub { text-align: center; color: #888; font-size: 12px; margin: 4px 0 16px; }
.bgm-list { display: flex; flex-direction: column; gap: 10px; }
.bgm-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #1f1f1f;
  border: 2px solid transparent;
  border-radius: 12px;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}
.bgm-item:active { background: #262626; }
.bgm-item.active { border-color: #fe2c55; background: rgba(254,44,85,0.12); }
.bgm-icon { font-size: 26px; }
.bgm-name { flex: 1; color: #fff; font-size: 15px; }
.bgm-check { color: #fe2c55; font-size: 18px; font-weight: bold; }
.bgm-off-btn { margin-top: 18px; background: #2a2a2a; color: #fff; }

/* ===================== Feature: 主播名片 (anchor intro card) ===================== */
/* Make the host info in the top bar feel tappable. */
.host-info-tap { cursor: pointer; padding: 2px 6px 2px 2px; border-radius: 18px; transition: background 0.2s; }
.host-info-tap:active { background: rgba(255,255,255,0.12); }
.host-arrow { margin-left: 2px; }

/* Anchor card sheet */
.anchor-card {
  background: #161616;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-bottom: 24px;
  overflow-y: auto;
}

/* Gradient header backdrop */
.ac-header {
  position: relative;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 28px 16px 18px;
  overflow: hidden;
}
.ac-gradient {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, #fe2c55 0%, #ff5c7a 40%, #9c27b0 100%);
  opacity: 0.9;
}
/* Keep header content above the gradient */
.ac-avatar,
.ac-name,
.ac-tags { position: relative; z-index: 1; }
.ac-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  border: 3px solid rgba(255,255,255,0.9);
  box-shadow: 0 4px 16px rgba(0,0,0,0.4);
}
.ac-name {
  color: #fff;
  font-size: 19px;
  font-weight: bold;
  margin-top: 10px;
  max-width: 70%;
  text-shadow: 0 1px 4px rgba(0,0,0,0.3);
}

/* Tags row */
.ac-tags { display: flex; gap: 6px; margin-top: 8px; flex-wrap: wrap; justify-content: center; }
.ac-tag {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 9px;
  line-height: 15px;
}
.tag-creator { background: rgba(255,255,255,0.25); color: #fff; border: 1px solid rgba(255,255,255,0.5); }
.tag-active { background: rgba(255,215,0,0.9); color: #4a3500; }

/* Stats row */
.ac-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  padding: 18px 16px 8px;
  width: 100%;
}
.ac-stat { display: flex; flex-direction: column; align-items: center; }
.ac-stat-val { color: #fff; font-size: 18px; font-weight: bold; }
.ac-stat-label { color: #888; font-size: 11px; margin-top: 2px; }
.ac-divider { width: 1px; height: 22px; background: #333; }

/* Bio */
.ac-bio {
  color: #ccc;
  font-size: 13px;
  line-height: 20px;
  text-align: center;
  padding: 8px 28px 18px;
}

/* Action buttons */
.ac-actions { display: flex; flex-direction: column; gap: 10px; width: 100%; padding: 0 28px; box-sizing: border-box; }
/* ===================== Feature: 主播开播提醒 (schedule reminder) ===================== */
.ac-reminder-block { display: flex; flex-direction: column; align-items: center; gap: 4px; width: 100%; }
.ac-next-live {
  color: #fe2c55;
  font-size: 12px;
  font-weight: 500;
  text-align: center;
}
.ac-btn { flex: 1; }
.ac-btn-outline {
  background: transparent;
  border: 1px solid #444;
  color: #fff;
}
.ac-btn-outline :deep(.van-button__text) { color: #fff; }

.ac-loading { padding: 60px; }

/* ===================== Feature: 直播间热度计 (heat meter) ===================== */
/* Vertical thermometer gauge fixed to the right edge, vertically centered. Sits
   above the action rail and below the entry buttons. */
.heat-meter {
  position: absolute;
  right: 14px;
  top: 48%;
  transform: translateY(-50%);
  z-index: 11;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  pointer-events: none;
}
/* Round bulb at the bottom of the thermometer. */
.heat-bulb {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid rgba(255,255,255,0.6);
  box-shadow: 0 0 10px rgba(0,0,0,0.5);
  order: 3;
  animation: heatBulbPulse 1.4s ease-in-out infinite;
}
@keyframes heatBulbPulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.12); }
}
/* The glass tube holding the fill. */
.heat-tube {
  position: relative;
  width: 14px;
  height: 120px;
  background: rgba(0,0,0,0.55);
  border: 2px solid rgba(255,255,255,0.5);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 0 8px rgba(0,0,0,0.4);
  order: 2;
}
/* Fill column — height driven by viewer count, color by tier. Grows smoothly. */
.heat-fill {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  border-radius: 0 0 5px 5px;
  transition: height 0.8s ease, background 0.8s ease;
  overflow: hidden;
}
/* Bubbling effect — small circles rising inside the fill at the top. */
.heat-bubble {
  position: absolute;
  left: 50%;
  bottom: 0;
  width: 5px;
  height: 5px;
  background: rgba(255,255,255,0.7);
  border-radius: 50%;
  transform: translateX(-50%);
  animation: heatBubble 1.6s ease-in infinite;
}
.heat-bubble-1 { animation-delay: 0s; }
.heat-bubble-2 { animation-delay: 0.5s; }
.heat-bubble-3 { animation-delay: 1s; }
@keyframes heatBubble {
  0%   { bottom: 0; opacity: 0; transform: translateX(-50%) scale(0.6); }
  20%  { opacity: 0.9; }
  100% { bottom: 100%; opacity: 0; transform: translateX(-50%) scale(1); }
}
/* Label + value under the gauge. */
.heat-label {
  order: 1;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  text-shadow: 0 1px 3px rgba(0,0,0,0.6);
  white-space: nowrap;
}
.heat-value {
  order: 4;
  font-size: 11px;
  font-weight: bold;
  color: #fff;
  font-variant-numeric: tabular-nums;
  text-shadow: 0 1px 3px rgba(0,0,0,0.7);
}
/* Tier-tinted glow on the tube border for a little extra flair. */
.heat-meter.heat-cold .heat-tube { box-shadow: 0 0 10px rgba(79,172,254,0.5); }
.heat-meter.heat-warm .heat-tube { box-shadow: 0 0 10px rgba(255,149,0,0.5); }
.heat-meter.heat-hot .heat-tube { box-shadow: 0 0 10px rgba(255,59,48,0.55); }
.heat-meter.heat-boom .heat-tube { box-shadow: 0 0 12px rgba(156,39,176,0.6); }
</style>
