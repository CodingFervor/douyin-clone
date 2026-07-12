<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { uploadVideo, uploadVideoFile, uploadImage } from '../api'

const uploadingCover = ref(false)
async function onUploadCover(item) {
  uploadingCover.value = true
  try {
    const res = await uploadImage(item.file)
    form.value.cover_url = res.url
    showToast('封面已上传')
  } catch (e) {
    showToast(e.response?.data?.error || '上传失败')
  } finally {
    uploadingCover.value = false
  }
}

// ===================== Feature: 音乐试听 (music preview beep) =====================
// A "🎵试听" button next to the music field. Tapping it plays a short 1-second
// Web Audio beep as a stand-in for previewing the chosen track (there is no real
// audio file in this demo). The button only appears once a music name is entered.
// The AudioContext is created lazily on first use (and resumed if suspended) to
// satisfy the browser's user-gesture requirement for audio playback.
let previewAudioCtx = null
const previewing = ref(false)

// hasMusicName — true when a non-empty (non-"原声") music name has been entered.
const hasMusicName = computed(() => {
  const m = (form.value.music || '').trim()
  return m.length > 0 && m !== '原声'
})

function previewMusic() {
  if (previewing.value) return
  try {
    if (!previewAudioCtx) {
      const AC = window.AudioContext || window.webkitAudioContext
      if (!AC) { showToast('当前浏览器不支持试听'); return }
      previewAudioCtx = new AC()
    }
    if (previewAudioCtx.state === 'suspended') previewAudioCtx.resume()
    const now = previewAudioCtx.currentTime
    // A pleasant 1s two-tone preview (a rising then falling chirp) so the beep
    // feels musical rather than like an error tone.
    const osc = previewAudioCtx.createOscillator()
    const gain = previewAudioCtx.createGain()
    osc.type = 'triangle'
    // Pitch rises 660→880Hz then falls back over the 1s duration.
    osc.frequency.setValueAtTime(660, now)
    osc.frequency.linearRampToValueAtTime(880, now + 0.4)
    osc.frequency.linearRampToValueAtTime(660, now + 0.95)
    // Envelope: quick attack, sustain, gentle decay to avoid a click.
    gain.gain.setValueAtTime(0, now)
    gain.gain.linearRampToValueAtTime(0.2, now + 0.03)
    gain.gain.setValueAtTime(0.2, now + 0.7)
    gain.gain.exponentialRampToValueAtTime(0.001, now + 1.0)
    osc.connect(gain)
    gain.connect(previewAudioCtx.destination)
    osc.start(now)
    osc.stop(now + 1.02)
    previewing.value = true
    osc.onended = () => { previewing.value = false }
    showToast('🎵 试听中...')
  } catch (e) {
    // Web Audio may be unavailable — fail silently with a friendly message.
    showToast('试听失败')
  }
}

const router = useRouter()
const route = useRoute()
const mode = ref('file')
const file = ref(null)
const uploading = ref(false)
const progress = ref(0)
// Pre-fill the music field when arriving from 音乐工作室 via the `music` query param.
const initialMusic = route.query.music || '原声'
// Pre-fill the tags field when arriving from 创作者中心 via the `tags` query param.
const initialTags = route.query.tags || ''
const form = ref({ title: '', description: '', video_url: '', cover_url: '', tags: initialTags, music: initialMusic, filter: 'none' })

// ===================== Feature: Upload draft auto-save (草稿自动保存) =====================
// The upload form is auto-saved to localStorage every 2s while the user is
// typing, so an accidental refresh or navigation doesn't lose their work. The
// text fields (title / description / tags / music / video_url / filter) are
// saved; the selected file blob and uploaded cover image are not (they can't be
// meaningfully serialized). On mount a saved draft is restored and a
// "📝草稿已保存" indicator is shown after each auto-save. Submitting or explicitly
// clearing clears the draft.
const DRAFT_KEY = 'dy_upload_draft'
const draftSaved = ref(false)    // true briefly after each auto-save
const draftRestored = ref(false) // true once when a draft is restored on mount
let draftTimer = null
let draftFlashTimer = null

// draftPayload returns the serializable subset of the form to persist.
function draftPayload() {
  const f = form.value
  return {
    title: f.title || '',
    description: f.description || '',
    video_url: f.video_url || '',
    tags: f.tags || '',
    music: f.music || '',
    filter: f.filter || 'none',
    mode: mode.value || 'file',
  }
}

// saveDraft persists the current form to localStorage and flashes the indicator.
function saveDraft() {
  const payload = draftPayload()
  // Don't save an empty draft — clear it instead so a fresh form stays fresh.
  const hasContent = Object.values(payload).some((v) => typeof v === 'string' && v.trim() && v !== 'none' && v !== 'file')
  try {
    if (!hasContent) {
      localStorage.removeItem(DRAFT_KEY)
      draftSaved.value = false
      return
    }
    localStorage.setItem(DRAFT_KEY, JSON.stringify(payload))
  } catch (e) {
    // localStorage may be unavailable — ignore.
  }
  // Flash the "📝草稿已保存" indicator for 1.6s.
  draftSaved.value = true
  if (draftFlashTimer) clearTimeout(draftFlashTimer)
  draftFlashTimer = setTimeout(() => { draftSaved.value = false }, 1600)
}

