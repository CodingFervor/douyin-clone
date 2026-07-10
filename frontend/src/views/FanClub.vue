<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile, getContributors, getLiveList, getSuggestFollows, toggleFollow } from '../api'

const router = useRouter()
const user = ref(null)
const contributors = ref([])
const suggests = ref([])
const loading = ref(true)

// Demo fan data used when no live-room contributors are available.
const DEMO_FANS = [
  { user_id: 1, nickname: '铁粉小明', avatar: 'https://via.placeholder.com/80', amount: 8888 },
  { user_id: 2, nickname: '老粉小红', avatar: 'https://via.placeholder.com/80', amount: 6666 },
  { user_id: 3, nickname: '忠实粉丝阿强', avatar: 'https://via.placeholder.com/80', amount: 5200 },
  { user_id: 4, nickname: '默默支持者', avatar: 'https://via.placeholder.com/80', amount: 3210 },
  { user_id: 5, nickname: '路人甲乙丙', avatar: 'https://via.placeholder.com/80', amount: 1888 },
  { user_id: 6, nickname: '夜猫子观众', avatar: 'https://via.placeholder.com/80', amount: 1024 },
]

// Hardcoded weekly fan-growth demo data (Mon..Sun).
const WEEK_GROWTH = [
  { day: '一', value: 120 },
  { day: '二', value: 80 },
  { day: '三', value: 200 },
  { day: '四', value: 150 },
  { day: '五', value: 320 },
  { day: '六', value: 280 },
  { day: '日', value: 410 },
]
const maxGrowth = computed(() => Math.max(...WEEK_GROWTH.map((d) => d.value)))

async function load() {
  try {
    const [me, liveData, suggestData] = await Promise.allSettled([
      getProfile(),
      // Sample live room for the contribution board; fall back to demo.
      getLiveList().then((rooms) => {
        const id = rooms && rooms.length ? rooms[0].id : null
        return id ? getContributors(id) : []
      }),
      getSuggestFollows(),
    ])
    if (me.status === 'fulfilled') user.value = me.value
    contributors.value = liveData.status === 'fulfilled' && liveData.value && liveData.value.length
      ? liveData.value
      : DEMO_FANS
    suggests.value = suggestData.status === 'fulfilled' ? (suggestData.value || []) : []
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// Fan-club level derived from follower count:
// 100=1, 1000=2, 10000=3, 100000=4
const fanLevel = computed(() => {
  const n = (user.value && user.value.followers_count) || 0
  if (n >= 100000) return 4
  if (n >= 10000) return 3
  if (n >= 1000) return 2
  if (n >= 100) return 1
  return 1
})

const coreFans = computed(() => contributors.value.slice(0, 8))

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n || 0)
}

function sendRedPacket() {
  showToast('红包功能即将上线，敬请期待')
}

async function doSuggestFollow(u) {
  if (!localStorage.getItem('dy_token')) { router.push('/login'); return }
  try {
    const res = await toggleFollow(u.id)
    u.followed = res.following
  } catch (e) {
    showToast('操作失败')
  }
}
</script>

