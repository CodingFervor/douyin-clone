<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getDuets, createDuet, getVideo, getProfile } from '../api'

const route = useRoute()
const router = useRouter()
const duets = ref([])
const loading = ref(true)
const showDuetDialog = ref(false)
const duetTitle = ref('')

// ===================== Feature: 合拍分屏预览 (duet split-screen preview) =====================
// Before creating a duet, the user sees a preview of what the split layout will
// look like: the original video's cover on the left, their own avatar placeholder
// on the right, separated by a vertical divider. They confirm with "开始合拍".
const showPreview = ref(false)
const previewLoading = ref(false)
const originVideo = ref(null)   // the original video being duetted (for cover + title)
const me = ref(null)            // the current user (for avatar placeholder)

onMounted(async () => {
  try { duets.value = await getDuets(route.params.id) } catch (e) { showToast('加载失败') } finally { loading.value = false }
  // Lazily fetch the original video + current user so the split-screen preview can
  // render the left cover and the right avatar without an extra click delay.
  getVideo(route.params.id).then((v) => { originVideo.value = v }).catch(() => {})
  getProfile().then((u) => { me.value = u }).catch(() => {})
})

// Open the split-screen preview overlay (instead of creating immediately).
// This is the entry point wired to the "我要合拍" button.
function openPreview() {
  showPreview.value = true
}

// Confirm the preview: actually create the duet via the API, then refresh the list.
async function doDuet() {
  previewLoading.value = true
  try {
    await createDuet(route.params.id, { title: duetTitle.value })
    showSuccessToast('合拍成功')
    showPreview.value = false
    duetTitle.value = ''
    duets.value = await getDuets(route.params.id)
  } catch (e) {
    showToast('请先登录')
    showPreview.value = false
    router.push('/login')
  } finally {
    previewLoading.value = false
  }
}

function fmtCount(n) {
  if (n >= 10000) return (n / 10000).toFixed(1) + 'w'
  return String(n)
}
</script>

