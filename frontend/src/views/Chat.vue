<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getConversation, sendDM } from '../api'

const route = useRoute()
const router = useRouter()
const otherId = route.params.userId
const messages = ref([])
const text = ref('')
const listRef = ref(null)
const loading = ref(true)

onMounted(async () => {
  try { messages.value = await getConversation(otherId); await nextTick(); scrollBottom() }
  catch (e) { showToast('加载失败') } finally { loading.value = false }
})

async function send() {
  if (!text.value.trim()) return
  try {
    const m = await sendDM(otherId, text.value)
    messages.value.push(m)
    text.value = ''
    await nextTick()
    scrollBottom()
  } catch (e) { showToast('请先登录'); router.push('/login') }
}
function scrollBottom() { if (listRef.value) listRef.value.scrollTop = listRef.value.scrollHeight }
</script>

<template>
  <div class="chat-page">
    <van-nav-bar title="私信聊天" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="msg-list" ref="listRef">
      <div v-for="m in messages" :key="m.id" class="msg-item" :class="{ mine: m.sender_id == otherId ? false : true }">
        <img v-if="m.sender_id != otherId || true" class="msg-avatar" :src="m.sender_avatar || 'https://via.placeholder.com/36'" />
        <div class="msg-bubble">{{ m.content }}</div>
      </div>
    </div>
    <div class="chat-input">
      <input v-model="text" placeholder="输入消息..." @keyup.enter="send" />
      <van-button size="small" type="primary" color="#fe2c55" @click="send">发送</van-button>
    </div>
  </div>
</template>

<style scoped>
.chat-page { height: 100vh; display: flex; flex-direction: column; background: #000; }
.msg-list { flex: 1; overflow-y: auto; padding: 16px; }
.msg-item { display: flex; align-items: flex-start; gap: 8px; margin-bottom: 12px; }
.msg-item.mine { flex-direction: row-reverse; }
.msg-avatar { width: 36px; height: 36px; border-radius: 50%; flex-shrink: 0; }
.msg-bubble { max-width: 70%; padding: 10px 14px; border-radius: 12px; font-size: 14px; line-height: 18px; background: #222; color: #fff; }
.msg-item.mine .msg-bubble { background: #fe2c55; }
.chat-input { display: flex; gap: 8px; padding: 10px 16px; background: #161616; border-top: 1px solid #222; }
.chat-input input { flex: 1; background: #222; border: none; outline: none; color: #fff; border-radius: 20px; padding: 0 16px; height: 36px; }
</style>
