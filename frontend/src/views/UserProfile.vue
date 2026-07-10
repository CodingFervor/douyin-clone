<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getUser, getUserVideos, toggleFollow } from '../api'

const route = useRoute()
const router = useRouter()
const user = ref(null)
const videos = ref([])
const loading = ref(true)

// ===================== Feature: 主页主题 (profile theme) =====================
// Four selectable themes for the profile header background. The selection is
// persisted to localStorage('dy_profile_theme') and read back on mount.
const themes = [
  { key: 'default', label: '默认(黑)', swatch: '#161616', gradient: '' },
  { key: 'pink', label: '渐变粉', swatch: 'linear-gradient(135deg, #fe2c55, #ff6b9d)', gradient: 'linear-gradient(135deg, #fe2c55, #ff6b9d)' },
  { key: 'blue', label: '渐变蓝', swatch: 'linear-gradient(135deg, #25f4ee, #4facfe)', gradient: 'linear-gradient(135deg, #25f4ee, #4facfe)' },
  { key: 'purple', label: '渐变紫', swatch: 'linear-gradient(135deg, #a855f7, #6366f1)', gradient: 'linear-gradient(135deg, #a855f7, #6366f1)' },
]

const selectedTheme = ref('default')

// The applied header background: a gradient for non-default themes, falling
// back to the solid dark color for the default theme.
const headerStyle = computed(() => {
  const t = themes.find((x) => x.key === selectedTheme.value)
  if (t && t.gradient) return { backgroundImage: t.gradient }
  return {}
})

// The "default" theme renders dark text colors, while gradient themes look
// better with light text. We toggle a class for that.
const headerClass = computed(() => ({
  'head-themed': selectedTheme.value !== 'default',
}))

function pickTheme(key) {
  selectedTheme.value = key
  localStorage.setItem('dy_profile_theme', key)
  showToast('主题已更换')
}

onMounted(async () => {
  // Restore saved theme (best-effort).
  try {
    const saved = localStorage.getItem('dy_profile_theme')
    if (saved && themes.some((t) => t.key === saved)) selectedTheme.value = saved
  } catch (e) {
    // ignore
  }
  try {
    user.value = await getUser(route.params.id)
    videos.value = await getUserVideos(route.params.id)
  } catch (e) {
    showToast('用户不存在')
  } finally {
    loading.value = false
  }
})

async function doFollow() {
  if (!localStorage.getItem('dy_token')) { router.push('/login'); return }
  try {
    const res = await toggleFollow(user.value.id)
    user.value.is_following = res.following
  } catch (e) {
    showToast('操作失败')
  }
}

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
</script>

<template>
  <div class="profile-page">
    <van-nav-bar left-arrow @click-left="router.back()" :placeholder="false" class="top-bar">
      <template #left><van-icon name="arrow-left" color="#fff" size="22" /></template>
    </van-nav-bar>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <template v-else-if="user">
      <div class="head" :class="headerClass" :style="headerStyle">
        <img class="avatar" :src="user.avatar || 'https://via.placeholder.com/80'" />
        <div class="nick">{{ user.nickname || user.username }}</div>
        <div class="uid">抖音号: {{ user.username }}</div>
        <div class="bio" v-if="user.bio">{{ user.bio }}</div>
        <div class="stats">
          <span><b>{{ fmtCount(user.following_count) }}</b> 关注</span>
          <span><b>{{ fmtCount(user.followers_count) }}</b> 粉丝</span>
          <span><b>{{ fmtCount(user.likes_count) }}</b> 获赞</span>
        </div>
        <van-button v-if="!user.is_following" size="small" round color="#fe2c55" class="follow-btn" @click="doFollow">+ 关注</van-button>
        <van-button v-else size="small" round color="#333" class="follow-btn" @click="doFollow">已关注</van-button>

        <!-- ===================== Feature: 主页主题 (theme picker) ===================== -->
        <div class="theme-picker">
          <span
            v-for="t in themes"
            :key="t.key"
            class="swatch"
            :class="{ active: selectedTheme === t.key }"
            :style="{ background: t.swatch }"
            :title="t.label"
            @click="pickTheme(t.key)"
          ></span>
        </div>
      </div>
      <div class="tab-head">作品 {{ videos.length }}</div>
      <div class="v-grid">
        <div v-for="v in videos" :key="v.id" class="v-item" @click="router.push('/feed')">
          <img class="v-cover" :src="v.cover_url" />
          <div class="v-title van-ellipsis">{{ v.title }}</div>
          <div class="v-plays"><van-icon name="play-circle-o" /> {{ fmtCount(v.plays) }}</div>
        </div>
        <van-empty v-if="!videos.length" description="暂无作品" image="search" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.profile-page { height: 100vh; overflow-y: auto; background: #000; }
.top-bar { background: transparent !important; }
.loading { text-align: center; padding: 80px; }
.head { position: relative; padding: 10px 20px 20px; background: #161616; text-align: center; transition: background-image 0.4s ease; }
.head-themed { color: #fff; }
.avatar { width: 80px; height: 80px; border-radius: 50%; margin: 0 auto; border: 2px solid rgba(255,255,255,0.4); }
.nick { color: #fff; font-size: 20px; font-weight: bold; margin-top: 10px; }
.uid { color: rgba(255,255,255,0.7); font-size: 12px; margin-top: 4px; }
.bio { color: rgba(255,255,255,0.85); font-size: 13px; margin-top: 8px; }
.stats { display: flex; justify-content: center; gap: 24px; margin-top: 14px; color: rgba(255,255,255,0.85); font-size: 13px; }
.stats b { color: #fff; font-size: 17px; }
.follow-btn { position: absolute; top: 16px; right: 16px; }
.tab-head { color: #888; font-size: 13px; padding: 14px 16px; }
.v-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 4px; padding: 4px; }
.v-item { background: #111; border-radius: 4px; overflow: hidden; }
.v-cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.v-title { color: #fff; font-size: 11px; padding: 4px; }
.v-plays { color: #888; font-size: 10px; padding: 0 4px 4px; }

/* ===================== Feature: 主页主题 (theme picker) ===================== */
.theme-picker {
  display: flex; justify-content: center; gap: 10px; margin-top: 16px;
}
.swatch {
  width: 22px; height: 22px; border-radius: 50%; cursor: pointer;
  border: 2px solid rgba(255,255,255,0.3); transition: transform 0.15s, border-color 0.15s, box-shadow 0.15s;
}
.swatch:hover { transform: scale(1.12); }
.swatch.active {
  border-color: #fff;
  box-shadow: 0 0 0 2px rgba(255,255,255,0.5);
  transform: scale(1.15);
}
</style>