// scheduleDraftSave debounces the save by 2s after the last input, so we only
// write once the user pauses typing.
function scheduleDraftSave() {
  if (draftTimer) clearTimeout(draftTimer)
  draftTimer = setTimeout(() => {
    draftTimer = null
    saveDraft()
  }, 2000)
}

// restoreDraft loads a saved draft into the form on mount (without overwriting
// query-param pre-fills like music/tags that were just navigated to).
function restoreDraft() {
  let saved = null
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    saved = raw ? JSON.parse(raw) : null
  } catch (e) {
    saved = null
  }
  if (!saved) return
  // Restore text fields; only overwrite query-param defaults when the draft has
  // a value so navigating in from 音乐工作室 still pre-fills when the draft is empty.
  if (saved.title) form.value.title = saved.title
  if (saved.description) form.value.description = saved.description
  if (saved.video_url) { form.value.video_url = saved.video_url; mode.value = 'url' }
  if (saved.tags) form.value.tags = saved.tags
  if (saved.music) form.value.music = saved.music
  if (saved.filter) form.value.filter = saved.filter
  if (saved.mode) mode.value = saved.mode
  draftRestored.value = true
  showToast('📝草稿已恢复')
}

// clearUploadDraft removes the saved draft (called after a successful publish).
function clearUploadDraft() {
  try { localStorage.removeItem(DRAFT_KEY) } catch (e) {}
  draftSaved.value = false
  if (draftTimer) { clearTimeout(draftTimer); draftTimer = null }
}

// Watch the form fields + mode and debounce-save on every change while typing.
watch(
  () => [form.value.title, form.value.description, form.value.video_url, form.value.tags, form.value.music, form.value.filter, mode.value],
  () => { scheduleDraftSave() }
)

onMounted(() => {
  restoreDraft()
})

onUnmounted(() => {
  if (draftTimer) clearTimeout(draftTimer)
  if (draftFlashTimer) clearTimeout(draftFlashTimer)
})

// ===================== Feature: 视频封面选择 (cover frame picker) =====================
// After a video is chosen (file or URL), the user can scrub through a small
// preview and capture the current frame as the cover via a canvas snapshot.
const pickerVideoRef = ref(null)
const pickerVideoEl = ref(null)
const pickerTime = ref(0)
const pickerDuration = ref(0)

// A playable source URL: an object URL when a file is selected, or the pasted
// video URL. Empty until a video is available.
const coverSourceUrl = computed(() => {
  if (mode.value === 'file' && pickerVideoRef.value) return pickerVideoRef.value
  if (mode.value === 'url' && form.value.video_url) return form.value.video_url
  return ''
})

// Update the live <video> element ref after it mounts.
function setPickerVideo(el) {
  pickerVideoEl.value = el
}

function onPickerLoadedMetadata(e) {
  pickerDuration.value = e.target.duration || 0
  // Start at ~10% so a representative frame shows, clamped to valid range.
  const start = pickerDuration.value ? Math.min(pickerDuration.value * 0.1, pickerDuration.value) : 0
  pickerTime.value = start
  e.target.currentTime = start
}

function onTimeUpdate(e) {
  pickerTime.value = e.target.currentTime
}

// Two-way scrub: dragging the slider seeks the video; the video updates the
// slider while playing.
function onScrub(e) {
  const v = pickerVideoEl.value
  if (!v) return
  const t = Number(e.target.value)
  pickerTime.value = t
  v.currentTime = t
}

// Capture the current frame of the preview video to a JPEG data URL and use it
// as the cover. Falls back gracefully if the frame can't be read.
function captureFrame() {
  const v = pickerVideoEl.value
  if (!v || !v.videoWidth) { showToast('视频未就绪'); return }
  try {
    const canvas = document.createElement('canvas')
    canvas.width = v.videoWidth
    canvas.height = v.videoHeight
    const ctx = canvas.getContext('2d')
    ctx.drawImage(v, 0, 0, canvas.width, canvas.height)
    form.value.cover_url = canvas.toDataURL('image/jpeg', 0.8)
    showSuccessToast('已截取当前帧为封面')
  } catch (err) {
    showToast('截取失败，请重试')
  }
}

function clearCover() {
  form.value.cover_url = ''
}

// ===================== Feature: AI文案生成器 (caption generator) =====================
// Pure-frontend template-based caption suggestions derived from the current
// tags and music. No API call — the pool of templates is shuffled and 3 are
// picked at random. "换一批" re-rolls the selection.
const aiTemplates = [
  '{tag}也太赞了吧！{music}配上绝了🔥 #{tag}',
  '被这个{tag}治愈了～用{music}刚刚好 #{tag}',
  '今日份{tag}分享，{music}是灵魂！谁懂啊 #{tag}',
  '这个{tag}绝绝子，{music}一响直接封神 #{tag}',
  '{tag}的快乐你们体会到了吗？BGM：{music} #{tag}',
]

