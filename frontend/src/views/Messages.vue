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
  for (const n of list.value) {
    const b = dateBucket(n.created_at)
    ;(map[b] || (map[b] = [])).push(n)
  }
  return order
    .filter((k) => map[k] && map[k].length)
    .map((k) => ({ label: k, items: map[k] }))
})

onMounted(load)
onActivated(load)
</script>

<template>
  <div class="msg-page">
    <van-nav-bar title="消息">
      <template #right>
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
        <span class="filter-all" v-if="activeType" @click="loadList('')">查看全部 ›</span>
      </div>
      <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
      <div v-else-if="!list.length" class="empty">
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
.filter-all { color: #fe2c55; }
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
</style>
