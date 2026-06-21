<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { login, register } from '../api'

const route = useRoute()
const router = useRouter()
const mode = ref('login')
const username = ref('admin')
const password = ref('admin123')
const nickname = ref('')

async function submit() {
  if (!username.value || !password.value) { showToast('请输入用户名和密码'); return }
  try {
    const res = mode.value === 'login' ? await login(username.value, password.value) : await register({ username: username.value, password: password.value, nickname: nickname.value })
    localStorage.setItem('dy_token', res.token)
    showSuccessToast(mode.value === 'login' ? '登录成功' : '注册成功')
    router.replace(route.query.redirect || '/mine')
  } catch (e) { showToast(e.response?.data?.error || '操作失败') }
}
</script>

<template>
  <div class="login-page">
    <van-icon name="arrow-left" size="22" class="back" @click="router.back()" />
    <div class="logo"><van-icon name="music-o" size="48" color="#fe2c55" /></div>
    <h2 class="title">{{ mode === 'login' ? '登录抖音' : '注册抖音' }}</h2>
    <div class="form">
      <van-cell-group inset style="background: #161616">
        <van-field v-model="username" placeholder="用户名" clearable />
        <van-field v-model="password" type="password" placeholder="密码" clearable />
        <van-field v-if="mode === 'register'" v-model="nickname" placeholder="昵称" clearable />
      </van-cell-group>
      <div style="margin: 20px"><van-button block round color="#fe2c55" @click="submit">{{ mode === 'login' ? '登 录' : '注 册' }}</van-button></div>
      <div class="switch" @click="mode = mode === 'login' ? 'register' : 'login'">{{ mode === 'login' ? '没有账号？去注册' : '已有账号？去登录' }}</div>
      <div class="hint">演示账号: admin / admin123</div>
    </div>
  </div>
</template>

<style scoped>
.login-page { min-height: 100vh; background: #000; padding-top: 60px; }
.back { position: absolute; top: 16px; left: 16px; color: #fff; }
.logo { text-align: center; }
.title { text-align: center; color: #fff; margin: 16px 0 30px; }
.login-page :deep(.van-cell) { background: #161616 !important; }
.login-page :deep(input) { color: #fff !important; }
.switch { text-align: center; color: #fe2c55; font-size: 14px; }
.hint { text-align: center; color: #666; font-size: 12px; margin-top: 16px; }
</style>
