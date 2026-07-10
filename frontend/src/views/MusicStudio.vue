<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getHotMusic } from '../api'

const router = useRouter()
const list = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    list.value = await getHotMusic()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})

// Top 3 highlighted as larger BGM recommendation cards.
const topThree = computed(() => list.value.slice(0, 3))
// The remaining entries render as compact list rows.
const restList = computed(() => list.value.slice(3).map((m, i) => ({ ...m, rank: i + 4 })))

function fmt(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}

function useMusic(name) {
  router.push({ path: '/upload', query: { music: name } })
}
</script>

<template>
  <div class="ms-page">
    <van-nav-bar title="音乐工作室" left-arrow @click-left="router.back()" fixed placeholder />
    <!-- Gradient header -->
    <div class="ms-header">
      <div class="ms-title">音乐工作室</div>
      <div class="ms-sub">为你的视频挑选最火 BGM</div>
    </div>

    <!-- 热门BGM推荐: top 3 in larger cards -->
    <div class="section-head">🔥 热门BGM推荐</div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <div v-else-if="!list.length"><van-empty description="暂无音乐" /></div>
    <template v-else>
      <div class="top-grid">
        <div v-for="(m, i) in topThree" :key="i" class="top-card" :class="'rank-' + (i + 1)">
          <div class="tc-rank">TOP {{ i + 1 }}</div>
          <div class="tc-name van-ellipsis">🎵 {{ m.music }}</div>
          <div class="tc-uses">{{ fmt(m.uses) }} 个作品使用</div>
          <van-button size="small" round color="#fe2c55" @click="useMusic(m.music)">使用此音乐</van-button>
        </div>
      </div>

      <!-- Remaining top music list -->
      <div class="section-head">🎵 热门音乐排行</div>
      <div class="music-list">
        <div v-for="m in restList" :key="m.rank" class="music-row">
          <span class="mr-rank">{{ m.rank }}</span>
          <div class="mr-info">
            <div class="mr-name van-ellipsis">{{ m.music }}</div>
            <div class="mr-uses">{{ fmt(m.uses) }} 次使用</div>
          </div>
          <van-button size="mini" round color="#fe2c55" @click="useMusic(m.music)">使用此音乐</van-button>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.ms-page { height: 100vh; overflow-y: auto; background: #000; }
.ms-page :deep(.van-nav-bar) { background: #000; }
.ms-page :deep(.van-nav-bar__title) { color: #fff; }
.ms-page :deep(.van-nav-bar .van-icon) { color: #fff; }

/* Gradient header */
.ms-header {
  background: linear-gradient(135deg, #fe2c55 0%, #ff6a88 50%, #25f4ee 100%);
  padding: 28px 20px 24px;
  text-align: center;
  border-bottom-left-radius: 18px;
  border-bottom-right-radius: 18px;
}
.ms-title { color: #fff; font-size: 26px; font-weight: bold; letter-spacing: 2px; text-shadow: 0 2px 8px rgba(0,0,0,0.3); }
.ms-sub { color: rgba(255,255,255,0.9); font-size: 13px; margin-top: 6px; }

.section-head { color: #fff; font-size: 16px; font-weight: bold; padding: 16px 16px 8px; }
.loading { text-align: center; padding: 60px; }

/* Top 3 larger cards */
.top-grid { display: grid; grid-template-columns: 1fr; gap: 12px; padding: 0 12px; }
.top-card { background: #161616; border-radius: 14px; padding: 16px; display: flex; flex-direction: column; gap: 6px; position: relative; overflow: hidden; border-left: 4px solid #444; }
.top-card.rank-1 { border-left-color: #ffd700; background: linear-gradient(135deg, #2a2014, #161616); }
.top-card.rank-2 { border-left-color: #c0c0c0; background: linear-gradient(135deg, #202028, #161616); }
.top-card.rank-3 { border-left-color: #cd7f32; background: linear-gradient(135deg, #281a16, #161616); }
.tc-rank { font-size: 11px; font-weight: bold; color: #fe2c55; letter-spacing: 1px; }
.tc-name { color: #fff; font-size: 17px; font-weight: bold; }
.tc-uses { color: #888; font-size: 12px; }
.top-card :deep(.van-button) { align-self: flex-start; margin-top: 4px; }

/* Remaining list rows */
.music-list { padding: 0 12px; }
.music-row { display: flex; align-items: center; gap: 12px; padding: 12px; background: #161616; border-radius: 10px; margin-bottom: 8px; }
.mr-rank { width: 24px; text-align: center; color: #666; font-weight: bold; font-size: 16px; }
.mr-info { flex: 1; min-width: 0; }
.mr-name { color: #fff; font-size: 14px; }
.mr-uses { color: #666; font-size: 11px; margin-top: 2px; }
</style>
