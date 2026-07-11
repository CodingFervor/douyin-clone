<script setup>
import { ref, reactive, onMounted, nextTick } from 'vue'
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

// ===================== Feature: Chat message reactions (聊天消息表情) =====================
// Long-pressing a message reveals a small reaction bar (👍❤️😂). Tapping an
// emoji adds it as a small badge below the message; tapping it again removes it
// (toggle). Reactions are tracked per message id in a reactive object so counts
// are shown when a message has multiple different reactions.
const REACTION_EMOJIS = ['👍', '❤️', '😂']
// reactions: { [messageId]: { [emoji]: count } }
const reactions = reactive({})
// The message id whose reaction bar is currently open (null when none).
const openReactionId = ref(null)
// Long-press detection state.
let pressTimer = null
const LONG_PRESS_MS = 450

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

// ---- Reaction bar (long-press) ----
// armReaction starts the long-press timer that opens the reaction bar. If the
// finger lifts (or the pointer leaves) before the timer fires, the press is
// cancelled — so a normal tap never opens the bar.
function armReaction(m, e) {
  cancelReactionPress()
  if (e && e.type === 'mousedown' && e.button !== 0) return // only left button
  pressTimer = setTimeout(() => {
    pressTimer = null
    openReactionId.value = m.id
  }, LONG_PRESS_MS)
}
function cancelReactionPress() {
  if (pressTimer) {
    clearTimeout(pressTimer)
    pressTimer = null
  }
}
// closeReactionBar hides the open reaction bar (e.g. tapping elsewhere).
function closeReactionBar() {
  openReactionId.value = null
}

// toggleReaction flips an emoji on/off for a message. Selecting an already-added
// emoji removes it (toggle); selecting a different one adds it. The reactive map
// is mutated directly so Vue tracks the nested changes.
function toggleReaction(m, emoji) {
  if (!reactions[m.id]) reactions[m.id] = {}
  const counts = reactions[m.id]
  if (counts[emoji]) {
    delete counts[emoji]
    // Clean up empty message entries so v-if renders nothing.
    if (Object.keys(counts).length === 0) delete reactions[m.id]
  } else {
    counts[emoji] = (counts[emoji] || 0) + 1
  }
  // Hide the reaction bar after a pick.
  openReactionId.value = null
}

// reactionEntries returns the [emoji, count] pairs for a message (for rendering
// the badges below the bubble). Returns [] when the message has no reactions.
function reactionEntries(m) {
  const counts = reactions[m.id]
  if (!counts) return []
  return Object.entries(counts).filter(([, n]) => n > 0)
}
</script>

<template>
  <div class="chat-page" @click="closeReactionBar" @touchstart="closeReactionBar">
    <van-nav-bar title="私信聊天" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="msg-list" ref="listRef">
      <div
        v-for="m in messages"
        :key="m.id"
        class="msg-item"
        :class="{ mine: m.sender_id == otherId ? false : true }"
        @click.stop
        @touchstart.self="closeReactionBar"
      >
        <img v-if="m.sender_id != otherId || true" class="msg-avatar" :src="m.sender_avatar || 'https://via.placeholder.com/36'" />
        <div class="msg-col">
          <div
            class="msg-bubble"
            @click.stop
            @touchstart.passive="armReaction(m, $event)"
            @touchend.passive="cancelReactionPress"
            @touchmove.passive="cancelReactionPress"
            @touchcancel="cancelReactionPress"
            @mousedown="armReaction(m, $event)"
            @mouseup="cancelReactionPress"
            @mouseleave="cancelReactionPress"
            @contextmenu.prevent="openReactionId = m.id"
          >{{ m.content }}</div>

          <!-- ===================== Feature: 聊天消息表情 (message reactions) =====================
               Long-pressing the bubble opens a small reaction bar (👍❤️😂) floating above
               it. Tapping an emoji toggles it as a badge below the bubble. -->
          <transition name="rx-pop">
            <div
              v-if="openReactionId === m.id"
              class="reaction-bar"
              @click.stop
              @touchstart.stop
            >
              <span
                v-for="emoji in REACTION_EMOJIS"
                :key="emoji"
                class="rx-pick"
                :class="{ active: reactions[m.id] && reactions[m.id][emoji] }"
                @click.stop="toggleReaction(m, emoji)"
                @touchstart.stop.prevent="toggleReaction(m, emoji)"
              >{{ emoji }}</span>
            </div>
          </transition>

          <!-- Reaction badges rendered below the bubble. Each emoji shows its count
               when there is more than one reaction on the message. -->
          <div v-if="reactionEntries(m).length" class="reaction-badges">
            <span
              v-for="[emoji, count] in reactionEntries(m)"
              :key="emoji"
              class="rx-badge"
              @click.stop="toggleReaction(m, emoji)"
            >{{ emoji }}<i v-if="reactionEntries(m).length > 1 || count > 1">{{ count }}</i></span>
          </div>
        </div>
      </div>
    </div>
    <div class="chat-input" @click.stop>
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
/* .msg-col holds the bubble + the reaction bar/badges so they align under it. */
.msg-col {
  position: relative;
  display: flex;
  flex-direction: column;
  max-width: 70%;
  min-width: 0;
}
.msg-item.mine .msg-col { align-items: flex-end; }
.msg-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 18px;
  background: #222;
  color: #fff;
  user-select: none;
  cursor: pointer;
}
.msg-item.mine .msg-bubble { background: #fe2c55; }

/* ===================== Feature: 聊天消息表情 (message reactions) ===================== */
/* Floating reaction bar (👍❤️😂) shown above the bubble on long-press. */
.reaction-bar {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #2a2a2a;
  border: 1px solid #333;
  border-radius: 20px;
  padding: 4px 8px;
  margin-bottom: 4px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.5);
  transform-origin: bottom center;
}
.msg-item.mine .reaction-bar { align-self: flex-end; transform-origin: bottom right; }
.rx-pick {
  font-size: 20px;
  line-height: 1;
  padding: 2px 4px;
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.12s, background 0.12s;
  user-select: none;
}
.rx-pick:hover { background: rgba(255,255,255,0.1); }
.rx-pick:active { transform: scale(1.25); }
.rx-pick.active { background: rgba(254,44,85,0.25); }
/* Pop-in animation for the reaction bar. */
.rx-pop-enter-active, .rx-pop-leave-active { transition: opacity 0.15s, transform 0.15s; }
.rx-pop-enter-from, .rx-pop-leave-to { opacity: 0; transform: translateY(6px) scale(0.85); }

/* Reaction badges rendered below the bubble. */
.reaction-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}
.msg-item.mine .reaction-badges { justify-content: flex-end; }
.rx-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 13px;
  background: #1f1f1f;
  border: 1px solid #333;
  color: #fff;
  padding: 1px 7px;
  border-radius: 12px;
  line-height: 18px;
  cursor: pointer;
  user-select: none;
  transition: border-color 0.15s, background 0.15s;
}
.rx-badge:hover { border-color: #fe2c55; }
.rx-badge:active { transform: scale(0.95); }
.rx-badge i {
  font-style: normal;
  font-size: 11px;
  font-weight: 600;
  color: #fe2c55;
}

.chat-input { display: flex; gap: 8px; padding: 10px 16px; background: #161616; border-top: 1px solid #222; }
.chat-input input { flex: 1; background: #222; border: none; outline: none; color: #fff; border-radius: 20px; padding: 0 16px; height: 36px; }
</style>
