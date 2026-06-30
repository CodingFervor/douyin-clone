import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/feed' },
  { path: '/feed', name: 'feed', component: () => import('../views/Feed.vue'), meta: { tab: 0 } },
  { path: '/live', name: 'live', component: () => import('../views/Live.vue'), meta: { tab: 1 } },
  { path: '/live/:id', name: 'liveRoom', component: () => import('../views/LiveRoom.vue') },
  { path: '/discover', name: 'discover', component: () => import('../views/Discover.vue'), meta: { tab: 2 } },
  { path: '/upload', name: 'upload', component: () => import('../views/Upload.vue'), meta: { tab: 3, auth: true } },
  { path: '/messages', name: 'messages', component: () => import('../views/Messages.vue'), meta: { tab: 4 } },
  { path: '/mine', name: 'mine', component: () => import('../views/Mine.vue'), meta: { tab: 5 } },
  { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
  { path: '/user/:id', name: 'user', component: () => import('../views/UserProfile.vue') },
  { path: '/profile', name: 'profile', component: () => import('../views/EditProfile.vue'), meta: { auth: true } },
  { path: '/tag/:tag', name: 'hashtag', component: () => import('../views/HashtagPage.vue') },
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
