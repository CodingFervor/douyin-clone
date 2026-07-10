<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getCreatorStats } from '../api'

const router = useRouter()
const stats = ref(null)
const loading = ref(true)

onMounted(async () => { try { stats.value = await getCreatorStats() } catch (e) { showToast('加载失败') } finally { loading.value = false } })
function fmt(n) { if (n >= 10000) return (n / 10000).toFixed(1) + 'w'; return String(n) }

// 6 achievements, in order
const ACHIEVEMENTS = [
  { id: 'first',    icon: '🎬', name: '首次发布', desc: '发布首个作品',   check: (s) => s.video_count >= 1 },
  { id: 'thousand', icon: '📈', name: '千播达人', desc: '总播放破 1,000', check: (s) => s.total_plays >= 1000 },
  { id: 'fans',     icon: '❤️', name: '万人迷',   desc: '总点赞破 10,000', check: (s) => s.total_likes >= 10000 },
  { id: 'interact', icon: '💬', name: '互动王',   desc: '总评论破 1,000',  check: (s) => s.total_comments >= 1000 },
  { id: 'viral',    icon: '🚀', name: '爆款制造', desc: '总播放破 100,000', check: (s) => s.total_plays >= 100000 },
  { id: 'master',   icon: '👑', name: '创作大师', desc: '集齐以上全部成就', check: (s) =>
    s.video_count >= 1 && s.total_plays >= 100000 && s.total_likes >= 10000 && s.total_comments >= 1000 },
]

const evaluated = computed(() => {
  if (!stats.value) return []
  return ACHIEVEMENTS.map((a) => ({ ...a, earned: a.check(stats.value) }))
})

const nextHint = computed(() => {
  const next = evaluated.value.find((a) => !a.earned)
  if (!next) return null
  return next
})
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
      <div class="ach-section">
        <div class="ach-title">🏆 成就徽章</div>
        <div class="ach-grid">
          <div
            v-for="a in evaluated"
            :key="a.id"
            class="ach-badge"
            :class="{ earned: a.earned, locked: !a.earned }"
          >
            <div class="ach-icon">{{ a.earned ? a.icon : '🔒' }}</div>
            <div class="ach-name">{{ a.name }}</div>
            <div class="ach-desc">{{ a.desc }}</div>
          </div>
        </div>
        <div v-if="nextHint" class="ach-progress">
          下一目标：<span class="next-name">{{ nextHint.icon }} {{ nextHint.name }}</span> · {{ nextHint.desc }}
        </div>
        <div v-else class="ach-progress done">🎉 已解锁全部成就，创作大师就是你！</div>
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
.ach-section { padding: 0 16px 24px; }
.ach-title { color: #fff; font-size: 16px; font-weight: bold; margin-bottom: 12px; }
.ach-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 8px;
}
.ach-badge {
  border-radius: 12px;
  padding: 14px 8px;
  text-align: center;
  border: 1px solid transparent;
  transition: transform 0.15s;
}
.ach-badge.earned {
  background: linear-gradient(135deg, rgba(254,44,85,0.18), rgba(37,244,238,0.12));
  border-color: rgba(254,44,85,0.45);
  box-shadow: 0 0 10px rgba(254,44,85,0.15);
}
.ach-badge.earned:active { transform: scale(0.97); }
.ach-badge.locked {
  background: #161616;
  border-color: #2a2a2a;
  opacity: 0.65;
  filter: grayscale(0.6);
}
.ach-icon { font-size: 26px; line-height: 1; }
.ach-name { color: #fff; font-size: 12px; font-weight: bold; margin-top: 6px; }
.ach-desc { color: #888; font-size: 10px; margin-top: 2px; line-height: 13px; }
.ach-badge.locked .ach-name { color: #666; }
.ach-badge.locked .ach-desc { color: #555; }
.ach-progress {
  margin-top: 14px;
  padding: 10px 14px;
  border-radius: 10px;
  background: rgba(255,255,255,0.06);
  color: #aaa;
  font-size: 12px;
  text-align: center;
}
.ach-progress .next-name { color: #fe2c55; font-weight: bold; }
.ach-progress.done { color: #25f4ee; }
</style>
