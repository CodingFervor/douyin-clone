<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getLiveRoom, likeLive, sendLiveMessage, getLiveGifts, startPK, getActivePK, scorePK, guardHost, getGuardStatus, dropRedPacket, getActiveRedPacket, grabRedPacket, getContributors, contribute, banUser, getSuggestFollows } from '../api'

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
  pollTimer = setInterval(pollMessages, 3000)
})

onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })

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

// Send a gift: plays a flying animation + posts a system-style danmaku.
function sendGift(g) {
  const id = Date.now() + Math.random()
  flyingGifts.value.push({ id, icon: g.icon, name: g.name })
  setTimeout(() => {
    flyingGifts.value = flyingGifts.value.filter((f) => f.id !== id)
  }, 2500)
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
</script>

<template>
  <div class="room-page" v-if="loading">
    <div class="loading-center"><van-loading color="#fe2c55" /></div>
  </div>
  <div class="room-page" v-else-if="room">
    <!-- HLS video player -->
    <video class="live-video" :src="room.stream_url" autoplay muted loop playsinline></video>

    <!-- Top bar -->
    <div class="top-bar">
      <van-icon name="arrow-left" size="22" color="#fff" @click="router.back()" />
      <div class="host-info">
        <img class="host-avatar" :src="room.host_avatar" />
        <div>
          <div class="host-name">{{ room.host_name }}</div>
          <div class="host-viewers">{{ fmt(room.viewers) }}观看</div>
        </div>
      </div>
      <van-button size="mini" round color="#fe2c55" @click="showToast('关注成功')">+ 关注</van-button>
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

    <!-- Guard count -->
    <div v-if="guardCount > 0" class="guard-info">🛡️ {{ guardCount }}人守护</div>
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
          <span class="cp-name">{{ c.nickname }}</span>
          <span class="cp-amount">{{ c.amount }}</span>
        </div>
        <van-button block round color="#fe2c55" style="margin-top: 16px" @click="doContribute">为TA打榜 +10</van-button>
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
        <van-icon name="like" color="#fe2c55" size="32" />
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

    <!-- Bottom: chat input + product teaser (小黄车入口) -->
    <div class="bottom-bar">
      <div class="chat-input">
        <input v-model="msgText" placeholder="说点什么..." @keyup.enter="sendMessage" />
        <van-icon name="smile-comment-o" color="#fe2c55" size="22" @click="sendMessage" />
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
  </div>
  <div v-else class="room-page"><van-empty description="直播间不存在" /></div>
</template>

<style scoped>
.room-page { height: 100vh; background: #000; position: relative; overflow: hidden; }
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
.gift-panel { background: #161616; height: 100%; display: flex; flex-direction: column; }
.gp-head { text-align: center; padding: 14px; color: #fff; font-size: 15px; border-bottom: 1px solid #222; }
.gp-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; padding: 12px; overflow-y: auto; }
.gp-item { display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 8px 4px; border-radius: 8px; }
.gp-item:active { background: #222; }
.gp-icon { font-size: 32px; }
.gp-name { color: #fff; font-size: 12px; }
.gp-price { color: #ffd700; font-size: 11px; }
</style>
