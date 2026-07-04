<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getHotMusic } from '../api'

const router = useRouter()
const list = ref([])
const loading = ref(true)

onMounted(async () => { try { list.value = await getHotMusic() } catch (e) { showToast('加载失败') } finally { loading.value = false } })
</script>

<template>
  <div class="hm-page">
    <van-nav-bar title="热门音乐榜" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="banner">🎵 抖音热歌榜 · 使用最多</div>
    <div v-if="loading" class="loading"><van-loading color="#fe2c55" /></div>
    <van-empty v-else-if="!list.length" description="暂无数据" />
    <div v-else class="music-list">
      <div v-for="(m, i) in list" :key="i" class="music-item">
        <span class="mi-rank" :class="{ top: i < 3 }">{{ i + 1 }}</span>
        <div class="mi-info">
          <div class="mi-name">🎵 {{ m.music }}</div>
          <div class="mi-uses">{{ m.uses }}个作品使用</div>
        </div>
        <van-icon name="play-circle-o" size="24" color="#fe2c55" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.hm-page { height: 100vh; overflow-y: auto; background: #000; }
.banner { background: linear-gradient(135deg, #25f4ee, #fe2c55); color: #fff; text-align: center; padding: 16px; font-size: 16px; font-weight: bold; }
.loading { text-align: center; padding: 60px; }
.music-item { display: flex; align-items: center; gap: 14px; padding: 14px 16px; background: #161616; border-bottom: 1px solid #1a1a1a; }
.mi-rank { width: 28px; text-align: center; color: #666; font-weight: bold; font-size: 18px; }
.mi-rank.top { color: #fe2c55; font-size: 20px; }
.mi-info { flex: 1; }
.mi-name { color: #fff; font-size: 15px; }
.mi-uses { color: #888; font-size: 12px; margin-top: 2px; }
</style>
