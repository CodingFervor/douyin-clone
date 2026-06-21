<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const showTabbar = computed(() => route.meta.tab !== undefined)
const active = computed(() => route.meta.tab ?? 0)
const tabs = [
  { name: 'feed', icon: 'home-o', label: '首页' },
  { name: 'discover', icon: 'search', label: '发现' },
  { name: 'upload', icon: 'add-o', label: '发布' },
  { name: 'messages', icon: 'chat-o', label: '消息' },
  { name: 'mine', icon: 'contact', label: '我' },
]
</script>

<template>
  <div class="app-wrap">
    <router-view v-slot="{ Component }">
      <keep-alive include="Discover,Messages">
        <component :is="Component" />
      </keep-alive>
    </router-view>
    <van-tabbar v-if="showTabbar" v-model="active" route active-color="#fe2c55" inactive-color="#999" class="dy-tabbar">
      <van-tabbar-item v-for="t in tabs" :key="t.name" :to="{ name: t.name }" :icon="t.icon">{{ t.label }}</van-tabbar-item>
    </van-tabbar>
  </div>
</template>

<style scoped>
.app-wrap { height: 100vh; overflow: hidden; }
.dy-tabbar { background: #161616; }
</style>
