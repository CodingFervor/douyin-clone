<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getNotifications, getNotificationCounts, markNotificationsRead } from '../api'
import { setUnreadTotal } from '../utils/notifyStore'

const router = useRouter()
const loggedIn = ref(false)
const counts = ref({ like: 0, comment: 0, follow: 0, system: 0 })
const activeType = ref('')
const list = ref([])
const loading = ref(false)

const tabs = [
  { key: 'like', icon: 'like-o', label: '赞', color: '#fe2c55' },
  { key: 'comment', icon: 'comment-o', label: '评论', color: '#25f4ee' },
  { key: 'follow', icon: 'friends-o', label: '新粉丝', color: '#ffc107' },
  { key: 'system', icon: 'envelop-o', label: '系统', color: '#1989fa' },
]

// Total unread across all types — drives the App tab bar badge.
const unreadTotal = computed(() =>
  Object.values(counts.value).reduce((a, b) => a + (Number(b) || 0), 0)
)
function syncBadge() {
  setUnreadTotal(unreadTotal.value)
}

async function load() {
  loggedIn.value = !!localStorage.getItem('dy_token')
  if (!loggedIn.value) { setUnreadTotal(0); return }
  try {
    counts.value = await getNotificationCounts()
    syncBadge()
    // Feature: 消息通知音效 — beep if the toggle is on and new notifications arrived.
    maybePlayNotifySound()
    await loadList(activeType.value)
  } catch (e) {
    // silent
  }
}

async function loadList(type) {
  activeType.value = type
  loading.value = true
  try {
    list.value = await getNotifications(type || undefined)
  } catch (e) {
    list.value = []
  } finally {
    loading.value = false
  }
}

async function readAll() {
  try {
    await markNotificationsRead()
    counts.value = { like: 0, comment: 0, follow: 0, system: 0 }
    list.value = list.value.map((n) => ({ ...n, is_read: 1 }))
    syncBadge()
    showSuccessToast('已全部已读')
  } catch (e) {
    showToast('操作失败')
  }
}

// ===================== Feature: 一键已读 (mark visible as read) =====================
// Marks the currently visible list items as read without a network round-trip
// affecting the counts object beyond what readAll already covers. Mirrors
// readAll behaviour but is triggered by the floating action button.
async function quickRead() {
  if (!list.value.some((n) => !n.is_read)) { showToast('没有未读消息'); return }
  try {
    await markNotificationsRead()
    counts.value = { like: 0, comment: 0, follow: 0, system: 0 }
    list.value = list.value.map((n) => ({ ...n, is_read: 1 }))
    syncBadge()
    showSuccessToast('一键已读')
  } catch (e) {
    showToast('操作失败')
  }
}

// ===================== Feature: swipe-to-delete =====================
// Removes a notification from the local list (client-side) when the user
// swipes left and taps delete.
function removeNotify(n) {
  list.value = list.value.filter((x) => x.id !== n.id)
  // If it was unread, decrement the badge so counts stay consistent.
  if (!n.is_read && counts.value[n.type] != null) {
    counts.value[n.type] = Math.max(0, (counts.value[n.type] || 0) - 1)
    syncBadge()
  }
  showToast('已删除')
}

function descOf(n) {
  if (n.type === 'like') return '赞了你的作品'
  if (n.type === 'comment') return n.content
  if (n.type === 'follow') return '关注了你'
  return n.content
}
function timeOf(t) {
  try {
    const d = new Date(t.replace(' ', 'T') + 'Z')
    const diff = (Date.now() - d.getTime()) / 1000
    if (diff < 60) return '刚刚'
    if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
    if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
    return Math.floor(diff / 86400) + '天前'
  } catch {
    return ''
  }
}

// ===================== Feature: 按日期分组 (group by date) =====================
// Buckets the current list into 今天 / 昨天 / 更早 so the UI can render
// section headers. Falls back gracefully when created_at is missing.
function dateBucket(t) {
  if (!t) return '更早'
  const d = new Date(t.replace(' ', 'T') + 'Z')
  if (isNaN(d.getTime())) return '更早'
  const now = new Date()
  const startOfToday = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()))
  const dayMs = 86400000
  const diffDays = Math.floor((startOfToday.getTime() - d.getTime()) / dayMs)
  if (diffDays <= 0) return '今天'
  if (diffDays === 1) return '昨天'
  return '更早'
}

const grouped = computed(() => {
  const order = ['今天', '昨天', '更早']
  const map = { 今天: [], 昨天: [], 更早: [] }
  for (const n of filteredList.value) {
    const b = dateBucket(n.created_at)
    ;(map[b] || (map[b] = [])).push(n)
  }
  return order
    .filter((k) => map[k] && map[k].length)
    .map((k) => ({ label: k, items: map[k] }))
})

