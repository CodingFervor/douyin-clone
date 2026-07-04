<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getConversations } from '../api'

const router = useRouter()
const conversations = ref([])
const loading = ref(true)

onMounted(async () => { try { conversations.value = await getConversations() } catch (e) { showToast('加载失败') } finally { loading.value = false } })
</script>

<template>
  <div class="dm-page">
    <van-nav-bar title="私信" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!conversations.length" description="还没有私信" />
    <div v-else class="conv-list">
      <div v-for="c in conversations" :key="c.other_id" class="conv-item" @click="router.push('/chat/' + c.other_id)">
        <img class="ci-avatar" :src="c.other_avatar || 'https://via.placeholder.com/48'" />
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
.conv-item { display: flex; align-items: center; gap: 12px; padding: 14px 16px; background: #161616; border-bottom: 1px solid #1a1a1a; }
.ci-avatar { width: 48px; height: 48px; border-radius: 50%; }
.ci-info { flex: 1; min-width: 0; }
.ci-name { color: #fff; font-size: 15px; }
.ci-last { color: #888; font-size: 13px; margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
