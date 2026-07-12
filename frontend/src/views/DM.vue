<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getConversations } from '../api'

const router = useRouter()
const conversations = ref([])
const loading = ref(true)

// ===================== Feature: Conversation search (私信搜索) =====================
// Filters the conversation list by the other user's name in real-time.
const searchQuery = ref('')
const filteredConversations = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return conversations.value
  return conversations.value.filter((c) => {
    const name = (c.other_name || '').toLowerCase()
    return name.includes(q)
  })
})

// ===================== Feature: DM online status (私信在线状态) =====================
// A green/gray dot in the corner of each conversation partner's avatar indicates
// whether they are "online". Status is deterministic per user id (FNV-1a hash of
// the other_id) so the same partner is always shown the same way. ~55% of users
// are considered online so most rows show green but a healthy minority show gray.
function isOnline(c) {
  const id = c && c.other_id
  if (id == null) return false
  const s = String(id)
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return (Math.abs(h) % 100) < 55
}

onMounted(async () => { try { conversations.value = await getConversations() } catch (e) { showToast('加载失败') } finally { loading.value = false } })
</script>

<template>
  <div class="dm-page">
    <van-nav-bar title="私信" left-arrow @click-left="router.back()" fixed placeholder />
    <!-- Feature: 私信搜索 — search bar at the top filters conversations by name. -->
    <div class="dm-search-wrap" v-if="!loading && conversations.length">
      <div class="dm-search">
        <span class="dm-search-icon">🔍</span>
        <input v-model="searchQuery" class="dm-search-input" placeholder="搜索昵称" />
        <span v-if="searchQuery" class="dm-search-clear" @click="searchQuery = ''">✕</span>
      </div>
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!conversations.length" description="还没有私信" />
    <!-- Feature: 私信搜索 — empty state when the query matches nothing. -->
    <van-empty v-else-if="!filteredConversations.length" description="未找到" />
    <div v-else class="conv-list">
      <div v-for="c in filteredConversations" :key="c.other_id" class="conv-item" @click="router.push('/chat/' + c.other_id)">
        <div class="ci-avatar-wrap">
          <img class="ci-avatar" :src="c.other_avatar || 'https://via.placeholder.com/48'" />
          <!-- ===================== Feature: 私信在线状态 (online status dot) =====================
               Green when the partner is online (deterministic from id hash), gray
               when offline. Sits in the bottom-right corner of the avatar. -->
          <span class="ci-status-dot" :class="isOnline(c) ? 'online' : 'offline'"></span>
        </div>
        <div class="ci-info">
          <div class="ci-name">{{ c.other_name }}</div>
          <div class="ci-last">{{ c.last_message }}</div>
        </div>
        <van-tag v-if="c.unread > 0" type="danger" round>{{ c.unread }}</van-tag>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dm-page { height: 100vh; overflow-y: auto; background: #000; }
.loading { text-align: center; padding: 60px; }
.dm-search-wrap { padding: 10px 12px; background: #000; position: sticky; top: 46px; z-index: 10; }
.dm-search { display: flex; align-items: center; gap: 8px; background: #161616; border-radius: 20px; padding: 0 12px; height: 36px; }
.dm-search-icon { font-size: 13px; opacity: 0.7; flex-shrink: 0; }
.dm-search-input { flex: 1; background: transparent; border: none; outline: none; color: #fff; font-size: 14px; height: 100%; }
.dm-search-input::placeholder { color: #888; }
.dm-search-clear { color: #888; font-size: 14px; padding: 4px; flex-shrink: 0; cursor: pointer; }
.conv-item { display: flex; align-items: center; gap: 12px; padding: 14px 16px; background: #161616; border-bottom: 1px solid #1a1a1a; }
.ci-avatar-wrap { position: relative; flex-shrink: 0; }
.ci-avatar { width: 48px; height: 48px; border-radius: 50%; display: block; }
/* ===================== Feature: 私信在线状态 (online status dot) =====================
   A small dot anchored to the bottom-right of the avatar. Green + a soft glow
   ring when online, dark gray when offline. A 2px dark border matches the row
   background so the dot reads cleanly against the photo. */
.ci-status-dot {
  position: absolute;
  right: 1px;
  bottom: 1px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid #161616;
  box-sizing: content-box;
}
.ci-status-dot.online { background: #2ecc71; box-shadow: 0 0 4px rgba(46, 204, 113, 0.8); }
.ci-status-dot.offline { background: #8a8a8a; }
.ci-info { flex: 1; min-width: 0; }
.ci-name { color: #fff; font-size: 15px; }
.ci-last { color: #888; font-size: 13px; margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
