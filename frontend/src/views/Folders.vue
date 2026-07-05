<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getFolders, createFolder } from '../api'

const router = useRouter()
const folders = ref([])
const loading = ref(true)
const showCreate = ref(false)
const folderName = ref('')

onMounted(async () => { try { folders.value = await getFolders() } catch (e) { showToast('加载失败') } finally { loading.value = false } })

async function doCreate() {
  if (!folderName.value.trim()) { showToast('请输入名称'); return }
  try { const f = await createFolder(folderName.value, ''); folders.value.unshift(f); showCreate.value = false; folderName.value = ''; showSuccessToast('已创建') }
  catch (e) { showToast('请先登录') }
}
</script>

<template>
  <div class="fo-page">
    <van-nav-bar title="我的收藏夹" left-arrow @click-left="router.back()" fixed placeholder>
      <template #right><van-icon name="plus" size="20" color="#fe2c55" @click="showCreate = true" /></template>
    </van-nav-bar>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!folders.length" description="还没有收藏夹">
      <van-button type="primary" color="#fe2c55" round @click="showCreate = true">新建收藏夹</van-button>
    </van-empty>
    <div v-else class="folder-list">
      <div v-for="f in folders" :key="f.id" class="folder-card" @click="router.push('/mine')">
        <div class="fc-cover">{{ f.cover_url ? '' : '📁' }}<img v-if="f.cover_url" :src="f.cover_url" /></div>
        <div class="fc-info">
          <div class="fc-name">{{ f.name }}</div>
          <div class="fc-count">{{ f.count }}个作品</div>
        </div>
      </div>
    </div>
    <van-dialog v-model:show="showCreate" title="新建收藏夹" show-cancel-button @confirm="doCreate">
      <van-field v-model="folderName" placeholder="收藏夹名称" style="margin: 12px" />
    </van-dialog>
  </div>
</template>

<style scoped>
.fo-page { height: 100vh; overflow-y: auto; background: #000; }
.loading { text-align: center; padding: 60px; }
.folder-list { padding: 12px; }
.folder-card { display: flex; align-items: center; gap: 12px; padding: 12px; background: #161616; border-radius: 10px; margin-bottom: 8px; }
.fc-cover { width: 56px; height: 56px; border-radius: 8px; background: #222; display: flex; align-items: center; justify-content: center; font-size: 28px; overflow: hidden; flex-shrink: 0; }
.fc-cover img { width: 100%; height: 100%; object-fit: cover; }
.fc-info { flex: 1; }
.fc-name { color: #fff; font-size: 15px; }
.fc-count { color: #888; font-size: 12px; margin-top: 2px; }
</style>