const aiShow = ref(false)
const aiSuggestions = ref([])

// Resolve the values to interpolate into the templates, applying sensible
// defaults when tags or music are empty.
function resolveContext() {
  const rawTag = (form.value.tags || '').trim()
  // If multiple tags are comma-separated, take the first as the representative one.
  const tag = rawTag ? rawTag.split(/[,，\s]+/).filter(Boolean)[0] : '生活'
  const music = (form.value.music || '').trim() || '原声'
  return { tag, music }
}

// Pick n distinct items at random from arr.
function sampleN(arr, n) {
  const pool = arr.slice()
  const out = []
  while (pool.length && out.length < n) {
    const idx = Math.floor(Math.random() * pool.length)
    out.push(pool.splice(idx, 1)[0])
  }
  return out
}

function generateSuggestions() {
  const { tag, music } = resolveContext()
  const picked = sampleN(aiTemplates, Math.min(3, aiTemplates.length))
  aiSuggestions.value = picked.map((t) => t.replace(/\{tag\}/g, tag).replace(/\{music\}/g, music))
}

function openAi() {
  generateSuggestions()
  aiShow.value = true
}

function closeAi() {
  aiShow.value = false
}

function applySuggestion(text) {
  form.value.title = text
  closeAi()
}

function reshuffleAi() {
  generateSuggestions()
}

// ===================== Feature: Upload preview mode (上传预览模式) =====================
// A 👁️预览 button (next to 发布) opens a full-screen mock of how the video will
// look in the feed: the chosen video/cover fills the screen with the title,
// description, and tags overlaid at the bottom, and an action rail on the right
// with all counts at 0 (the post hasn't been published yet). Two buttons let
// the creator go back to editing or confirm + publish directly.
const showPreview = ref(false)
// The media source used by the preview: the selected file's object URL when in
// file mode, otherwise the pasted video URL. Falls back to the cover image.
const previewSrc = computed(() => {
  if (mode.value === 'file' && pickerVideoRef.value) return pickerVideoRef.value
  if (mode.value === 'url' && form.value.video_url) return form.value.video_url
  return form.value.cover_url || ''
})
// Whether the preview has a real video to play (vs. falling back to the cover).
// True when a file is selected or a video URL is filled; the cover alone is an
// image and is rendered by a separate branch.
const previewHasVideo = computed(() => {
  if (mode.value === 'file') return !!file.value
  return !!form.value.video_url
})
// Parsed tags (comma-separated) for the preview overlay.
const previewTags = computed(() => {
  return (form.value.tags || '').split(/[,，]+/).map((t) => t.trim()).filter(Boolean)
})

// canPreview requires at least a title and a media source so the preview is meaningful.
const canPreview = computed(() => {
  return !!form.value.title && !!previewSrc.value
})

// openPreview gates on canPreview so an empty form can't open the preview.
function openPreview() {
  if (!form.value.title) { showToast('请先填写标题'); return }
  if (!previewSrc.value) { showToast('请先选择视频'); return }
  showPreview.value = true
  // Auto-play the preview video once it mounts.
  nextTick(() => {
    const el = document.querySelector('.preview-video')
    if (el) { el.muted = true; el.play().catch(() => {}) }
  })
}
function closePreview() {
  // Pause the preview video when leaving to avoid background audio.
  const el = document.querySelector('.preview-video')
  if (el) el.pause()
  showPreview.value = false
}
// confirmPublish closes the preview and proceeds with the normal submit flow.
async function confirmPublish() {
  showPreview.value = false
  await submit()
}
const filters = [
  { value: 'none', label: '原图' },
  { value: 'vintage', label: '复古' },
  { value: 'warm', label: '暖阳' },
  { value: 'cool', label: '清冷' },
  { value: 'mono', label: '黑白' },
  { value: 'vivid', label: '鲜艳' },
]

// ===================== Feature: 发布增强 (upload tips) =====================
// Expandable "拍摄建议" section with shooting tips for the creator.
const tipsOpen = ref(false)
const shootingTips = [
  '💡 竖屏拍摄效果更佳(9:16)',
  '💡 保持画面稳定，避免晃动',
  '💡 光线充足时画质更好',
  '💡 添加热门话题可获得更多推荐',
]

