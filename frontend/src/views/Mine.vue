<script setup>
import { ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile, getUserVideos, getFavoriteVideos } from '../api'

const router = useRouter()
const user = ref(null)
const videos = ref([])
const favVideos = ref([])
const tab = ref('works')
const loggedIn = ref(false)

async function load() {
  loggedIn.value = !!localStorage.getItem('dy_token')
  if (!loggedIn.value) return
  try {
    user.value = await getProfile()
    videos.value = await getUserVideos(user.value.id)
    favVideos.value = await getFavoriteVideos()
  } catch (e) {
    loggedIn.value = false
  }
}
onMounted(load); onActivated(load)

function logout() {
  localStorage.removeItem('dy_token')
  loggedIn.value = false
  user.value = null
  showToast('已退出登录')
}
function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
</script>

<template>
  <div class="mine-page">
    <div class="header">
      <van-icon name="setting-o" size="22" class="set-btn" />
      <div v-if="loggedIn && user" class="profile">
        <img class="avatar" :src="user.avatar || 'https://via.placeholder.com/80'" />
        <div class="nick">
          {{ user.nickname || user.username }}
          <span v-if="user.level" class="level-badge" :class="'lv-' + user.level">Lv.{{ user.level }} {{ user.level_title }}</span>
        </div>
        <div class="uid">抖音号: {{ user.username }}</div>
        <div class="bio" v-if="user.bio">{{ user.bio }}</div>
        <div class="stats">
          <span><b>{{ fmtCount(user.following_count) }}</b> 关注</span>
          <span><b>{{ fmtCount(user.followers_count) }}</b> 粉丝</span>
          <span><b>{{ fmtCount(user.likes_count) }}</b> 获赞</span>
        </div>
      </div>
      <div v-else class="profile">
        <div class="avatar-placeholder" @click="router.push('/login')"><van-icon name="user-o" size="40" /></div>
        <div class="nick" @click="router.push('/login')">点击登录</div>
      </div>
    </div>
    <div v-if="loggedIn" class="action-row">
      <van-button size="small" round color="#fe2c55" plain @click="router.push('/profile')"><van-icon name="edit" /> 编辑资料</van-button>
      <van-button size="small" round color="#333" @click="router.push('/upload')">发布作品</van-button>
      <van-button size="small" round color="#333" @click="router.push('/dm')">私信</van-button>
      <van-button size="small" round color="#333" @click="router.push('/creator-stats')">数据中心</van-button>
    </div>
    <van-tabs v-model:active="tab" color="#fe2c55" background="#161616" title-active-color="#fff" title-inactive-color="#888" v-if="loggedIn">
      <van-tab title="作品" name="works">
        <div class="v-grid">
          <div v-for="v in videos" :key="v.id" class="v-item" @click="router.push('/feed')">
            <img class="v-cover" :src="v.cover_url" />
            <div class="v-title van-ellipsis">{{ v.title }}</div>
          </div>
          <van-empty v-if="!videos.length" description="还没有作品" image="search" />
        </div>
      </van-tab>
      <van-tab title="喜欢" name="likes">
        <div class="v-grid">
          <div v-for="v in favVideos" :key="v.id" class="v-item" @click="router.push('/feed')">
            <img class="v-cover" :src="v.cover_url" />
            <div class="v-title van-ellipsis">{{ v.title }}</div>
          </div>
          <van-empty v-if="!favVideos.length" description="还没有喜欢的内容" image="search" />
        </div>
      </van-tab>
    </van-tabs>
    <div v-if="loggedIn" style="margin: 20px"><van-button block plain color="#666" @click="logout">退出登录</van-button></div>
  </div>
</template>

<style scoped>
.mine-page { height: 100vh; overflow-y: auto; background: #000; }
.header { background: #161616; padding: 50px 20px 20px; position: relative; }
.set-btn { position: absolute; top: 16px; right: 16px; color: #fff; }
.profile { display: flex; flex-direction: column; align-items: center; }
.avatar { width: 80px; height: 80px; border-radius: 50%; }
.avatar-placeholder { width: 80px; height: 80px; border-radius: 50%; background: #333; display: flex; align-items: center; justify-content: center; color: #666; }
.nick { color: #fff; font-size: 18px; font-weight: bold; margin-top: 10px; display: flex; align-items: center; justify-content: center; gap: 8px; }
.level-badge { font-size: 11px; font-weight: normal; padding: 2px 8px; border-radius: 10px; color: #fff; }
.lv-1 { background: #8d6e63; }
.lv-2 { background: #9e9e9e; }
.lv-3 { background: #ffc107; color: #333; }
.lv-4 { background: #00bcd4; }
.lv-5 { background: #9c27b0; }
.lv-6 { background: linear-gradient(90deg, #fe2c55, #ffaa00); }
.uid { color: #888; font-size: 12px; margin-top: 4px; }
.bio { color: #ccc; font-size: 13px; margin-top: 8px; text-align: center; }
.stats { display: flex; gap: 24px; margin-top: 14px; color: #999; font-size: 13px; }
.stats b { color: #fff; font-size: 17px; }
.action-row { display: flex; justify-content: center; gap: 12px; padding: 12px; background: #161616; }
.v-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 4px; padding: 4px; background: #000; }
.v-item { background: #111; border-radius: 4px; overflow: hidden; }
.v-cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.v-title { color: #fff; font-size: 11px; padding: 4px; }
</style>
