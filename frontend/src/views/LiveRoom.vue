<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getLiveRoom, likeLive, sendLiveMessage, getLiveGifts, startPK, getActivePK, scorePK, guardHost, getGuardStatus, dropRedPacket, getActiveRedPacket, grabRedPacket, getContributors, contribute, banUser, getSuggestFollows, getUser, toggleFollow } from '../api'

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
  pollTimer = setInterval(pollMessages, 3000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  // Clean up dice game timers to avoid leaks when leaving the room.
  if (diceSpinTimer) clearInterval(diceSpinTimer)
  if (diceStopTimer) clearTimeout(diceStopTimer)
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
  try { contributors.value = await getContributors(route.params.id) } catch (e) { contributors.value = [] }
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

// ===================== Feature: Live viewer list (直播间观众列表) =====================
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

    <!-- ===================== Feature: 直播间主题装扮 (theme picker) ===================== -->
    <div class="theme-entry" @click="showTheme = true">
      <span class="theme-swatch" :style="{ background: themeAccent }"></span>
      🎨 主题
    </div>
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
        <div v-for="(c, i) in contributors" :key="c.user_id" class="cp-item">
          <span class="cp-rank" :class="{ top: i < 3 }">{{ i + 1 }}</span>
          <img class="cp-avatar" :src="c.avatar" />
          <span class="cp-name">
            {{ c.nickname }}
            <!-- Feature: 粉丝勋章等级配色 — tier-colored pill next to the name -->
            <span class="fan-badge" :class="fanTier(c.amount).cls">{{ fanTier(c.amount).label }}</span>
          </span>
          <span class="cp-amount">{{ c.amount }}</span>
        </div>
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
.ac-btn { flex: 1; }
.ac-btn-outline {
  background: transparent;
  border: 1px solid #444;
  color: #fff;
}
.ac-btn-outline :deep(.van-button__text) { color: #fff; }

.ac-loading { padding: 60px; }
</style>
