<script setup>
import { ref, computed } from 'vue'
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
const filters = [
  { value: 'none', label: '原图' },
  { value: 'vintage', label: '复古' },
  { value: 'warm', label: '暖阳' },
  { value: 'cool', label: '清冷' },
  { value: 'mono', label: '黑白' },
  { value: 'vivid', label: '鲜艳' },
]

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
      showSuccessToast('上传成功')
      router.push('/mine')
    } else {
      if (!form.value.video_url) { showToast('请输入视频地址'); uploading.value = false; return }
      await uploadVideo(form.value)
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

    <!-- ===================== Feature: 视频封面选择 (cover frame picker) ===================== -->
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
    </div>

    <van-cell-group inset style="margin-top: 12px; background: #161616">
      <van-field v-model="form.title" label="标题" placeholder="给作品起个标题" label-width="80" />
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
      <van-field v-model="form.music" label="音乐" placeholder="原声" label-width="80" />
      <div class="filter-row">
        <span class="fr-label">滤镜</span>
        <div class="fr-chips">
          <span v-for="f in filters" :key="f.value" class="fr-chip" :class="{ active: form.filter === f.value }" @click="form.filter = f.value">{{ f.label }}</span>
        </div>
      </div>
    </van-cell-group>

    <van-progress v-if="uploading" :percentage="progress" color="#fe2c55" style="margin: 12px 24px; width: calc(100% - 48px)" />
    <div style="margin: 20px">
      <van-button type="primary" color="#fe2c55" block round :loading="uploading" @click="submit">发布</van-button>
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
</style>
