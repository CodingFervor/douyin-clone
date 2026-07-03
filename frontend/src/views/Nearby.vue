<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getNearbyUsers, updateLocation } from '../api'

const router = useRouter()
const users = ref([])
const loading = ref(true)

onMounted(async () => {
  // Try to record the visitor's location (best-effort), then load the list.
  if (navigator.geolocation) {
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        try {
          await updateLocation(pos.coords.latitude, pos.coords.longitude, '')
        } catch (e) { /* ignore */ }
        await load()
      },
      async () => { await load() }, // permission denied — load with stored/default
      { timeout: 5000 }
    )
  } else {
    await load()
  }
})

async function load() {
  loading.value = true
  try {
    users.value = await getNearbyUsers()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}

function fmtDist(km) {
  if (km < 1) return Math.round(km * 1000) + 'm'
  return km.toFixed(1) + 'km'
}
</script>

<template>
  <div class="nearby-page">
    <van-nav-bar title="附近的人" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="banner">📍 发现身边有趣的灵魂</div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!users.length" description="附近暂无其他用户" />
    <div v-else class="user-list">
      <div v-for="u in users" :key="u.id" class="user-item" @click="router.push('/user/' + u.id)">
        <img class="u-avatar" :src="u.avatar || 'https://via.placeholder.com/50'" />
        <div class="u-info">
          <div class="u-name">{{ u.nickname || u.username }}</div>
          <div class="u-city" v-if="u.city">{{ u.city }}</div>
        </div>
        <div class="u-dist">{{ fmtDist(u.distance) }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.nearby-page { height: 100vh; overflow-y: auto; background: #000; }
.banner { background: linear-gradient(135deg, #25f4ee, #fe2c55); color: #fff; text-align: center; padding: 14px; font-size: 15px; font-weight: bold; }
.loading { text-align: center; padding: 60px; }
.user-list { padding: 8px; }
.user-item { display: flex; align-items: center; gap: 12px; padding: 12px; background: #161616; border-radius: 10px; margin-bottom: 8px; }
.u-avatar { width: 50px; height: 50px; border-radius: 50%; }
.u-info { flex: 1; }
.u-name { color: #fff; font-size: 15px; }
.u-city { color: #888; font-size: 12px; margin-top: 2px; }
.u-dist { color: #25f4ee; font-size: 13px; font-weight: bold; }
</style>
