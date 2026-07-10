<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getNearbyUsers, updateLocation } from '../api'

const router = useRouter()
const users = ref([])
const loading = ref(true)

// ===================== Feature: Distance range filter (附近的人距离筛选) =====================
// Three filter buttons narrow the list by the user's `distance` (in km).
// 'all' = 全部, '1km' = ≤1km, '5km' = ≤5km. The active button is highlighted.
const DIST_FILTERS = [
  { key: 'all', label: '全部', max: Infinity },
  { key: '1km', label: '1km内', max: 1 },
  { key: '5km', label: '5km内', max: 5 }
]
const activeDist = ref('all')
const filteredUsers = computed(() => {
  const f = DIST_FILTERS.find((d) => d.key === activeDist.value) || DIST_FILTERS[0]
  if (f.max === Infinity) return users.value
  return users.value.filter((u) => (typeof u.distance === 'number' ? u.distance : parseFloat(u.distance)) <= f.max)
})

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
    <!-- Feature: 附近的人距离筛选 — three distance filter buttons. -->
    <div class="dist-bar" v-if="!loading && users.length">
      <span
        v-for="f in DIST_FILTERS"
        :key="f.key"
        class="dist-pill"
        :class="{ active: activeDist === f.key }"
        @click="activeDist = f.key"
      >{{ f.label }}</span>
    </div>
    <!-- Feature: 找到N位附近的人 — result count, shown when filtering is active. -->
    <div class="dist-count" v-if="!loading && users.length">
      找到{{ filteredUsers.length }}位附近的人
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!users.length" description="附近暂无其他用户" />
    <van-empty v-else-if="!filteredUsers.length" :description="`${activeDist === '1km' ? '1km' : '5km'}内暂无用户`" />
    <div v-else class="user-list">
      <div v-for="u in filteredUsers" :key="u.id" class="user-item" @click="router.push('/user/' + u.id)">
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
/* Feature: 附近的人距离筛选 — filter button row + count text. */
.dist-bar { display: flex; gap: 10px; padding: 12px; background: #000; }
.dist-pill { background: #161616; color: #fff; font-size: 13px; padding: 6px 16px; border-radius: 16px; transition: background 0.15s, color 0.15s; }
.dist-pill.active { background: #fe2c55; color: #fff; font-weight: bold; }
.dist-count { color: #888; font-size: 12px; padding: 0 16px 8px; }
.loading { text-align: center; padding: 60px; }
.user-list { padding: 8px; }
.user-item { display: flex; align-items: center; gap: 12px; padding: 12px; background: #161616; border-radius: 10px; margin-bottom: 8px; }
.u-avatar { width: 50px; height: 50px; border-radius: 50%; }
.u-info { flex: 1; }
.u-name { color: #fff; font-size: 15px; }
.u-city { color: #888; font-size: 12px; margin-top: 2px; }
.u-dist { color: #25f4ee; font-size: 13px; font-weight: bold; }
</style>