// ===================== Feature: 文案模板 (caption template gallery) =====================
// A "📋文案模板" button opens a gallery of 5 pre-made video templates covering
// the common creator verticals (旅行日记/美食探店/日常分享/穿搭展示/技能教学).
// Tapping a template pre-fills the title, tags, and description so the creator
// can publish faster. The gallery is a bottom-sheet popup; selecting a template
// closes the sheet and toasts a confirmation.
const VIDEO_TEMPLATES = [
  {
    id: 'travel',
    icon: '✈️',
    name: '旅行日记',
    title: '📍这趟旅程治愈了我｜绝美风景打卡',
    tags: '旅行,风景,打卡,vlog',
    description: '记录旅途中每一个心动瞬间，风景与人情都值得被珍藏。镜头里的远方，就是生活给我们的礼物🌍',
  },
  {
    id: 'food',
    icon: '🍜',
    name: '美食探店',
    title: '这家宝藏小店我吹爆！人均不到50吃到扶墙',
    tags: '美食,探店,种草,隐藏菜单',
    description: '深巷里的烟火气，藏在街角的宝藏美食。今天带大家打卡一家本地人才知道的神仙小店，必点菜单已整理，吃货们赶紧冲！',
  },
  {
    id: 'daily',
    icon: '☕',
    name: '日常分享',
    title: '普通一天的浪漫碎片🧩',
    tags: '日常,生活,记录,治愈',
    description: '把平凡的日子过成诗。一杯咖啡、一缕阳光、一段音乐，记录生活里那些微小却闪光的瞬间，温柔又治愈。',
  },
  {
    id: 'ootd',
    icon: '👗',
    name: '穿搭展示',
    title: '今日穿搭OOTD｜显瘦又显高的一套',
    tags: '穿搭,ootd,时尚,显瘦',
    description: '分享一套超百搭的日常穿搭，显瘦显高又舒适。详细单品信息在评论区，姐妹们可以根据自己的风格调整搭配哦✨',
  },
  {
    id: 'skill',
    icon: '🎓',
    name: '技能教学',
    title: '3分钟学会｜新手也能轻松上手',
    tags: '教程,干货,技能,学习',
    description: '干货预警！手把手教你掌握这项实用技能，从零基础到独立完成只需3分钟。建议收藏反复观看，有问题欢迎在评论区交流～',
  },
]
const showTemplates = ref(false) // gallery popup visibility

// applyTemplate pre-fills title + tags + description from the chosen template,
// then closes the gallery and surfaces a toast. Existing field values are
// overwritten so the template applies cleanly.
function applyTemplate(t) {
  form.value.title = t.title
  form.value.tags = t.tags
  form.value.description = t.description
  showTemplates.value = false
  showToast('已应用「' + t.name + '」模板')
}

function onFile(item) {
  // Revoke any previous object URL to avoid leaking blob URLs.
  if (pickerVideoRef.value && pickerVideoRef.value.startsWith('blob:')) {
    URL.revokeObjectURL(pickerVideoRef.value)
  }
  file.value = item.file
  pickerVideoRef.value = item.file ? URL.createObjectURL(item.file) : ''
  // Reset scrub state for the new video.
  pickerTime.value = 0
  pickerDuration.value = 0
  // Clear any stale captured cover so the user picks a fresh frame.
  form.value.cover_url = ''
}

