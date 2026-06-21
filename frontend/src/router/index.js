import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/feed' },
  { path: '/feed', name: 'feed', component: () => import('../views/Feed.vue'), meta: { tab: 0 } },
  { path: '/discover', name: 'discover', component: () => import('../views/Discover.vue'), meta: { tab: 1 } },
  { path: '/upload', name: 'upload', component: () => import('../views/Upload.vue'), meta: { tab: 2, auth: true } },
  { path: '/messages', name: 'messages', component: () => import('../views/Messages.vue'), meta: { tab: 3 } },
  { path: '/mine', name: 'mine', component: () => import('../views/Mine.vue'), meta: { tab: 4 } },
  { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
  { path: '/user/:id', name: 'user', component: () => import('../views/UserProfile.vue') },
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to, from, next) => {
  if (to.meta.auth && !localStorage.getItem('dy_token')) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else {
    next()
  }
})

export default router
