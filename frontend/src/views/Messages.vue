<script setup>
// Messages stub — notification center placeholders.
const items = [
  { icon: 'envelop-o', label: '粉丝', count: 0, color: '#fe2c55' },
  { icon: 'comment-o', label: '评论', count: 0, color: '#25f4ee' },
  { icon: 'like-o', label: '赞', count: 0, color: '#ffc107' },
  { icon: 'newspaper-o', label: '系统通知', count: 0, color: '#1989fa' },
]
const chats = [
  { name: '旅行的意义', msg: '你的视频太棒了！', time: '10:30' },
  { name: '吃货日记', msg: '求这家店地址～', time: '昨天' },
]
</script>

<template>
  <div class="msg-page">
    <van-nav-bar title="消息" />
    <div class="msg-grid">
      <div v-for="it in items" :key="it.label" class="msg-item">
        <div class="mi-icon" :style="{ background: it.color }"><van-icon :name="it.icon" size="22" color="#fff" /></div>
        <span>{{ it.label }}</span>
      </div>
    </div>
    <div class="chat-list">
      <div class="cl-head">私信</div>
      <div v-for="c in chats" :key="c.name" class="chat-item">
        <van-icon name="user-o" size="40" color="#666" />
        <div class="ci-info"><div class="ci-name">{{ c.name }}</div><div class="ci-msg van-ellipsis">{{ c.msg }}</div></div>
        <span class="ci-time">{{ c.time }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.msg-page { height: 100vh; overflow-y: auto; background: #000; }
.msg-grid { display: grid; grid-template-columns: repeat(4, 1fr); padding: 20px 0; background: #161616; }
.msg-item { display: flex; flex-direction: column; align-items: center; gap: 6px; color: #fff; font-size: 12px; }
.mi-icon { width: 50px; height: 50px; border-radius: 50%; display: flex; align-items: center; justify-content: center; }
.chat-list { margin-top: 10px; background: #161616; }
.cl-head { color: #888; font-size: 13px; padding: 12px 16px; border-bottom: 1px solid #222; }
.chat-item { display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-bottom: 1px solid #1a1a1a; }
.ci-info { flex: 1; }
.ci-name { color: #fff; font-size: 14px; }
.ci-msg { color: #888; font-size: 12px; margin-top: 2px; }
.ci-time { color: #666; font-size: 11px; }
</style>
