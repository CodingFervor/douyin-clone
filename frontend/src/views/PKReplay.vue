<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

// ===================== Feature: 直播PK回放 (PK battle replay) =====================
// Frontend-only demo page. The data below is hardcoded to emulate past PK
// battles recorded in the pk_battles table; there is no dedicated history API,
// so this serves as a self-contained replay showcase.
const battles = ref([
  {
    room_a_name: '甜美小猫咪',
    room_b_name: '潮流音乐人',
    score_a: 2860000,
    score_b: 3120000,
    winner: 'b',
    time_ago: '2小时前',
  },
  {
    room_a_name: '舞蹈天后Lily',
    room_b_name: '搞笑大王阿杰',
    score_a: 4510000,
    score_b: 3980000,
    winner: 'a',
    time_ago: '5小时前',
  },
  {
    room_a_name: '美妆达人Coco',
    room_b_name: '健身教练Leo',
    score_a: 1230000,
    score_b: 1190000,
    winner: 'a',
    time_ago: '昨天',
  },
  {
    room_a_name: '游戏大神Tom',
    room_b_name: '美食家大胃王',
    score_a: 5670000,
    score_b: 6210000,
    winner: 'b',
    time_ago: '2天前',
  },
  {
    room_a_name: '萌宠乐园',
    room_b_name: '旅行日记',
    score_a: 890000,
    score_b: 1020000,
    winner: 'b',
    time_ago: '3天前',
  },
])

// Format large scores into a compact, readable form (e.g. 312w / 1.2k).
function fmtScore(n) {
  if (n >= 10000) return (n / 10000).toFixed(1).replace(/\.0$/, '') + 'w'
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k'
  return String(n)
}

// For each battle, the score split as a percentage of the total. Used to draw
// the progress bar that visualizes how close the battle was.
function split(b) {
  const total = b.score_a + b.score_b
  if (!total) return { aPct: 50, bPct: 50, gap: 0 }
  const aPct = Math.round((b.score_a / total) * 100)
  const bPct = 100 - aPct
  const gap = Math.abs(b.score_a - b.score_b)
  return { aPct, bPct, gap }
}

// A qualitative label for how competitive the battle was, derived from the
// score gap relative to the total.
function closenessLabel(b) {
  const total = b.score_a + b.score_b
  if (!total) return '势均力敌'
  const ratio = Math.abs(b.score_a - b.score_b) / total
  if (ratio < 0.02) return '势均力敌'
  if (ratio < 0.05) return '激烈对决'
  if (ratio < 0.12) return '实力接近'
  return '一边倒'
}

const totalBattles = computed(() => battles.value.length)
</script>

<template>
  <div class="pk-page">
    <van-nav-bar title="PK精彩回顾" left-arrow @click-left="router.back()" />
    <div class="pk-banner">
      <div class="pk-banner-title">🏆 PK精彩回顾</div>
      <div class="pk-banner-sub">共 {{ totalBattles }} 场经典对决，重温燃情时刻</div>
    </div>

    <div class="pk-list">
      <div v-for="(b, i) in battles" :key="i" class="pk-card">
        <div class="pk-card-top">
          <span class="pk-time">⏱ {{ b.time_ago }}</span>
          <span class="pk-closeness">{{ closenessLabel(b) }}</span>
        </div>

        <div class="pk-vs-row">
          <!-- Room A -->
          <div class="pk-side" :class="{ winner: b.winner === 'a' }">
            <div class="pk-crown" v-if="b.winner === 'a'">👑</div>
            <div class="pk-name">{{ b.room_a_name }}</div>
            <div class="pk-score" :class="{ 'score-win': b.winner === 'a' }">{{ fmtScore(b.score_a) }}</div>
            <div class="pk-tag" :class="{ 'tag-win': b.winner === 'a' }">
              {{ b.winner === 'a' ? '胜' : '负' }}
            </div>
          </div>

          <div class="pk-vs">VS</div>

          <!-- Room B -->
          <div class="pk-side" :class="{ winner: b.winner === 'b' }">
            <div class="pk-crown" v-if="b.winner === 'b'">👑</div>
            <div class="pk-name">{{ b.room_b_name }}</div>
            <div class="pk-score" :class="{ 'score-win': b.winner === 'b' }">{{ fmtScore(b.score_b) }}</div>
            <div class="pk-tag" :class="{ 'tag-win': b.winner === 'b' }">
              {{ b.winner === 'b' ? '胜' : '负' }}
            </div>
          </div>
        </div>

        <!-- Score split bar -->
        <div class="pk-bar-wrap">
          <div class="pk-bar-labels">
            <span class="pk-bar-a">{{ split(b).aPct }}%</span>
            <span class="pk-bar-gap">差距 {{ fmtScore(split(b).gap) }}</span>
            <span class="pk-bar-b">{{ split(b).bPct }}%</span>
          </div>
          <div class="pk-bar">
            <div class="pk-bar-fill-a" :style="{ width: split(b).aPct + '%' }"></div>
            <div class="pk-bar-fill-b" :style="{ width: split(b).bPct + '%' }"></div>
          </div>
        </div>
      </div>
    </div>

    <div class="pk-footer">— 没有更多回放了 —</div>
  </div>
