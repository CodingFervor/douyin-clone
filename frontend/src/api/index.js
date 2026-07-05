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

// ---- Image upload (multipart) ----
export const uploadImage = (file) => {
  const fd = new FormData()
  fd.append('file', file)
  return http.post('/upload', fd, { headers: { 'Content-Type': 'multipart/form-data' }, timeout: 30000 }).then((r) => r.data)
}

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

// ---- Live streaming ----
export const getLiveList = () => http.get('/live').then((r) => r.data.data)
export const getLiveRoom = (id) => http.get(`/live/${id}`).then((r) => r.data)
export const likeLive = (id) => http.post(`/live/${id}/like`).then((r) => r.data)

// ---- Search (FTS5) ----
export const searchVideos = (q) => http.get('/videos/search', { params: { q } }).then((r) => r.data.data)
export const searchSuggest = (q) => http.get('/videos/search/suggest', { params: { q } }).then((r) => r.data.data)
export const getHotSearch = () => http.get('/videos/hot-search').then((r) => r.data.data)

// ---- Live danmaku ----
export const getLiveMessages = (id, limit) => http.get(`/live/${id}/messages`, { params: { limit } }).then((r) => r.data.data)
export const sendLiveMessage = (id, content) => http.post(`/live/${id}/messages`, { content }).then((r) => r.data.data)

// ---- Live gifts ----
export const getLiveGifts = () => http.get('/live/gifts').then((r) => r.data.data)

// ---- Live PK ----
export const startPK = (roomId) => http.post(`/live/${roomId}/pk`).then((r) => r.data.data)
export const getActivePK = (roomId) => http.get(`/live/${roomId}/pk`).then((r) => r.data.data)
export const scorePK = (pkId, side, points) => http.post(`/live/${pkId}/pk/score`, { side, points }).then((r) => r.data.data)
export const endPK = (pkId) => http.post(`/live/${pkId}/pk/end`).then((r) => r.data)

// ---- Fan guard (守护) ----
export const guardHost = (roomId) => http.post(`/live/${roomId}/guard`).then((r) => r.data)
export const getGuardStatus = (roomId) => http.get(`/live/${roomId}/guard`).then((r) => r.data)

// ---- Red packets (红包雨) ----
export const dropRedPacket = (roomId, total, amountPer) => http.post(`/live/${roomId}/redpacket`, { total, amount_per: amountPer }).then((r) => r.data.data)
export const getActiveRedPacket = (roomId) => http.get(`/live/${roomId}/redpacket`).then((r) => r.data.data)
export const grabRedPacket = (roomId) => http.post(`/live/${roomId}/redpacket/grab`).then((r) => r.data)

// ---- City channel (城市频道) ----
export const getLiveCities = () => http.get('/live/cities').then((r) => r.data.data)
export const getLiveByCity = (city) => http.get(`/live/city/${encodeURIComponent(city)}`).then((r) => r.data.data)

// ---- Contribution board (贡献榜) ----
export const getContributors = (roomId) => http.get(`/live/${roomId}/contributors`).then((r) => r.data.data)
export const contribute = (roomId, amount) => http.post(`/live/${roomId}/contribute`, { amount }).then((r) => r.data)

// ---- Private messages (私信) ----
export const getConversations = () => http.get('/dm').then((r) => r.data.data)
export const getConversation = (userId) => http.get(`/dm/${userId}`).then((r) => r.data.data)
export const sendDM = (userId, content) => http.post(`/dm/${userId}`, { content }).then((r) => r.data.data)

// ---- Creator analytics (视频数据统计) ----
export const getCreatorStats = () => http.get('/creator/stats').then((r) => r.data.data)

// ---- Live schedules (直播预告) ----
export const getLiveSchedules = () => http.get('/live/schedules').then((r) => r.data.data)

// ---- Hot music (热门音乐榜) ----
export const getHotMusic = () => http.get('/videos/hot-music').then((r) => r.data.data)

// ---- Video report (视频举报) ----
export const reportVideo = (id, reason) => http.post(`/videos/${id}/report`, { reason }).then((r) => r.data)

// ---- Live ban (直播间禁言) ----
export const banUser = (roomId, userId) => http.post(`/live/${roomId}/ban`, { user_id: userId }).then((r) => r.data)

// ---- Hashtags (#话题) ----
export const getHotHashtags = () => http.get('/videos/hashtags').then((r) => r.data.data)
export const getVideosByTag = (tag) => http.get(`/videos/tag/${encodeURIComponent(tag)}`).then((r) => r.data.data)

// ---- Same BGM (同款BGM) ----
export const getVideosByMusic = (id) => http.get(`/videos/music/${id}`).then((r) => r.data)

// ---- Following feed (关注 tab) ----
export const getFollowingFeed = (limit = 20) => http.get('/videos/following', { params: { limit } }).then((r) => r.data.data)

// ---- Duets (合拍) ----
export const getDuets = (id) => http.get(`/videos/${id}/duets`).then((r) => r.data.data)
export const createDuet = (id, payload) => http.post(`/videos/${id}/duet`, payload).then((r) => r.data)

// ---- Nearby users (LBS / 附近的人) ----
export const getNearbyUsers = () => http.get('/users/nearby').then((r) => r.data.data)
export const updateLocation = (lat, lng, city) => http.put('/auth/location', { latitude: lat, longitude: lng, city }).then((r) => r.data)

// ---- Comment likes ----
export const likeComment = (id) => http.post(`/comments/${id}/like`).then((r) => r.data)

// ---- Profile ----
export const updateProfile = (payload) => http.put('/auth/profile', payload).then((r) => r.data.user)
