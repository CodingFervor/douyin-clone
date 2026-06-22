import axios from 'axios'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api/v1',
  timeout: 15000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('dy_token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  return config
})

http.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response && err.response.status === 401) {
      localStorage.removeItem('dy_token')
    }
    return Promise.reject(err)
  }
)

export const login = (username, password) => http.post('/auth/login', { username, password }).then((r) => r.data)
export const register = (payload) => http.post('/auth/register', payload).then((r) => r.data)
export const getProfile = () => http.get('/auth/profile').then((r) => r.data.user)
export const getUser = (id) => http.get(`/users/${id}`).then((r) => r.data.user)
export const getFeed = (limit = 20) => http.get('/videos/feed', { params: { limit } }).then((r) => r.data.data)
export const getVideo = (id) => http.get(`/videos/${id}`).then((r) => r.data.data)
export const getUserVideos = (id) => http.get(`/users/${id}/videos`).then((r) => r.data.data)
export const toggleLike = (id) => http.post(`/videos/${id}/like`).then((r) => r.data)
export const toggleFavorite = (id) => http.post(`/videos/${id}/favorite`).then((r) => r.data)
export const getFavoriteVideos = () => http.get('/users/me/favorites').then((r) => r.data.data)
export const getComments = (id) => http.get(`/videos/${id}/comments`).then((r) => r.data.data)
export const createComment = (payload) => http.post('/comments', payload).then((r) => r.data.data)
export const toggleFollow = (id) => http.post(`/users/${id}/follow`).then((r) => r.data)
export const getFollowers = (id) => http.get(`/users/${id}/followers`).then((r) => r.data.data)
export const getFollowing = (id) => http.get(`/users/${id}/following`).then((r) => r.data.data)
export const uploadVideo = (payload) => http.post('/admin/videos', payload).then((r) => r.data)

// ---- Recommendations ----
export const getRecommendFeed = (limit = 20) => http.get('/videos/recommend', { params: { limit } }).then((r) => r.data.data)
export const recordPlay = (id, completion) => http.post(`/videos/${id}/play`, { completion }).then((r) => r.data)

// ---- File upload (multipart) ----
export const uploadVideoFile = (formData, onProgress) =>
  http.post('/admin/videos/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: onProgress,
    timeout: 120000,
  }).then((r) => r.data)

// ---- Notifications ----
export const getNotifications = (type) => http.get('/notifications', { params: type ? { type } : {} }).then((r) => r.data.data)
export const getNotificationCounts = () => http.get('/notifications/counts').then((r) => r.data.counts)
export const markNotificationsRead = () => http.post('/notifications/read-all').then((r) => r.data)