<template>
  <div class="duet-page">
    <van-nav-bar title="合拍作品" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="banner">
      <div class="b-text">🎬 和TA合拍，一起创作</div>
      <van-button size="small" round color="#fe2c55" @click="openPreview">我要合拍</van-button>
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!duets.length" description="还没有合拍作品，快来第一个合拍吧" />
    <div v-else class="grid">
      <div v-for="v in duets" :key="v.id" class="grid-item" @click="router.push('/feed')">
        <!-- Feature: 合拍 badge — duet videos (parent_id > 0) show a red 合拍 pill. -->
        <span v-if="v.parent_id && v.parent_id > 0" class="duet-badge">合拍</span>
        <img class="cover" :src="v.cover_url" />
        <div class="grid-title van-multi-ellipsis--l2">{{ v.title }}</div>
        <div class="grid-meta">
          <span><van-icon name="like-o" /> {{ fmtCount(v.likes) }}</span>
        </div>
      </div>
    </div>

    <!-- Legacy simple dialog kept for accessibility / quick title entry fallback -->
    <van-dialog v-model:show="showDuetDialog" title="发起合拍" show-cancel-button @confirm="doDuet">
      <van-field v-model="duetTitle" placeholder="给合拍作品起个标题（可选）" style="margin: 12px" />
    </van-dialog>

    <!-- ===================== Feature: 合拍分屏预览 (split-screen preview overlay) ===================== -->
    <van-popup v-model:show="showPreview" position="bottom" round closeable close-icon-position="top-left" :style="{ height: '88%' }">
      <div class="preview-panel">
        <div class="preview-head">合拍预览</div>
        <div class="preview-sub">查看分屏效果，准备好就开始合拍</div>

        <!-- Split-screen stage: 9:16 aspect, left = original, right = my duet -->
        <div class="split-stage">
          <!-- Left: original video cover (or mini player poster) -->
          <div class="split-side split-left">
            <img v-if="originVideo" class="split-cover" :src="originVideo.cover_url" />
            <div v-else class="split-cover split-cover-ph"><van-loading color="#fe2c55" /></div>
            <span class="split-label">原视频</span>
          </div>

          <!-- Vertical divider -->
          <div class="split-divider"></div>

          <!-- Right: my duet placeholder with avatar -->
          <div class="split-side split-right">
            <div class="split-cover split-mine">
              <img v-if="me && me.avatar" class="mine-avatar" :src="me.avatar" />
              <van-icon v-else name="user-o" size="48" color="#fe2c55" />
              <div class="mine-hint">你的合拍</div>
            </div>
            <span class="split-label">我的合拍</span>
          </div>
        </div>

        <!-- Optional title field -->
        <van-field v-model="duetTitle" placeholder="给合拍作品起个标题（可选）" class="duet-title-field" />

        <!-- Action buttons -->
        <div class="preview-actions">
          <van-button round block class="cancel-btn" @click="showPreview = false">取消</van-button>
          <van-button round block color="#fe2c55" :loading="previewLoading" @click="doDuet">开始合拍</van-button>
        </div>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.duet-page { height: 100vh; overflow-y: auto; background: #000; }
.banner { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; background: linear-gradient(135deg, #fe2c55, #25f4ee); }
.b-text { color: #fff; font-size: 15px; font-weight: bold; }
.loading { text-align: center; padding: 60px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.grid-item { position: relative; background: #111; border-radius: 6px; overflow: hidden; }
.cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.grid-title { color: #fff; font-size: 12px; line-height: 16px; padding: 4px 6px; height: 32px; }
.grid-meta { color: #999; font-size: 11px; padding: 0 6px 6px; }

/* Feature: 合拍 badge on duet videos */
.duet-badge {
  position: absolute;
  top: 6px;
  left: 6px;
  z-index: 2;
  background: linear-gradient(135deg, #fe2c55, #ff5c7a);
  color: #fff;
  font-size: 10px;
  font-weight: bold;
  padding: 1px 7px;
  border-radius: 8px;
  line-height: 14px;
  box-shadow: 0 1px 4px rgba(254, 44, 85, 0.5);
}

/* ===================== Feature: 合拍分屏预览 styles ===================== */
.preview-panel {
  background: #161616;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 16px;
  gap: 14px;
}
.preview-head { color: #fff; font-size: 18px; font-weight: bold; }
.preview-sub { color: #888; font-size: 12px; margin-top: -8px; }

/* Split-screen stage: keeps the 9:16 duet proportion, two columns + divider */
.split-stage {
  position: relative;
  width: 100%;
  max-width: 300px;
  aspect-ratio: 9 / 16;
  display: flex;
  border-radius: 12px;
  overflow: hidden;
  background: #000;
  box-shadow: 0 4px 24px rgba(254, 44, 85, 0.25);
}
.split-side { flex: 1; position: relative; display: flex; align-items: center; justify-content: center; min-width: 0; }
.split-cover { width: 100%; height: 100%; object-fit: cover; }
.split-cover-ph { display: flex; align-items: center; justify-content: center; background: #1a1a1a; }

/* Vertical divider line in the middle */
.split-divider {
  width: 2px;
  flex: 0 0 2px;
  background: linear-gradient(to bottom, transparent, #fe2c55, transparent);
  box-shadow: 0 0 8px rgba(254, 44, 85, 0.6);
}

/* Right side — my duet placeholder */
.split-mine {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: linear-gradient(160deg, #1a1a1a, #2a0a12);
}
.mine-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  border: 2px solid #fe2c55;
  box-shadow: 0 0 12px rgba(254, 44, 85, 0.5);
}
.mine-hint { color: rgba(255,255,255,0.7); font-size: 12px; }

/* Per-side labels (原视频 / 我的合拍) overlaid at the bottom of each half */
.split-label {
  position: absolute;
  bottom: 8px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0,0,0,0.55);
  color: #fff;
  font-size: 11px;
  padding: 2px 10px;
  border-radius: 10px;
  white-space: nowrap;
}

/* Optional title input */
.duet-title-field {
  width: 100%;
  background: #1f1f1f;
  border-radius: 8px;
}
.duet-title-field :deep(input) { color: #fff; }

/* Confirm / cancel buttons */
.preview-actions { display: flex; gap: 10px; width: 100%; margin-top: auto; }
.preview-actions .van-button { flex: 1; }
.cancel-btn { background: #2a2a2a; color: #fff; border: none; }
.cancel-btn :deep(.van-button__text) { color: #fff; }
</style>
