<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getDuets, createDuet } from '../api'

const route = useRoute()
const router = useRouter()
const duets = ref([])
const loading = ref(true)
const showDuetDialog = ref(false)
const duetTitle = ref('')

onMounted(async () => {
  try { duets.value = await getDuets(route.params.id) } catch (e) { showToast('加载失败') } finally { loading.value = false }
})

async function doDuet() {
  try {
    await createDuet(route.params.id, { title: duetTitle.value })
    showSuccessToast('合拍成功')
    showDuetDialog.value = false
    duetTitle.value = ''
    duets.value = await getDuets(route.params.id)
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
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
      <van-button size="small" round color="#fe2c55" @click="showDuetDialog = true">我要合拍</van-button>
    </div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!duets.length" description="还没有合拍作品，快来第一个合拍吧" />
    <div v-else class="grid">
      <div v-for="v in duets" :key="v.id" class="grid-item" @click="router.push('/feed')">
        <img class="cover" :src="v.cover_url" />
        <div class="grid-title van-multi-ellipsis--l2">{{ v.title }}</div>
        <div class="grid-meta">
          <span><van-icon name="like-o" /> {{ fmtCount(v.likes) }}</span>
        </div>
      </div>
    </div>

    <van-dialog v-model:show="showDuetDialog" title="发起合拍" show-cancel-button @confirm="doDuet">
      <van-field v-model="duetTitle" placeholder="给合拍作品起个标题（可选）" style="margin: 12px" />
    </van-dialog>
  </div>
</template>

<style scoped>
.duet-page { height: 100vh; overflow-y: auto; background: #000; }
.banner { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; background: linear-gradient(135deg, #fe2c55, #25f4ee); }
.b-text { color: #fff; font-size: 15px; font-weight: bold; }
.loading { text-align: center; padding: 60px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 6px; }
.grid-item { background: #111; border-radius: 6px; overflow: hidden; }
.cover { width: 100%; aspect-ratio: 9/16; object-fit: cover; }
.grid-title { color: #fff; font-size: 12px; line-height: 16px; padding: 4px 6px; height: 32px; }
.grid-meta { color: #999; font-size: 11px; padding: 0 6px 6px; }
</style>
