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

// ===================== Feature: Profile stats radar chart (主页数据雷达图) =====================
// A CSS/SVG pentagon radar visualizing 5 normalized (0-100) stats:
//   粉丝力  — followers
//   创作力  — video count
//   互动力  — likes
//   影响力  — followers × following (reach)
//   活跃度  — level
// Normalization is log-based so small and large accounts both map to a readable
// spread (no external chart library used — pure inline SVG).
const RADAR_AXES = ['粉丝力', '创作力', '互动力', '影响力', '活跃度']
// norm maps a raw count to 0-100 using a log scale capped at `cap` (saturation).
// 0 → 0, cap → 100, with smooth log growth in between so a brand-new account
// isn't pinned at the floor.
function normLog(value, cap) {
  const v = Math.max(0, value || 0)
  if (v <= 0) return 0
  if (cap <= 0) return 100
  if (v >= cap) return 100
  // log1p keeps very small values visible while large values saturate.
  return Math.round((Math.log1p(v) / Math.log1p(cap)) * 100)
}

// The 5 normalized axis values for the current user. Computed once user data is
// loaded; capped so the radar stays readable for popular accounts.
const radarValues = computed(() => {
  const u = user.value || {}
  const followers = u.followers_count || 0
  const following = u.following_count || 0
  const likes = u.likes_count || 0
  const videoCount = videos.value.length
  const level = u.level || 0
  return [
    normLog(followers, 100000),               // 粉丝力 — cap 10w followers
    normLog(videoCount, 200),                  // 创作力 — cap 200 videos
    normLog(likes, 1000000),                   // 互动力 — cap 100w likes
    normLog(followers * following, 5000000),   // 影响力 — followers × following
    normLog(level, 60),                        // 活跃度 — level (王者 ~ 60)
  ]
})

// Geometry: a pentagon centered in a 200×200 viewBox, radius 70, with the first
// vertex pointing up. Points start at the top and go clockwise.
const RADAR_CX = 100
const RADAR_CY = 100
const RADAR_R = 70
// angleFor returns the (x, y) of axis i at a given fraction (0-1) of the radius.
function radarPoint(i, fraction) {
  // -90deg offset so axis 0 points straight up.
  const angle = (Math.PI * 2 * i) / 5 - Math.PI / 2
  const r = RADAR_R * fraction
  return {
    x: RADAR_CX + r * Math.cos(angle),
    y: RADAR_CY + r * Math.sin(angle),
  }
}
// The outer pentagon vertices (the 100% reference frame).
const radarOuter = RADAR_AXES.map((_, i) => {
  const p = radarPoint(i, 1)
  return `${p.x.toFixed(1)},${p.y.toFixed(1)}`
}).join(' ')
// The data polygon points, scaled by each axis value.
const radarData = computed(() =>
  radarValues.value
    .map((val, i) => {
      const p = radarPoint(i, val / 100)
      return `${p.x.toFixed(1)},${p.y.toFixed(1)}`
    })
    .join(' ')
)
// Concentric reference rings at 25/50/75/100% so the data shape has context.
const radarRings = [0.25, 0.5, 0.75, 1].map((f) =>
  RADAR_AXES.map((_, i) => {
    const p = radarPoint(i, f)
    return `${p.x.toFixed(1)},${p.y.toFixed(1)}`
  }).join(' ')
)
// Axis label positions sit just outside the outer pentagon.
const radarLabels = RADAR_AXES.map((label, i) => {
  const p = radarPoint(i, 1.28)
  return { label, x: p.x, y: p.y, val: radarValues.value[i] }
})

