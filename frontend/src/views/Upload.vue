<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
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
const mode = ref('file')
const file = ref(null)
const uploading = ref(false)
const progress = ref(0)
const form = ref({ title: '', description: '', video_url: '', cover_url: '', tags: '', music: '原声', filter: 'none' })
const filters = [
  { value: 'none', label: '原图' },
  { value: 'vintage', label: '复古' },
  { value: 'warm', label: '暖阳' },
  { value: 'cool', label: '清冷' },
  { value: 'mono', label: '黑白' },
  { value: 'vivid', label: '鲜艳' },
]

function onFile(item) {
  file.value = item.file
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
</style>
