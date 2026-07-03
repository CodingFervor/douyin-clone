<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getLiveRoom, likeLive, sendLiveMessage, getLiveGifts, startPK, getActivePK, scorePK, guardHost, getGuardStatus, dropRedPacket, getActiveRedPacket, grabRedPacket } from '../api'

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
    // Refresh remaining.
    redPacket.value = await getActiveRedPacket(route.params.id)
    if (!redPacket.value) fallingPackets.value = []
  } catch (e) {
    showToast(e.response?.data?.error || '手慢了')
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

    <!-- Red packet drop button + rain -->
    <div v-if="!redPacket" class="rp-drop" @click="doDropPacket">🧧 发红包</div>
    <div v-if="redPacket" class="rp-banner" @click="grab">
      🧧 红包雨进行中 · 剩余 {{ redPacket.remaining }}/{{ redPacket.total }} · 点击抢
    </div>
    <div class="rp-rain">
      <div v-for="f in fallingPackets" :key="f.id" class="rp-fall" :style="{ left: f.left + '%' }">🧧</div>
    </div>

    <!-- Danmaku / chat list -->
    <div class="danmaku-layer" ref="msgListRef">
      <div v-for="m in messages" :key="m.id" class="dm-item">
        <span class="dm-user">{{ m.username }}:</span>
        <span class="dm-text">{{ m.content }}</span>
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