// ===================== Feature: 日期筛选 (date range filter) =====================
// A dropdown next to the list header that filters the notifications by age.
// Options: 全部 (no filter) / 今天 (<= 1 day) / 三天内 (<= 3 days). The chosen
// range applies on top of the existing type filter, so it only narrows the
// currently-loaded list. 'all' shows everything (including future-dated items);
// 'today' and 'three' compute the cutoff from the start of the current UTC day.
const DATE_FILTERS = [
  { key: 'all', label: '全部' },
  { key: 'today', label: '今天' },
  { key: 'three', label: '三天内' },
]
const dateFilter = ref('all')
const showDateFilter = ref(false) // dropdown menu visibility

// cutoffMsFor returns the timestamp (ms) before which a notification is filtered
// out for the given range. Returns 0 for 'all' so nothing is excluded.
function cutoffMsFor(key) {
  if (key === 'all') return 0
  const days = key === 'today' ? 1 : 3
  const now = new Date()
  const startOfToday = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())
  return startOfToday - days * 86400000 + 1
}

// filteredList narrows the loaded list by the selected date range. Items without
// a parseable created_at are kept only under the 'all' range.
const filteredList = computed(() => {
  const cutoff = cutoffMsFor(dateFilter.value)
  if (!cutoff) return list.value
  return list.value.filter((n) => {
    if (!n.created_at) return false
    const d = new Date(String(n.created_at).replace(' ', 'T') + 'Z')
    const t = d.getTime()
    if (isNaN(t)) return false
    return t >= cutoff
  })
})

// pickDateFilter applies a range from the dropdown, closes the menu, and toasts
// the selection (except 'all' which is the silent default).
function pickDateFilter(key) {
  dateFilter.value = key
  showDateFilter.value = false
  if (key !== 'all') {
    const opt = DATE_FILTERS.find((f) => f.key === key)
    if (opt) showToast('已筛选: ' + opt.label)
  }
}

// ===================== Feature: Notification sound (消息通知音效) =====================
// A 🔔 toggle in the nav bar enables/disables a short beep (Web Audio API) that
// plays when new notifications are detected on load. The preference persists in
// localStorage 'dy_notify_sound' (default off, since autoplaying audio is
// intrusive). We track the total unread count we've already seen so the beep
// only fires when the count *increases*, not on the first load.
const NOTIFY_SOUND_KEY = 'dy_notify_sound'
const soundEnabled = ref(false)
let lastSeenUnread = null    // total unread count we've already beeped for
let audioCtx = null          // lazily-created Web Audio context

function loadNotifySoundPref() {
  try {
    soundEnabled.value = localStorage.getItem(NOTIFY_SOUND_KEY) === '1'
  } catch (e) {
    soundEnabled.value = false
  }
}

