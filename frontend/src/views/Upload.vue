<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { uploadVideo } from '../api'

const router = useRouter()
const form = ref({ title: '', description: '', video_url: '', cover_url: '', tags: '', music: '原声' })

async function submit() {
  if (!form.value.video_url) {
    showToast('请输入视频地址')
    return
  }
  if (!form.value.title) {
    showToast('请填写标题')
    return
  }
  try {
    await uploadVideo(form.value)
    showSuccessToast('发布成功')
    router.push('/feed')
  } catch (e) {
    showToast(e.response?.data?.error || '发布失败')
  }
}
</script>

<template>
  <div class="upload-page">
    <van-nav-bar title="发布作品" left-arrow @click-left="router.back()" />
    <van-cell-group inset style="margin-top: 12px; background: #161616">
      <van-field v-model="form.video_url" label="视频地址" placeholder="https://...mp4" label-width="80" />
      <van-field v-model="form.cover_url" label="封面地址" placeholder="https://..." label-width="80" />
      <van-field v-model="form.title" label="标题" placeholder="给作品起个标题" label-width="80" />
      <van-field v-model="form.description" type="textarea" label="描述" placeholder="作品描述" rows="2" label-width="80" />
      <van-field v-model="form.tags" label="话题" placeholder="旅行,美食 (逗号分隔)" label-width="80" />
      <van-field v-model="form.music" label="音乐" placeholder="原声" label-width="80" />
    </van-cell-group>
    <div class="tip">
      <van-icon name="info-o" /> 演示模式：粘贴一个 mp4 视频地址即可发布（如 Google sample videos）
    </div>
    <div style="margin: 20px">
      <van-button type="primary" color="#fe2c55" block round @click="submit">发布</van-button>
    </div>
  </div>
</template>

<style scoped>
.upload-page { height: 100vh; overflow-y: auto; background: #000; color: #fff; }
.upload-page :deep(.van-cell-group) { border-radius: 12px; }
.upload-page :deep(.van-cell) { background: #161616 !important; color: #fff; }
.upload-page :deep(.van-field__label) { color: #ccc; }
.upload-page :deep(input), .upload-page :deep(textarea) { color: #fff !important; }
.tip { color: #888; font-size: 12px; padding: 12px 24px; line-height: 18px; }
</style>