// ===================== Feature: Profile visit counter (主页访客统计) =====================
// A visitor-stats card on the profile page. The total + daily counts are
// deterministic per user (derived from a hash of the user id) so the same
// profile always shows the same numbers (demo range 100–99999). A row of 3–5
// random recent-visitor avatars (from dicebear) sits under the counts; a
// "查看全部" link surfaces a toast.
// hashId turns a user id into a stable 32-bit integer (FNV-1a-ish).
function hashId(id) {
  const s = String(id == null ? '' : id)
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return Math.abs(h)
}
// totalVisitors — deterministic in 100–99999 from the user id hash.
const totalVisitors = computed(() => 100 + (hashId(user.value?.id) % 99900))
// todayVisitors — deterministic in 1–999 (a slice of the total, day-keyed so it
// shifts day to day but stays stable within a day).
const todayVisitors = computed(() => {
  const base = hashId(user.value?.id)
  const d = new Date()
  const dayKey = d.getFullYear() * 1000 + (d.getMonth() + 1) * 50 + d.getDate()
  return 1 + ((base ^ dayKey) % 999)
})
// recentVisitorAvatars — 3–5 deterministic dicebear avatars seeded by the user id.
const recentVisitorAvatars = computed(() => {
  const base = hashId(user.value?.id)
  const count = 3 + (base % 3) // 3, 4, or 5
  const out = []
  for (let i = 0; i < count; i++) {
    const seed = (base + i * 2654435761) >>> 0
    out.push(`https://api.dicebear.com/7.x/avataaars/svg?seed=u${seed}`)
  }
  return out
})
// showVisitors surfaces the "查看访客详情" toast from the 查看全部 link.
function showVisitors() {
  showToast('查看访客详情')
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

      <!-- ===================== Feature: Profile stats radar chart (主页数据雷达图) =====================
           A pure CSS/SVG pentagon radar visualizing 5 normalized stats. The filled
           semi-transparent area is the user's profile; concentric rings + axis
           labels give context. Shown just below the user info section. -->
      <div class="radar-card">
        <div class="radar-title">📊 数据雷达</div>
        <svg class="radar-svg" viewBox="0 0 200 200">
          <!-- Concentric reference rings -->
          <polygon v-for="(ring, ri) in radarRings" :key="'ring' + ri" class="radar-ring" :points="ring" />
          <!-- Spokes from center to each outer vertex -->
          <line
            v-for="(lbl, i) in radarLabels"
            :key="'spoke' + i"
            class="radar-spoke"
            :x1="RADAR_CX"
            :y1="RADAR_CY"
            :x2="radarPoint(i, 1).x"
            :y2="radarPoint(i, 1).y"
          />
          <!-- Filled data area -->
          <polygon class="radar-fill" :points="radarData" />
          <!-- Data vertices -->
          <circle
            v-for="(val, i) in radarValues"
            :key="'dot' + i"
            class="radar-dot"
            :cx="radarPoint(i, val / 100).x"
            :cy="radarPoint(i, val / 100).y"
            r="3"
          />
          <!-- Axis labels with value -->
          <text
            v-for="(lbl, i) in radarLabels"
            :key="'lbl' + i"
            class="radar-label"
            :x="lbl.x"
            :y="lbl.y"
            text-anchor="middle"
            dominant-baseline="middle"
          >{{ lbl.label }}</text>
        </svg>
        <div class="radar-legend">
          <span v-for="(axis, i) in RADAR_AXES" :key="'leg' + i" class="radar-legend-item">
            <span class="radar-legend-name">{{ axis }}</span>
            <span class="radar-legend-val">{{ radarValues[i] }}</span>
          </span>
        </div>
      </div>

      <!-- ===================== Feature: Profile visit counter (主页访客统计) =====================
           A visitor-stats card: today's visitors, total visitors, a row of recent
           visitor avatars (dicebear), and a "查看全部" link that toasts. -->
      <div class="visitor-card">
        <div class="vc-head">
          <span class="vc-title">👁️ 访客</span>
          <span class="vc-all" @click="showVisitors">查看全部 &rsaquo;</span>
        </div>
        <div class="vc-stats">
          <div class="vc-stat">
            <div class="vc-num">{{ fmtCount(todayVisitors) }}</div>
            <div class="vc-lbl">今日访客</div>
          </div>
          <div class="vc-divider"></div>
          <div class="vc-stat">
            <div class="vc-num">{{ fmtCount(totalVisitors) }}</div>
            <div class="vc-lbl">总访客</div>
          </div>
        </div>
        <div class="vc-recent">
          <span class="vc-recent-label">最近访客</span>
          <div class="vc-avatars">
            <img
              v-for="(a, i) in recentVisitorAvatars"
              :key="i"
              class="vc-avatar"
              :style="{ zIndex: recentVisitorAvatars.length - i }"
              :src="a"
            />
            <span class="vc-more">…</span>
          </div>
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

/* ===================== Feature: Profile stats radar chart (主页数据雷达图) ===================== */
/* A card holding the SVG pentagon radar + a legend of the normalized values. */
.radar-card {
  background: #161616;
  margin: 12px;
  padding: 16px 12px;
  border-radius: 14px;
}
.radar-title { color: #fff; font-size: 14px; font-weight: bold; text-align: center; margin-bottom: 8px; }
.radar-svg { width: 100%; max-width: 300px; height: auto; display: block; margin: 0 auto; }
/* Concentric reference rings — subtle gray pentagons */
.radar-ring { fill: none; stroke: rgba(255,255,255,0.12); stroke-width: 1; }
/* Spokes from center to each vertex */
.radar-spoke { stroke: rgba(255,255,255,0.12); stroke-width: 1; }
/* Filled semi-transparent data area — themed red */
.radar-fill {
  fill: rgba(254,44,85,0.28);
  stroke: #fe2c55;
  stroke-width: 2;
  stroke-linejoin: round;
}
/* Data vertex dots */
.radar-dot { fill: #fe2c55; }
/* Axis labels */
.radar-label { fill: rgba(255,255,255,0.85); font-size: 11px; font-weight: 600; }
/* Legend — 5 normalized values in a row */
.radar-legend {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px 14px;
  margin-top: 12px;
}
.radar-legend-item { display: flex; flex-direction: column; align-items: center; }
.radar-legend-name { color: #888; font-size: 11px; }
.radar-legend-val { color: #fe2c55; font-size: 14px; font-weight: bold; }

/* ===================== Feature: 主页访客统计 (profile visit counter) ===================== */
/* A card below the radar showing today/total visitor counts, recent avatars,
   and a 查看全部 link that toasts. */
.visitor-card {
  background: #161616;
  margin: 12px;
  padding: 14px 16px;
  border-radius: 14px;
}
.vc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.vc-title { color: #fff; font-size: 14px; font-weight: bold; }
.vc-all { color: #fe2c55; font-size: 12px; cursor: pointer; }
.vc-all:active { opacity: 0.6; }
.vc-stats { display: flex; align-items: center; justify-content: center; gap: 28px; }
.vc-stat { display: flex; flex-direction: column; align-items: center; }
.vc-num { color: #fff; font-size: 20px; font-weight: bold; font-variant-numeric: tabular-nums; }
.vc-lbl { color: #888; font-size: 11px; margin-top: 2px; }
.vc-divider { width: 1px; height: 26px; background: #2a2a2a; }
/* Recent visitor avatars row */
.vc-recent { display: flex; align-items: center; gap: 10px; margin-top: 12px; }
.vc-recent-label { color: #888; font-size: 12px; }
.vc-avatars { display: flex; align-items: center; }
.vc-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid #161616;
  background: #222;
  margin-left: -8px;
  object-fit: cover;
}
.vc-avatar:first-child { margin-left: 0; }
.vc-more {
  color: #888;
  font-size: 14px;
  font-weight: bold;
  margin-left: 4px;
}
</style>
