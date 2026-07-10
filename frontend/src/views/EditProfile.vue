<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getProfile, updateProfile, uploadImage } from '../api'

const router = useRouter()
const form = ref({ nickname: '', avatar: '', bio: '' })
const loading = ref(true)

// ---- Feature: Profile bio editor enhancement (个人简介增强) ----
// Max length + template suggestions for the bio field. The character counter is
// derived reactively from the bio; the template popup toggles with showTemplates.
const BIO_MAX = 100
const BIO_TEMPLATES = [
  '热爱生活，记录美好✨',
  '分享快乐，传递正能量💪',
  '用镜头记录每一个精彩瞬间📸',
  '生活不止眼前的苟且📷',
  '有趣的灵魂万里挑一🌈',
]
const showTemplates = ref(false)
const bioCount = computed(() => (form.value.bio || '').length)

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

// onBioInput enforces the 100-character max by truncating overflow. This is the
// validation hook (in addition to the counter); the field itself has no native
// maxlength so the counter stays meaningful.
function onBioInput() {
  if ((form.value.bio || '').length > BIO_MAX) {
    form.value.bio = form.value.bio.slice(0, BIO_MAX)
  }
}

// useTemplate fills the bio field with a chosen suggestion and closes the popup.
function useTemplate(tpl) {
  form.value.bio = tpl.slice(0, BIO_MAX)
  showTemplates.value = false
  showSuccessToast('已应用模板')
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
        <van-field
          v-model="form.bio"
          label="简介"
          type="textarea"
          placeholder="介绍一下自己吧"
          rows="2"
          autosize
          @input="onBioInput"
        />
      </van-cell-group>
      <!-- ===================== Feature: Profile bio editor enhancement (个人简介增强) =====================
           Character counter + inspiration templates. The counter turns red when at
           the 100-char max; the ✨ button opens a popup of 5 bio templates. -->
      <div class="bio-tools">
        <span class="bio-templates-btn" @click="showTemplates = true">✨ 灵感模板</span>
        <span class="bio-counter" :class="{ over: bioCount >= BIO_MAX }">{{ bioCount }}/{{ BIO_MAX }}</span>
      </div>

      <!-- Bio templates popup -->
      <van-popup v-model:show="showTemplates" position="bottom" round>
        <div class="bio-tpl-panel">
          <div class="btp-head">✨ 灵感模板</div>
          <div class="btp-sub">选择一个模板快速填充简介</div>
          <div class="btp-list">
            <div
              v-for="(tpl, i) in BIO_TEMPLATES"
              :key="i"
              class="btp-item"
              @click="useTemplate(tpl)"
            >{{ tpl }}</div>
          </div>
        </div>
      </van-popup>
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

/* ===================== Feature: Profile bio editor enhancement (个人简介增强) ===================== */
/* Row below the bio field: template button on the left, char counter on the
   right. The counter turns theme-red when the 100-char cap is reached. */
.bio-tools {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 24px 0;
}
.bio-templates-btn {
  color: #fe2c55;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
}
.bio-templates-btn:active { opacity: 0.7; }
.bio-counter {
  color: #888;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.bio-counter.over { color: #fe2c55; }
/* Bottom-sheet templates popup */
.bio-tpl-panel { background: #161616; padding: 16px; }
.btp-head { text-align: center; color: #fff; font-size: 16px; font-weight: bold; }
.btp-sub { text-align: center; color: #888; font-size: 12px; margin: 4px 0 14px; }
.btp-list { display: flex; flex-direction: column; gap: 8px; }
.btp-item {
  color: #fff;
  font-size: 14px;
  padding: 14px 12px;
  background: #222;
  border-radius: 10px;
  cursor: pointer;
  border: 1px solid #2a2a2a;
}
.btp-item:active { background: #2c2c2c; border-color: #fe2c55; }
</style>
