package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/handler"
)

func New(h *handler.Handler, allowedOrigins string) *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware(allowedOrigins))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Serve uploaded video files from the local data/uploads directory.
	r.Static("/uploads", "data/uploads")
	// Serve uploaded images from the local data/images directory.
	r.Static("/images", "data/images")

	api := r.Group("/api/v1")
	{
		// Auth (public)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/register", h.Register)

		// Live streaming (public)
		api.GET("/live", h.ListLive)
		api.GET("/live/gifts", h.ListGifts)
		api.GET("/live/:id", h.GetLive)
		api.GET("/live/:id/messages", h.ListDanmaku)
		api.POST("/live/:id/like", h.LikeLive)

		// Public feed & content (identifies user if token present)
		api.GET("/videos/feed", h.Feed)
		api.GET("/videos/recommend", h.RecommendFeed)
		api.GET("/videos/search", h.SearchVideos)
		api.GET("/videos/search/suggest", h.SearchSuggest)
		api.GET("/videos/hot-search", h.HotSearch)
		api.GET("/videos/hashtags", h.HotHashtags)
		api.GET("/videos/tag/:tag", h.VideosByTag)
		api.GET("/videos/music/:id", h.VideosByMusic)
		api.GET("/videos/:id", h.GetVideo)
		api.GET("/videos/:id/comments", h.ListComments)
		api.GET("/users/:id", h.GetUser)
		api.GET("/users/:id/videos", h.UserVideos)
		api.GET("/users/:id/followers", h.Followers)
		api.GET("/users/:id/following", h.Following)

		// Authenticated actions (require token)
		auth := api.Group("/")
		auth.Use(authMiddleware())
		{
			auth.GET("/auth/profile", h.Profile)
			auth.PUT("/auth/profile", h.UpdateProfile)

			// Following feed (关注 tab)
			auth.GET("/videos/following", h.FollowingFeed)

			auth.POST("/videos/:id/like", h.ToggleLike)
			auth.POST("/videos/:id/favorite", h.ToggleFavorite)
			auth.POST("/videos/:id/play", h.RecordPlay)
			auth.GET("/users/me/favorites", h.FavoriteVideos)

			auth.POST("/comments", h.CreateComment)
			auth.POST("/comments/:id/like", h.LikeComment)
			// Live danmaku (send a chat message)
			auth.POST("/live/:id/messages", h.SendDanmaku)

			auth.POST("/users/:id/follow", h.ToggleFollow)

			// Upload / manage own videos
			auth.POST("/admin/videos", h.UploadVideo)
			auth.POST("/admin/videos/upload", h.UploadVideoFile)
			auth.DELETE("/admin/videos/:id", h.DeleteVideo)

			// Image upload (video covers, avatars)
			auth.POST("/upload", h.UploadImage)

			// Notifications
			auth.GET("/notifications", h.ListNotifications)
			auth.GET("/notifications/counts", h.NotificationCounts)
			auth.POST("/notifications/read-all", h.MarkNotificationsRead)
		}
	}
	return r
}

// authMiddleware requires a Bearer token (returns 401 if absent).
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}
		c.Next()
	}
}

func corsMiddleware(allowed string) gin.HandlerFunc {
	allowAll := strings.TrimSpace(allowed) == "*" || allowed == ""
	origins := map[string]bool{}
	for _, o := range strings.Split(allowed, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = true
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		switch {
		case allowAll:
			c.Header("Access-Control-Allow-Origin", "*")
		case origin != "" && origins[origin]:
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
