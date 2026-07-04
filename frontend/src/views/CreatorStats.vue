<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getCreatorStats } from '../api'

const router = useRouter()
const stats = ref(null)
const loading = ref(true)

onMounted(async () => { try { stats.value = await getCreatorStats() } catch (e) { showToast('加载失败') } finally { loading.value = false } })
function fmt(n) { if (n >= 10000) return (n / 10000).toFixed(1) + 'w'; return String(n) }
</script>

<template>
  <div class="cs-page">
    <van-nav-bar title="创作数据中心" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <div v-else-if="stats" class="cs-body">
      <div class="cs-banner">📊 我的数据</div>
      <div class="stat-grid">
        <div class="stat-card">
          <div class="sc-value">{{ stats.video_count }}</div>
          <div class="sc-label">作品数</div>
        </div>
        <div class="stat-card">
          <div class="sc-value">{{ fmt(stats.total_plays) }}</div>
          <div class="sc-label">总播放</div>
        </div>
        <div class="stat-card">
          <div class="sc-value">{{ fmt(stats.total_likes) }}</div>
          <div class="sc-label">总点赞</div>
        </div>
        <div class="stat-card">
          <div class="sc-value">{{ fmt(stats.total_comments) }}</div>
          <div class="sc-label">总评论</div>
        </div>
        <div class="stat-card">
          <div class="sc-value">{{ fmt(stats.total_shares) }}</div>
          <div class="sc-label">总分享</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cs-page { height: 100vh; overflow-y: auto; background: #000; }
.loading { text-align: center; padding: 60px; }
.cs-banner { background: linear-gradient(135deg, #fe2c55, #25f4ee); color: #fff; text-align: center; padding: 20px; font-size: 18px; font-weight: bold; }
.stat-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; padding: 16px; }
.stat-card { background: #161616; border-radius: 12px; padding: 24px 16px; text-align: center; }
.sc-value { color: #fe2c55; font-size: 28px; font-weight: bold; }
.sc-label { color: #888; font-size: 13px; margin-top: 6px; }
.stat-card:nth-child(1) { grid-column: span 2; }
.stat-card:nth-child(1) .sc-value { font-size: 36px; }
</style>