function toggleNotifySound() {
  soundEnabled.value = !soundEnabled.value
  try {
    localStorage.setItem(NOTIFY_SOUND_KEY, soundEnabled.value ? '1' : '0')
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
  if (soundEnabled.value) {
    // Play a sample beep immediately so the user knows what it sounds like, and
    // to satisfy the user-gesture requirement for AudioContext creation.
    playNotifyBeep()
    showToast('已开启通知声音')
  } else {
    showToast('已关闭通知声音')
  }
}

// playNotifyBeep plays a short two-tone ascending beep via the Web Audio API.
// The AudioContext is created lazily on first use (and resumed if suspended).
function playNotifyBeep() {
  try {
    if (!audioCtx) {
      const AC = window.AudioContext || window.webkitAudioContext
      if (!AC) return
      audioCtx = new AC()
    }
    if (audioCtx.state === 'suspended') audioCtx.resume()
    const now = audioCtx.currentTime
    // Two short oscillator beeps (880Hz then 1320Hz) for a pleasant "ding-dong".
    const tones = [{ f: 880, t: 0 }, { f: 1320, t: 0.12 }]
    tones.forEach(({ f, t }) => {
      const osc = audioCtx.createOscillator()
      const gain = audioCtx.createGain()
      osc.type = 'sine'
      osc.frequency.value = f
      // Envelope: quick attack, gentle decay, no click.
      gain.gain.setValueAtTime(0, now + t)
      gain.gain.linearRampToValueAtTime(0.18, now + t + 0.02)
      gain.gain.exponentialRampToValueAtTime(0.001, now + t + 0.22)
      osc.connect(gain)
      gain.connect(audioCtx.destination)
      osc.start(now + t)
      osc.stop(now + t + 0.24)
    })
  } catch (e) {
    // Web Audio may be unavailable (e.g. older browsers) — fail silently.
  }
}

// maybePlayNotifySound fires the beep if the toggle is on AND the total unread
// count increased since the last check. On the very first load we just record
// the baseline so we don't beep for already-existing notifications.
function maybePlayNotifySound() {
  if (!soundEnabled.value) {
    lastSeenUnread = unreadTotal.value
    return
  }
  const total = unreadTotal.value
  if (lastSeenUnread === null) {
    lastSeenUnread = total
    return
  }
  if (total > lastSeenUnread) {
    playNotifyBeep()
  }
  lastSeenUnread = total
}

// Restore the saved sound preference on mount, before the first load() runs.
loadNotifySoundPref()

onMounted(load)
onActivated(load)
</script>

<template>
  <div class="msg-page">
    <van-nav-bar title="消息">
      <template #right>
        <!-- ===================== Feature: 消息通知音效 (notification sound) =====================
             🔔 toggle enables/disables a Web Audio beep when new notifications load.
             The preference persists in localStorage 'dy_notify_sound'. -->
        <span
          v-if="loggedIn"
          class="notify-sound-btn"
          :class="{ on: soundEnabled }"
          @click="toggleNotifySound"
        >{{ soundEnabled ? '🔔' : '🔕' }}</span>
        <span v-if="loggedIn" style="color: #fe2c55; font-size: 13px" @click="readAll">全部已读</span>
      </template>
    </van-nav-bar>
    <div v-if="!loggedIn" class="login-hint">
      <van-icon name="comment-o" size="48" color="#333" />
      <p>登录后查看消息</p>
      <van-button round color="#fe2c55" @click="router.push('/login')">去登录</van-button>
    </div>
    <div v-else>
      <div class="msg-grid">
        <div v-for="t in tabs" :key="t.key" class="msg-item" @click="loadList(t.key)">
          <div class="mi-icon" :style="{ background: t.color }">
            <van-icon :name="t.icon" size="22" color="#fff" />
            <span v-if="counts[t.key]" class="badge">{{ counts[t.key] }}</span>
          </div>
          <span>{{ t.label }}</span>
        </div>
      </div>
      <div class="list-head">
        <span v-if="activeType">{{ tabs.find((t) => t.key === activeType)?.label }}通知</span>
        <span v-else>全部消息</span>
        <span class="list-head-right">
          <span class="filter-all" v-if="activeType" @click="loadList('')">查看全部 ›</span>
          <!-- ===================== Feature: 日期筛选 (date range filter) =====================
               Dropdown trigger. Clicking toggles the menu; the active label updates. -->
          <span class="date-filter" @click="showDateFilter = !showDateFilter">
            <span class="df-label">{{ DATE_FILTERS.find((f) => f.key === dateFilter)?.label }}</span>
            <van-icon name="arrow-down" size="10" />
          </span>
        </span>
      </div>
      <!-- ===================== Feature: 日期筛选 (date range filter) =====================
           Dropdown menu anchored under the list header. One option per range. -->
      <div v-if="showDateFilter" class="date-filter-menu" @click.self="showDateFilter = false">
        <div class="df-menu-inner">
          <div
            v-for="f in DATE_FILTERS"
            :key="f.key"
            class="df-opt"
            :class="{ active: dateFilter === f.key }"
            @click="pickDateFilter(f.key)"
          >{{ f.label }}</div>
        </div>
      </div>
      <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
      <div v-else-if="!filteredList.length" class="empty">
        <van-icon name="comment-o" size="40" color="#333" />
        <p>暂无消息</p>
      </div>
      <div v-else class="notify-list">
        <!-- ===================== Feature: 按日期分组 (group by date) ===================== -->
        <div v-for="g in grouped" :key="g.label" class="date-group">
          <div class="date-head">{{ g.label }}</div>
          <!-- ===================== Feature: swipe-to-delete ===================== -->
          <van-swipe-cell v-for="n in g.items" :key="n.id">
            <div class="notify-item" :class="{ unread: !n.is_read }">
              <img class="n-avatar" :src="n.actor_avatar || 'https://via.placeholder.com/40'" />
              <div class="n-body">
                <div class="n-user">{{ n.actor_name }} <small>{{ descOf(n) }}</small></div>
                <div class="n-time">{{ timeOf(n.created_at) }}</div>
              </div>
              <van-icon v-if="n.type === 'like'" name="like" color="#fe2c55" size="20" />
              <van-icon v-else-if="n.type === 'comment'" name="comment-o" color="#25f4ee" size="20" />
              <van-icon v-else-if="n.type === 'follow'" name="friends-o" color="#ffc107" size="20" />
            </div>
            <template #right>
              <van-button square type="danger" text="删除" class="del-btn" @click="removeNotify(n)" />
            </template>
          </van-swipe-cell>
        </div>
      </div>
    </div>

    <!-- ===================== Feature: 一键已读 floating action button ===================== -->
    <div v-if="loggedIn && list.length" class="fab-read" @click="quickRead">
      <van-icon name="success" size="22" color="#fff" />
      <span>一键已读</span>
    </div>
  </div>
</template>

<style scoped>
.msg-page { height: 100vh; overflow-y: auto; background: #000; position: relative; }
.login-hint, .empty { text-align: center; padding: 80px 20px; color: #666; }
.login-hint p, .empty p { margin: 12px 0 16px; font-size: 14px; }
.msg-grid { display: grid; grid-template-columns: repeat(4, 1fr); padding: 20px 0; background: #161616; }
.msg-item { display: flex; flex-direction: column; align-items: center; gap: 6px; color: #fff; font-size: 12px; }
.mi-icon { width: 50px; height: 50px; border-radius: 50%; display: flex; align-items: center; justify-content: center; position: relative; }
.badge { position: absolute; top: -2px; right: -2px; background: #fff; color: #fe2c55; font-size: 10px; min-width: 16px; height: 16px; line-height: 16px; border-radius: 8px; padding: 0 4px; }
.list-head { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; color: #888; font-size: 13px; background: #161616; }
.list-head-right { display: flex; align-items: center; gap: 12px; }
.filter-all { color: #fe2c55; }

/* ===================== Feature: 日期筛选 (date range filter) =====================
   A small inline dropdown trigger in the list header. The menu is a
   position:absolute card anchored under the trigger; the overlay div closes it. */
.date-filter {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: #25f4ee;
  font-size: 12px;
  padding: 3px 9px;
  border: 1px solid rgba(37, 244, 238, 0.4);
  border-radius: 12px;
  background: rgba(37, 244, 238, 0.1);
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, border-color 0.15s;
}
.date-filter:active { background: rgba(37, 244, 238, 0.25); }
.df-label { white-space: nowrap; }
.date-filter-menu {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 30;
  background: rgba(0, 0, 0, 0.3);
}
.df-menu-inner {
  position: absolute;
  top: 92px;
  right: 14px;
  min-width: 120px;
  background: #1c1c1c;
  border: 1px solid #2a2a2a;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
}
.df-opt {
  padding: 11px 18px;
  color: #ddd;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  border-bottom: 1px solid #232323;
  transition: background 0.15s;
}
.df-opt:last-child { border-bottom: none; }
.df-opt:active { background: #262626; }
.df-opt.active { color: #fe2c55; font-weight: 600; }
.loading { text-align: center; padding: 40px; }
.date-group { background: #000; }
.date-head { color: #999; font-size: 12px; padding: 10px 16px 4px; background: #000; }
.notify-item { display: flex; align-items: center; gap: 10px; padding: 12px 16px; border-bottom: 1px solid #1a1a1a; background: #000; }
.notify-item.unread { background: #1a0a0a; }
.n-avatar { width: 40px; height: 40px; border-radius: 50%; flex-shrink: 0; }
.n-body { flex: 1; min-width: 0; }
.n-user { color: #fff; font-size: 14px; }
.n-user small { color: #999; font-size: 13px; margin-left: 4px; }
.n-time { color: #555; font-size: 11px; margin-top: 2px; }
.del-btn { height: 100%; }

/* ===================== Feature: 一键已读 floating action button ===================== */
.fab-read {
  position: fixed; bottom: 80px; right: 16px; z-index: 20;
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  background: linear-gradient(135deg, #fe2c55, #ff6b9d);
  color: #fff; font-size: 11px; font-weight: bold;
  padding: 10px 12px; border-radius: 28px;
  box-shadow: 0 4px 16px rgba(254,44,85,0.45); cursor: pointer;
  transition: transform 0.15s ease;
}
.fab-read:active { transform: scale(0.92); }

/* ===================== Feature: 消息通知音效 (notification sound) =====================
   The 🔔/🔕 toggle in the nav bar. When on it takes the theme accent color and a
   subtle pulse so the user can see at a glance that sound is enabled. */
.notify-sound-btn {
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  user-select: none;
  margin-right: 14px;
  filter: grayscale(0.6);
  opacity: 0.7;
  transition: opacity 0.15s, filter 0.15s;
}
.notify-sound-btn.on {
  filter: none;
  opacity: 1;
  animation: notifyBellRing 1.8s ease-in-out infinite;
}
@keyframes notifyBellRing {
  0%, 70%, 100% { transform: rotate(0); }
  75% { transform: rotate(12deg); }
  80% { transform: rotate(-10deg); }
  85% { transform: rotate(6deg); }
  90% { transform: rotate(-4deg); }
}
</style>