async function submit() {
  if (!form.value.title) { showToast('请填写标题'); return }
  uploading.value = true
  progress.value = 0
  try {
    if (mode.value === 'file') {
      if (!file.value) { showToast('请选择视频文件'); uploading.value = false; return }
      const fd = new FormData()
      fd.append('file', file.value)
      fd.append('title', form.value.title)
      fd.append('description', form.value.description)
      fd.append('cover_url', form.value.cover_url)
      fd.append('tags', form.value.tags)
      fd.append('music', form.value.music)
      fd.append('filter', form.value.filter)
      await uploadVideoFile(fd, (e) => {
        if (e.total) progress.value = Math.round((e.loaded / e.total) * 100)
      })
      // Feature: 草稿自动保存 — clear the draft now that the upload succeeded.
      clearUploadDraft()
      showSuccessToast('上传成功')
      router.push('/mine')
    } else {
      if (!form.value.video_url) { showToast('请输入视频地址'); uploading.value = false; return }
      await uploadVideo(form.value)
      // Feature: 草稿自动保存 — clear the draft now that the publish succeeded.
      clearUploadDraft()
      showSuccessToast('发布成功')
      router.push('/mine')
    }
  } catch (e) {
    showToast(e.response?.data?.error || '发布失败')
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <div class="upload-page">
    <van-nav-bar title="发布作品" left-arrow @click-left="router.back()" />
    <van-tabs v-model:active="mode" color="#fe2c55" background="#161616" title-active-color="#fff" title-inactive-color="#888">
      <van-tab title="上传视频文件" name="file">
        <van-cell-group inset style="margin-top: 12px; background: #161616">
          <van-cell title="选择视频">
            <template #title>
              <van-uploader :after-read="onFile" accept="video/*" max-count="1">
                <van-button icon="video-o" size="small" round color="#fe2c55">选择视频</van-button>
              </van-uploader>
            </template>
            <template #value>
              <span v-if="file" class="file-name van-ellipsis">{{ file.name }}</span>
            </template>
          </van-cell>
        </van-cell-group>
      </van-tab>
      <van-tab title="填写视频地址" name="url">
        <van-cell-group inset style="margin-top: 12px; background: #161616">
          <van-field v-model="form.video_url" label="视频地址" placeholder="https://...mp4" label-width="80" />
        </van-cell-group>
      </van-tab>
    </van-tabs>

    <!-- ===================== Feature: 发布增强 (拍摄建议 expandable tips) ===================== -->
    <div class="tips-card">
      <div class="tips-head" @click="tipsOpen = !tipsOpen">
        <span class="tips-title">💡 拍摄建议</span>
        <van-icon :name="tipsOpen ? 'arrow-up' : 'arrow-down'" color="#fe2c55" size="16" />
      </div>
      <transition name="tips-slide">
        <ul v-show="tipsOpen" class="tips-list">
          <li v-for="(tip, i) in shootingTips" :key="i" class="tip-line">{{ tip }}</li>
        </ul>
      </transition>
    </div>
    <div v-if="coverSourceUrl" class="cover-picker">
      <div class="cp-title">选择封面</div>
      <div class="cp-row">
        <video
          :ref="setPickerVideo"
          class="cp-video"
          :src="coverSourceUrl"
          muted
          playsinline
          preload="metadata"
          @loadedmetadata="onPickerLoadedMetadata"
          @timeupdate="onTimeUpdate"
        ></video>
        <div class="cp-side">
          <div class="cp-time">{{ pickerTime.toFixed(1) }}s / {{ pickerDuration ? pickerDuration.toFixed(1) + 's' : '--' }}</div>
          <van-button size="small" round color="#fe2c55" icon="photograph" @click="captureFrame">截取当前帧</van-button>
          <div v-if="form.cover_url" class="cp-cover-wrap">
            <van-image width="70" height="100" radius="6" :src="form.cover_url" fit="cover" />
            <span class="cp-clear" @click="clearCover">清除</span>
          </div>
        </div>
      </div>
      <input
        v-if="pickerDuration"
        class="cp-seek"
        type="range"
        min="0"
        :max="pickerDuration"
        step="0.1"
        :value="pickerTime"
        @input="onScrub"
      />
      <div v-else class="cp-hint">正在加载视频…</div>
      <!-- ===================== Feature: 发布增强 (封面优化建议) ===================== -->
      <div class="cover-tip">💡 封面优化建议：选择画面清晰、主体突出的瞬间作为封面，可显著提升点击率</div>
    </div>

    <van-cell-group inset style="margin-top: 12px; background: #161616">
      <van-field v-model="form.title" label="标题" placeholder="给作品起个标题" label-width="80">
        <template #button>
          <!-- Feature: 文案模板 — opens the pre-made template gallery. -->
          <van-button size="small" round color="#25f4ee" @click="showTemplates = true">📋文案模板</van-button>
          <van-button size="small" round color="#fe2c55" @click="openAi">✨ AI文案</van-button>
        </template>
      </van-field>
      <van-field v-model="form.description" type="textarea" label="描述" placeholder="作品描述" rows="2" label-width="80" />
      <van-field label="封面图" :loading="uploadingCover">
        <template #input>
          <van-uploader :after-read="onUploadCover" accept="image/*" max-count="1" :preview-image="false">
            <van-button icon="photo-o" size="small" round color="#fe2c55">上传封面</van-button>
          </van-uploader>
          <van-image v-if="form.cover_url" width="50" height="80" radius="4" :src="form.cover_url" fit="cover" style="margin-left: 8px" />
        </template>
      </van-field>
      <van-field v-model="form.tags" label="话题" placeholder="旅行,美食 (逗号分隔)" label-width="80" />
      <van-field v-model="form.music" label="音乐" placeholder="原声" label-width="80">
        <!-- ===================== Feature: 音乐试听 (music preview beep) =====================
             "🎵试听" plays a 1s Web Audio beep once a music name is entered. The
             button is disabled while a preview is already playing. -->
        <template #button>
          <van-button
            v-if="hasMusicName"
            size="small"
            round
            color="#fe2c55"
            :loading="previewing"
            class="music-preview-btn"
            @click="previewMusic"
          >🎵试听</van-button>
        </template>
      </van-field>
      <div class="filter-row">
        <span class="fr-label">滤镜</span>
        <div class="fr-chips">
          <span v-for="f in filters" :key="f.value" class="fr-chip" :class="{ active: form.filter === f.value }" @click="form.filter = f.value">{{ f.label }}</span>
        </div>
      </div>
    </van-cell-group>

    <van-progress v-if="uploading" :percentage="progress" color="#fe2c55" style="margin: 12px 24px; width: calc(100% - 48px)" />
    <!-- ===================== Feature: 草稿自动保存 (draft auto-save indicator) =====================
         Flashes "📝草稿已保存" for ~1.6s after each debounced auto-save. Also shows a
         steady "📝已恢复草稿" badge when a draft was restored on mount. -->
    <div class="draft-indicator">
      <transition name="draft-flash">
        <span v-if="draftSaved" class="draft-saved">📝草稿已保存</span>
      </transition>
      <span v-if="draftRestored && !draftSaved" class="draft-restored">📝已恢复草稿</span>
    </div>
    <!-- ===================== Feature: 发布增强 (预计处理时间) ===================== -->
    <div class="process-time"><van-icon name="clock-o" color="#25f4ee" size="14" /> 预计处理时间: ~3秒</div>
    <!-- ===================== Feature: 上传预览模式 (upload preview) =====================
         👁️预览 sits next to 发布; opens a full-screen mock of how the post will
         look in the feed before publishing. -->
    <div class="publish-row">
      <van-button round :disabled="!canPreview" class="preview-btn" @click="openPreview">👁️预览</van-button>
      <van-button type="primary" color="#fe2c55" round :loading="uploading" class="publish-btn" @click="submit">发布</van-button>
    </div>

    <!-- ===================== Feature: AI文案生成器 popup ===================== -->
    <van-popup v-model:show="aiShow" round position="bottom" :style="{ background: '#161616' }">
      <div class="ai-wrap">
        <div class="ai-header">
          <span class="ai-title">✨ AI文案建议</span>
          <span class="ai-close" @click="closeAi">✕</span>
        </div>
        <div class="ai-cards">
          <div v-for="(s, i) in aiSuggestions" :key="i" class="ai-card" @click="applySuggestion(s)">
            <span class="ai-text">{{ s }}</span>
            <span class="ai-use">使用</span>
          </div>
        </div>
        <div class="ai-footer">
          <van-button size="small" plain round color="#fe2c55" icon="replay" @click="reshuffleAi">换一批</van-button>
        </div>
      </div>
    </van-popup>

    <!-- ===================== Feature: 文案模板 (caption template gallery) =====================
         A bottom-sheet gallery of 5 pre-made video templates. Each card shows the
         template's icon, name, and a preview of the title/tags it will fill.
         Tapping a card pre-fills title + tags + description and closes the sheet. -->
    <van-popup v-model:show="showTemplates" round position="bottom" :style="{ background: '#161616' }">
      <div class="tpl-wrap">
        <div class="tpl-header">
          <span class="tpl-title">📋 文案模板</span>
          <span class="tpl-close" @click="showTemplates = false">✕</span>
        </div>
        <div class="tpl-sub">选择模板快速填充标题、话题和描述</div>
        <div class="tpl-grid">
          <div
            v-for="t in VIDEO_TEMPLATES"
            :key="t.id"
            class="tpl-card"
            @click="applyTemplate(t)"
          >
            <div class="tpl-card-head">
              <span class="tpl-icon">{{ t.icon }}</span>
              <span class="tpl-name">{{ t.name }}</span>
            </div>
            <div class="tpl-preview">{{ t.title }}</div>
            <div class="tpl-tags">
              <span v-for="tag in t.tags.split(',')" :key="tag" class="tpl-tag">#{{ tag }}</span>
            </div>
            <div class="tpl-use">使用模板 ›</div>
          </div>
        </div>
      </div>
    </van-popup>

    <!-- ===================== Feature: 上传预览模式 (upload preview) =====================
         A full-screen mock of how the video will appear in the feed. The chosen
         video (or cover image fallback) fills the screen with the title,
         description and tags overlaid at the bottom and an action rail on the
         right with all counts at 0. Two buttons let the creator return to
         editing or confirm + publish. -->
    <div v-if="showPreview" class="preview-overlay">
      <div class="preview-stage">
        <!-- Full-screen media: plays the video when available, else the cover -->
        <video
          v-if="previewHasVideo"
          class="preview-video"
          :src="previewSrc"
          :poster="form.cover_url"
          loop
          muted
          playsinline
          autoplay
        ></video>
        <img v-else-if="form.cover_url" class="preview-cover" :src="form.cover_url" />
        <div v-else class="preview-empty">无预览内容</div>

        <!-- "预览模式" label top-left -->
        <div class="preview-mode-tag">👁️ 预览模式</div>

        <!-- Mock action rail (counts at 0 — the post is unpublished) -->
        <div class="preview-rail">
          <div class="pr-avatar-wrap">
            <div class="mood-ring">
              <img class="pr-avatar" src="https://via.placeholder.com/48" />
            </div>
          </div>
          <div class="pr-item"><van-icon name="like-o" color="#fff" size="32" /><span>0</span></div>
          <div class="pr-item"><van-icon name="chat-o" color="#fff" size="32" /><span>0</span></div>
          <div class="pr-item"><van-icon name="star-o" color="#fff" size="32" /><span>0</span></div>
          <div class="pr-item"><van-icon name="share-o" color="#fff" size="32" /><span>分享</span></div>
          <div class="pr-disc"><van-icon name="music-o" /></div>
        </div>

        <!-- Overlaid title / description / tags, feed-style -->
        <div class="preview-info">
          <div class="pr-author">@我</div>
          <div class="pr-title">{{ form.title || '未填写标题' }}</div>
          <div v-if="form.description" class="pr-desc">{{ form.description }}</div>
          <div v-if="previewTags.length" class="pr-tags">
            <span v-for="t in previewTags" :key="t" class="pr-tag">#{{ t }}</span>
          </div>
          <div class="pr-music"><van-icon name="music-o" size="14" /><span>{{ form.music || '原声' }}</span></div>
        </div>

        <!-- Bottom action buttons: return to editor or confirm + publish -->
        <div class="preview-actions">
          <van-button round class="pa-btn pa-back" @click="closePreview">返回编辑</van-button>
          <van-button round color="#fe2c55" class="pa-btn pa-confirm" @click="confirmPublish">确认发布</van-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.upload-page { height: 100vh; overflow-y: auto; background: #000; color: #fff; }
.upload-page :deep(.van-cell-group) { border-radius: 12px; }
.upload-page :deep(.van-cell) { background: #161616 !important; color: #fff; }
.upload-page :deep(.van-field__label) { color: #ccc; }
.upload-page :deep(input), .upload-page :deep(textarea) { color: #fff !important; }
.upload-page :deep(.van-tabs__nav) { background: #161616 !important; }
.file-name { max-width: 180px; color: #25f4ee; font-size: 12px; }
.filter-row { padding: 12px 16px; }
.fr-label { color: #ccc; font-size: 14px; }
.fr-chips { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.fr-chip { padding: 6px 14px; border-radius: 16px; background: #222; color: #fff; font-size: 13px; }
.fr-chip.active { background: #fe2c55; }

/* ===================== Feature: 视频封面选择 (cover frame picker) ===================== */
.cover-picker {
  margin: 12px 16px 0;
  padding: 12px;
  background: #161616;
  border-radius: 12px;
}
.cp-title { color: #fff; font-size: 14px; font-weight: bold; margin-bottom: 10px; }
.cp-row { display: flex; gap: 12px; }
.cp-video { width: 160px; height: auto; max-height: 220px; border-radius: 8px; background: #000; flex-shrink: 0; }
.cp-side { flex: 1; display: flex; flex-direction: column; gap: 10px; justify-content: center; min-width: 0; }
.cp-time { color: #25f4ee; font-size: 12px; }
.cp-cover-wrap { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.cp-clear { color: #fe2c55; font-size: 12px; cursor: pointer; }
.cp-seek {
  width: 100%;
  margin-top: 12px;
  accent-color: #fe2c55;
  height: 4px;
}
.cp-hint { color: #888; font-size: 12px; margin-top: 10px; text-align: center; }

/* ===================== Feature: AI文案生成器 (caption generator) ===================== */
.ai-wrap { padding: 20px 16px 24px; }
.ai-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.ai-title { color: #fff; font-size: 17px; font-weight: bold; }
.ai-close { color: #888; font-size: 18px; padding: 4px 8px; cursor: pointer; }
.ai-cards { display: flex; flex-direction: column; gap: 10px; }
.ai-card {
  display: flex; align-items: center; justify-content: space-between; gap: 10px;
  background: #222; border-radius: 10px; padding: 14px;
  border: 1px solid #2a2a2a; cursor: pointer; transition: border-color 0.2s, transform 0.1s;
}
.ai-card:active { transform: scale(0.99); }
.ai-card:hover { border-color: #fe2c55; }
.ai-text { color: #fff; font-size: 14px; line-height: 20px; flex: 1; }
.ai-use { color: #fe2c55; font-size: 12px; font-weight: bold; white-space: nowrap; }
.ai-footer { display: flex; justify-content: center; margin-top: 18px; }

/* ===================== Feature: 文案模板 (caption template gallery) =====================
   A bottom-sheet gallery of pre-made video templates. Each template is a tappable
   card showing its icon, name, a title preview, and its hashtags; selecting a
   card fills title + tags + description. */
.tpl-wrap { padding: 20px 16px 24px; }
.tpl-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.tpl-title { color: #fff; font-size: 17px; font-weight: bold; }
.tpl-close { color: #888; font-size: 18px; padding: 4px 8px; cursor: pointer; }
.tpl-sub { color: #888; font-size: 12px; margin-bottom: 16px; }
.tpl-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.tpl-card {
  background: #222;
  border: 1px solid #2a2a2a;
  border-radius: 12px;
  padding: 14px;
  cursor: pointer;
  transition: border-color 0.2s, transform 0.1s;
}
.tpl-card:active { transform: scale(0.98); }
.tpl-card:hover { border-color: #25f4ee; }
.tpl-card-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.tpl-icon { font-size: 20px; line-height: 1; }
.tpl-name { color: #fff; font-size: 14px; font-weight: bold; }
.tpl-preview {
  color: #ccc;
  font-size: 12px;
  line-height: 17px;
  margin-bottom: 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.tpl-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}
.tpl-tag {
  color: #25f4ee;
  font-size: 10px;
  background: rgba(37, 244, 238, 0.1);
  padding: 1px 6px;
  border-radius: 8px;
  line-height: 14px;
}
.tpl-use {
  color: #fe2c55;
  font-size: 11px;
  font-weight: 600;
  text-align: right;
}

/* ===================== Feature: 发布增强 (upload tips) ===================== */
.tips-card {
  margin: 12px 16px 0; padding: 12px 14px; background: #161616; border-radius: 12px;
}
.tips-head {
  display: flex; align-items: center; justify-content: space-between; cursor: pointer;
}
.tips-title { color: #fff; font-size: 14px; font-weight: bold; }
.tips-list { margin: 12px 0 0; padding: 0; list-style: none; }
.tip-line { color: #ccc; font-size: 13px; line-height: 24px; }
.cover-tip {
  color: #25f4ee; font-size: 11px; margin-top: 10px; line-height: 16px;
  background: rgba(37,244,238,0.08); padding: 8px 10px; border-radius: 8px;
}
.process-time {
  display: flex; align-items: center; justify-content: center; gap: 6px;
  color: #25f4ee; font-size: 12px; margin-top: 4px;
}

/* ===================== Feature: 草稿自动保存 (draft auto-save indicator) =====================
   A small centered indicator. "📝草稿已保存" flashes after each auto-save;
   "📝已恢复草稿" shows steadily when a draft was restored on mount. */
.draft-indicator {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 22px;
  margin-top: 6px;
}
.draft-saved {
  font-size: 12px;
  color: #34c759;
  background: rgba(52, 199, 89, 0.12);
  padding: 3px 12px;
  border-radius: 12px;
}
.draft-restored {
  font-size: 12px;
  color: #25f4ee;
  background: rgba(37, 244, 238, 0.1);
  padding: 3px 12px;
  border-radius: 12px;
}
/* flash transition for the saved indicator */
.draft-flash-enter-active, .draft-flash-leave-active { transition: opacity 0.25s ease; }
.draft-flash-enter-from, .draft-flash-leave-to { opacity: 0; }
/* expand/collapse transition for the tips list */
.tips-slide-enter-active, .tips-slide-leave-active { transition: all 0.25s ease; overflow: hidden; }
.tips-slide-enter-from, .tips-slide-leave-to { opacity: 0; max-height: 0; margin-top: 0; }
.tips-slide-enter-to, .tips-slide-leave-from { opacity: 1; max-height: 200px; }

/* ===================== Feature: 上传预览模式 (upload preview) ===================== */
/* Publish row: 👁️预览 + 发布 side by side */
.publish-row { display: flex; gap: 10px; margin: 20px; }
.preview-btn {
  flex: 1;
  color: #fff !important;
  background: #222 !important;
  border: 1px solid #333 !important;
}
.preview-btn:active { opacity: 0.8; }
.publish-btn { flex: 1.6; }
/* Full-screen overlay mocking the feed video card */
.preview-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: #000;
  display: flex;
  align-items: stretch;
  justify-content: center;
}
.preview-stage {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: #000;
}
.preview-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: #000;
}
.preview-cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.preview-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #888;
  font-size: 14px;
}
/* "预览模式" label */
.preview-mode-tag {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 12;
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  background: rgba(254, 44, 85, 0.85);
  padding: 4px 12px;
  border-radius: 12px;
}
/* Mock action rail on the right */
.preview-rail {
  position: absolute;
  right: 10px;
  bottom: 150px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  z-index: 10;
}
.pr-avatar-wrap { margin-bottom: 6px; }
/* A static mood ring (gray) so the mock mirrors the feed's avatar styling */
.mood-ring {
  position: relative;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  padding: 3px;
  background: #000;
}
.mood-ring::before {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  background: conic-gradient(from 0deg, #555, #888, #555, #555);
}
.pr-avatar {
  position: relative;
  z-index: 1;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  border: 2px solid #000;
  box-sizing: border-box;
}
.pr-item { display: flex; flex-direction: column; align-items: center; gap: 3px; }
.pr-item span { color: #fff; font-size: 12px; }
.pr-disc {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: #222;
  display: flex;
  align-items: center;
  justify-content: center;
}
.pr-disc .van-icon { color: #25f4ee; font-size: 24px; }
/* Overlaid info bottom-left */
.preview-info {
  position: absolute;
  left: 12px;
  right: 76px;
  bottom: 84px;
  z-index: 10;
}
.pr-author { color: #fff; font-size: 15px; font-weight: bold; margin-bottom: 6px; }
.pr-title { color: #fff; font-size: 14px; line-height: 20px; margin-bottom: 6px; }
.pr-desc { color: rgba(255,255,255,0.85); font-size: 13px; line-height: 18px; margin-bottom: 6px; }
.pr-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }
.pr-tag {
  color: #fff;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(254,44,85,0.75);
  line-height: 16px;
}
.pr-music { display: flex; align-items: center; gap: 6px; color: #fff; font-size: 12px; }
/* Bottom action buttons */
.preview-actions {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 20px;
  z-index: 13;
  display: flex;
  gap: 10px;
  padding: 0 16px;
  box-sizing: border-box;
}
.pa-btn { flex: 1; }
.pa-back {
  color: #fff !important;
  background: rgba(0,0,0,0.6) !important;
  border: 1px solid rgba(255,255,255,0.35) !important;
}
.pa-back :deep(.van-button__text) { color: #fff; }

/* ===================== Feature: 音乐试听 (music preview beep) ===================== */
.music-preview-btn { flex-shrink: 0; }
</style>
