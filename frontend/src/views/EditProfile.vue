<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getProfile, updateProfile, uploadImage } from '../api'

const router = useRouter()
const form = ref({ nickname: '', avatar: '', bio: '' })
const loading = ref(true)

onMounted(async () => {
  try {
    const u = await getProfile()
    form.value = { nickname: u.nickname || '', avatar: u.avatar || '', bio: u.bio || '' }
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})

async function onUploadAvatar(item) {
  try {
    const res = await uploadImage(item.file)
    form.value.avatar = res.url
    showSuccessToast('头像已更新')
  } catch (e) {
    showToast('上传失败')
  }
}

async function save() {
  try {
    await updateProfile(form.value)
    showSuccessToast('资料已更新')
    router.back()
  } catch (e) {
    showToast(e.response?.data?.error || '保存失败')
  }
}
</script>

<template>
  <div class="profile-page">
    <van-nav-bar title="编辑资料" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <div v-else class="form-body">
      <div class="avatar-cell">
        <span>头像</span>
        <div class="avatar-right">
          <img class="avatar" :src="form.avatar || 'https://via.placeholder.com/60'" />
          <van-uploader :after-read="onUploadAvatar" accept="image/*" :preview-image="false">
            <van-button size="small" plain round color="#fe2c55">更换</van-button>
          </van-uploader>
        </div>
      </div>
      <van-cell-group inset>
        <van-field v-model="form.nickname" label="昵称" placeholder="请输入昵称" />
        <van-field v-model="form.bio" label="简介" type="textarea" placeholder="介绍一下自己吧" rows="2" autosize />
      </van-cell-group>
      <div class="save-btn"><van-button block round color="#fe2c55" @click="save">保存</van-button></div>
    </div>
  </div>
</template>

<style scoped>
.profile-page { min-height: 100vh; background: #000; }
.loading { text-align: center; padding: 80px; }
.form-body { padding-top: 12px; }
.avatar-cell { display: flex; justify-content: space-between; align-items: center; padding: 16px; margin: 0 16px; background: #161616; border-radius: 8px; margin-bottom: 12px; }
.avatar-cell span { color: #fff; font-size: 15px; }
.avatar-right { display: flex; align-items: center; gap: 12px; }
.avatar { width: 56px; height: 56px; border-radius: 50%; object-fit: cover; }
:deep(.van-cell-group--inset) { margin: 0 16px; background: #161616; border-radius: 8px; }
:deep(.van-cell) { background: #161616; }
:deep(.van-cell__title), :deep(.van-field__control) { color: #fff; }
:deep(.van-field__control::placeholder) { color: #666; }
.save-btn { padding: 24px 16px; }
</style>