<template>
  <div class="fc-page">
    <van-nav-bar title="粉丝团" left-arrow @click-left="router.back()" fixed placeholder />

    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <template v-else>
      <!-- Header: avatar + follower count + fan-club level -->
      <div class="fc-header">
        <div class="fc-title">💜 粉丝团</div>
        <div class="fc-user">
          <img class="fc-avatar" :src="(user && user.avatar) || 'https://via.placeholder.com/80'" />
          <div class="fc-user-info">
            <div class="fc-nick">{{ (user && (user.nickname || user.username)) || '抖音用户' }}</div>
            <div class="fc-followers"><b>{{ fmtCount(user && user.followers_count) }}</b> 粉丝</div>
          </div>
          <div class="fc-level">粉丝团等级 Lv.{{ fanLevel }}</div>
        </div>
        <div class="fc-likes" v-if="user">
          💗 共获 <b>{{ fmtCount(user.likes_count) }}</b> 赞
        </div>
      </div>

      <!-- 核心粉丝: grid of top contributor avatars -->
      <div class="panel">
        <div class="panel-head">👥 核心粉丝</div>
        <div v-if="!coreFans.length" class="empty-tip">暂无核心粉丝数据</div>
        <div v-else class="fans-grid">
          <div v-for="(c, i) in coreFans" :key="c.user_id || i" class="fan-item">
            <div class="fan-rank" :class="{ top: i < 3 }">{{ i + 1 }}</div>
            <img class="fan-avatar" :src="c.avatar" />
            <div class="fan-name van-ellipsis">{{ c.nickname }}</div>
            <div class="fan-amount">{{ fmtCount(c.amount) }}</div>
          </div>
        </div>
      </div>

      <!-- 粉丝增长: 7-day CSS bar chart -->
      <div class="panel">
        <div class="panel-head">📈 粉丝增长（近7天）</div>
        <div class="chart">
          <div v-for="d in WEEK_GROWTH" :key="d.day" class="chart-col">
            <div class="chart-bar-wrap">
              <div class="chart-bar" :style="{ height: Math.max(8, Math.round((d.value / maxGrowth) * 100)) + '%' }"></div>
            </div>
            <div class="chart-value">{{ d.value }}</div>
            <div class="chart-day">{{ d.day }}</div>
          </div>
        </div>
      </div>

      <!-- 你可能感兴趣的创作者 -->
      <div v-if="suggests.length" class="panel">
        <div class="panel-head">✨ 你可能感兴趣的创作者</div>
        <div class="suggest-list">
          <div v-for="u in suggests.slice(0, 4)" :key="u.id" class="su-item" @click="router.push('/user/' + u.id)">
            <img class="su-avatar" :src="u.avatar" />
            <div class="su-info">
              <div class="su-name van-ellipsis">{{ u.nickname }}</div>
              <div class="su-fans">{{ fmtCount(u.followers_count) }}粉丝</div>
            </div>
            <van-button size="mini" round color="#fe2c55" :disabled="u.followed" @click.stop="doSuggestFollow(u)">{{ u.followed ? '已关注' : '关注' }}</van-button>
          </div>
        </div>
      </div>

      <!-- 粉丝福利: 发红包 card -->
      <div class="panel">
        <div class="panel-head">🎁 粉丝福利</div>
        <div class="welfare-card">
          <div class="wc-text">🎉 感谢粉丝支持！发送红包回馈你的铁粉吧～</div>
          <van-button type="primary" color="#fe2c55" round block @click="sendRedPacket">
            <van-icon name="balance-pay" /> 发红包
          </van-button>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.fc-page { height: 100vh; overflow-y: auto; background: #000; }
.fc-page :deep(.van-nav-bar) { background: #000; }
.fc-page :deep(.van-nav-bar__title) { color: #fff; }
.fc-page :deep(.van-nav-bar .van-icon) { color: #fff; }
.loading { text-align: center; padding: 80px; }

/* Header */
.fc-header {
  background: linear-gradient(135deg, #2a0a14, #161616);
  padding: 16px; border-bottom: 1px solid #1f1f1f;
}
.fc-title { color: #fff; font-size: 20px; font-weight: bold; }
.fc-user { display: flex; align-items: center; gap: 14px; margin-top: 14px; }
.fc-avatar { width: 64px; height: 64px; border-radius: 50%; flex-shrink: 0; }
.fc-user-info { flex: 1; min-width: 0; }
.fc-nick { color: #fff; font-size: 17px; font-weight: bold; }
.fc-followers { color: #aaa; font-size: 13px; margin-top: 4px; }
.fc-followers b { color: #fe2c55; font-size: 16px; }
.fc-level {
  font-size: 12px; color: #fff; font-weight: bold;
  background: linear-gradient(90deg, #fe2c55, #ffaa00);
  padding: 6px 12px; border-radius: 14px; white-space: nowrap; flex-shrink: 0;
}
.fc-likes { color: #aaa; font-size: 13px; margin-top: 14px; }
.fc-likes b { color: #fff; }

/* Panel */
.panel { background: #161616; border-radius: 12px; margin: 12px; padding: 14px; border: 1px solid #1f1f1f; }
.panel-head { color: #fff; font-size: 15px; font-weight: bold; margin-bottom: 14px; }
.empty-tip { color: #666; font-size: 13px; text-align: center; padding: 20px 0; }

/* Core fans grid */
.fans-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px 8px; }
.fan-item { display: flex; flex-direction: column; align-items: center; position: relative; }
.fan-rank {
  position: absolute; top: -2px; left: 50%; transform: translateX(-10px);
  font-size: 10px; color: #888; background: #2a2a2a; width: 16px; height: 16px;
  border-radius: 50%; display: flex; align-items: center; justify-content: center; z-index: 2;
}
.fan-rank.top { color: #fff; background: #fe2c55; }
.fan-avatar { width: 48px; height: 48px; border-radius: 50%; }
.fan-name { color: #ddd; font-size: 11px; margin-top: 6px; max-width: 64px; text-align: center; }
.fan-amount { color: #fe2c55; font-size: 11px; margin-top: 2px; }

/* Growth chart */
.chart { display: flex; align-items: flex-end; justify-content: space-between; height: 140px; gap: 6px; }
.chart-col { flex: 1; display: flex; flex-direction: column; align-items: center; height: 100%; }
.chart-bar-wrap { flex: 1; width: 100%; display: flex; align-items: flex-end; }
.chart-bar { width: 70%; margin: 0 auto; background: linear-gradient(180deg, #ff6a88, #fe2c55); border-radius: 4px 4px 0 0; transition: height 0.5s ease; }
.chart-value { color: #fe2c55; font-size: 10px; margin-top: 4px; }
.chart-day { color: #888; font-size: 11px; }

/* Suggest list */
.suggest-list { display: flex; flex-direction: column; gap: 12px; }
.su-item { display: flex; align-items: center; gap: 10px; }
.su-avatar { width: 40px; height: 40px; border-radius: 50%; flex-shrink: 0; }
.su-info { flex: 1; min-width: 0; }
.su-name { color: #fff; font-size: 14px; }
.su-fans { color: #888; font-size: 11px; margin-top: 2px; }

/* Welfare */
.welfare-card { text-align: center; }
.wc-text { color: #ccc; font-size: 13px; margin-bottom: 14px; line-height: 1.6; }
</style>