</template>

<style scoped>
.pk-page { height: 100vh; overflow-y: auto; background: #0a0000; }
.pk-page :deep(.van-nav-bar) { background: #1a0505; }
.pk-page :deep(.van-nav-bar__title) { color: #ffd700; font-weight: bold; }
.pk-page :deep(.van-nav-bar .van-icon) { color: #ffd700; }

/* Banner */
.pk-banner {
  background: linear-gradient(135deg, #2a0a0a 0%, #1a0505 60%, #3a0000 100%);
  padding: 20px 16px 24px;
  text-align: center;
  border-bottom: 1px solid #3a1010;
}
.pk-banner-title {
  font-size: 22px;
  font-weight: bold;
  background: linear-gradient(90deg, #ffd700, #ffaa00, #fe2c55);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.pk-banner-sub { color: #b8860b; font-size: 12px; margin-top: 8px; }

/* List */
.pk-list { padding: 12px; display: flex; flex-direction: column; gap: 14px; }

/* Card */
.pk-card {
  background: linear-gradient(160deg, #1a0808 0%, #0f0303 100%);
  border-radius: 14px;
  padding: 14px;
  border: 1px solid #3a1010;
  box-shadow: 0 4px 16px rgba(254, 44, 85, 0.12);
}
.pk-card-top {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 14px;
}
.pk-time { color: #8a6a3a; font-size: 12px; }
.pk-closeness {
  font-size: 11px; color: #ffd700;
  background: rgba(255, 215, 0, 0.12); border: 1px solid rgba(255, 215, 0, 0.3);
  padding: 2px 10px; border-radius: 10px;
}

/* VS row */
.pk-vs-row {
  display: flex; align-items: stretch; justify-content: space-between; gap: 8px;
}
.pk-side {
  flex: 1; display: flex; flex-direction: column; align-items: center;
  gap: 6px; padding: 8px 4px; border-radius: 10px; position: relative;
  border: 1px solid transparent;
}
.pk-side.winner {
  border-color: rgba(255, 215, 0, 0.5);
  background: radial-gradient(circle at center, rgba(255, 215, 0, 0.1), transparent 70%);
}
.pk-crown { font-size: 22px; position: absolute; top: -12px; filter: drop-shadow(0 0 6px rgba(255, 215, 0, 0.7)); }
.pk-name { color: #fff; font-size: 14px; font-weight: bold; text-align: center; line-height: 18px; }
.pk-score { color: #ccc; font-size: 22px; font-weight: bold; font-family: 'DIN', monospace; }
.pk-score.score-win { color: #ffd700; text-shadow: 0 0 10px rgba(255, 215, 0, 0.5); }
.pk-tag {
  font-size: 11px; padding: 1px 10px; border-radius: 8px;
  background: #333; color: #888;
}
.pk-tag.tag-win { background: #fe2c55; color: #fff; }

.pk-vs {
  align-self: center; font-size: 20px; font-weight: bold;
  color: #fe2c55;
  text-shadow: 0 0 12px rgba(254, 44, 85, 0.8);
  letter-spacing: 1px;
  padding: 0 4px;
}

/* Bar */
.pk-bar-wrap { margin-top: 16px; }
.pk-bar-labels { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.pk-bar-a { color: #fe2c55; font-size: 12px; font-weight: bold; }
.pk-bar-b { color: #ffd700; font-size: 12px; font-weight: bold; }
.pk-bar-gap { color: #8a6a3a; font-size: 11px; }
.pk-bar {
  display: flex; height: 8px; border-radius: 4px; overflow: hidden;
  background: #2a1010;
}
.pk-bar-fill-a {
  background: linear-gradient(90deg, #fe2c55, #ff6a85);
  transition: width 0.4s ease;
}
.pk-bar-fill-b {
  background: linear-gradient(90deg, #ffaa00, #ffd700);
  transition: width 0.4s ease;
}

.pk-footer { text-align: center; color: #5a3a1a; font-size: 12px; padding: 20px; }
</style>
