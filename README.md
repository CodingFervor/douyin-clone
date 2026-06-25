# Douyin Clone | 抖音短视频仿制

## English | [中文](#中文)

A Douyin (抖音/TikTok) clone — a full-screen short-video app with swipe-to-switch feed, like/comment/follow/favorite, user profiles, and content upload. Go + Gin backend, Vue 3 + Vant frontend, SQLite storage. Auto-seeds users, videos, and comments on first run.

### Features
- **Immersive full-screen video feed** — vertical swipe (touch up/down) to switch videos, tap to pause/play
- **Recommendation engine** — collaborative filtering (users who share your likes) + tag affinity + cold-start popularity fallback; play-completion ratio as implicit feedback
- **Right action rail** — like (toggle with animation), comment, favorite (收藏), share, author avatar + follow
- **Comments** — bottom-sheet comment panel, post comments in real time
- **Notification system** — like/comment/follow notifications with per-type tabs + unread badges + mark-all-read
- **Live streaming (直播)** — live room grid with viewer counts, full-screen HLS player, host info + follow, like with floating-heart animation
- **Shopping cart (小黄车)** — pinned products per live room, product teaser bar, cart popup with price/sales/抢购
- **Follow system** — follow/unfollow authors, followers & following lists
- **User profiles** — avatar, bio, following/followers/likes stats, works & liked tabs
- **Discover** — search videos by title/tags/author, trending topic tags
- **Video upload** — real file upload (multipart) with progress bar + cover image upload, or URL paste
- **Image upload** — real multipart upload for video covers
- **Auth** — register/login with JWT
- **Black-theme UI** — authentic Douyin dark aesthetic with cyan/pink accents

### Tech Stack
- **Backend**: Go 1.22 + Gin + SQLite (`modernc.org/sqlite`, pure-Go, CGO-free)
- **Frontend**: Vue 3 + Vite + Vant 4 + Vue Router + Axios
- **Deploy**: Docker Compose (backend + nginx frontend) + SQLite volume

### Project Structure
```
douyin-clone/
├── backend/
│   ├── cmd/server/main.go        # entry: init DB → seed → routes → shutdown
│   └── internal/
│       ├── config/db/seed/model/
│       ├── repository/           # videos/likes/comments/follows/favorites
│       ├── handler/              # auth + feed + interactions
│       └── server/               # gin routes + CORS
├── frontend/
│   └── src/views/                # Feed/Discover/Upload/Messages/Mine/Login/UserProfile
├── docker-compose.yml
└── README.md
```

### Quick Start

#### Docker Compose
```bash
docker-compose up -d --build
# Frontend: http://localhost  ·  API: http://localhost:8080
```

#### Run separately (dev)
```bash
cd backend && go run ./cmd/server      # :8080, auto-seeds
cd frontend && npm install && npm run dev   # :5173
```

### Demo Account
`admin` / `admin123`

### API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/auth/login · /register | Auth |
| GET | /api/v1/auth/profile | My profile (auth) |
| GET | /api/v1/videos/feed | Video feed (public; identifies user) |
| GET | /api/v1/videos/:id | Single video |
| GET | /api/v1/videos/:id/comments | Comments |
| POST | /api/v1/videos/:id/like | Toggle like (auth) |
| POST | /api/v1/videos/:id/favorite | Toggle favorite (auth) |
| POST | /api/v1/comments | Add comment (auth) |
| GET | /api/v1/users/:id | User profile |
| GET | /api/v1/users/:id/videos | User's videos |
| GET | /api/v1/users/:id/followers · /following | Follow lists |
| POST | /api/v1/users/:id/follow | Toggle follow (auth) |
| GET | /api/v1/users/me/favorites | My favorites (auth) |
| POST/DELETE | /api/v1/admin/videos | Upload/delete (auth) |

### Video Content
Mock videos use Google's public sample video library (commondatastorage.googleapis.com/gtv-videos-bucket). Replace with your own video URLs in production.

### License
MIT — see [LICENSE](LICENSE).

---

<a id="中文"></a>
# 抖音短视频仿制

抖音（Douyin/TikTok）仿制 —— 全屏沉浸式短视频 App，含上下滑切换、点赞/评论/关注/收藏、个人主页、内容发布。Go + Gin 后端 + Vue 3 + Vant 前端 + SQLite，首次启动自动填充用户/视频/评论。

### 功能特性
- **全屏沉浸式视频流** — 上下滑动切换视频，点击暂停/播放
- **右侧操作栏** — 点赞（带动画切换）、评论、收藏、分享、作者头像 + 关注
- **评论** — 底部评论面板，实时发送评论
- **关注系统** — 关注/取关作者，粉丝 & 关注列表
- **个人主页** — 头像、简介、关注/粉丝/获赞数、作品 & 喜欢页签
- **发现** — 按标题/标签/作者搜索视频，热门话题
- **直播** — 直播间网格（观看人数脉冲徽章）、全屏 HLS 播放器、主播信息 + 关注、点赞飘心动画
- **小黄车** — 直播间挂载商品、底部商品预览条、购物车弹层（商品/价格/已售/抢购）
- **消息通知** — 点赞/评论/关注通知，分类 tab + 未读角标 + 全部已读
- **发布** — 真实视频文件上传（multipart + 进度条）+ 封面图上传，或 URL 填写
- **图片上传** — 视频封面真实 multipart 上传
- **登录注册** — JWT 鉴权
- **黑色主题** — 还原抖音暗色风格，青/粉撞色

### 技术栈
- **后端**：Go 1.22 + Gin + SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO-free）
- **前端**：Vue 3 + Vite + Vant 4 + Vue Router + Axios
- **部署**：Docker Compose（后端 + nginx 前端）+ SQLite 数据卷

### 快速开始
```bash
# Docker 一键
docker-compose up -d --build
# 分别运行
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

### 演示账号
`admin` / `admin123`

API 端点详见上方英文版表格。

### 视频内容
Mock 视频使用 Google 公共示例视频库，生产环境请替换为自己的视频地址。

### 开源许可
MIT — 详见 [LICENSE](LICENSE)。
