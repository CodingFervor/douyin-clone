package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
)

// ---- Feed ----

// Feed: GET /videos/feed?limit=20  (public; identifies user if logged in)
func (h *Handler) Feed(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	uid, _ := h.currentUserID(c, true)
	videos, err := h.Video.Feed(limit, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": videos})
}

// GetVideo: GET /videos/:id
func (h *Handler) GetVideo(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	uid, _ := h.currentUserID(c, true)
	v, err := h.Video.Get(id, uid)
	if err != nil || v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}
	// count the play
	go h.Video.IncrementPlays(id)
	c.JSON(http.StatusOK, gin.H{"data": v})
}

// UserVideos: GET /users/:id/videos
func (h *Handler) UserVideos(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	uid, _ := h.currentUserID(c, true)
	videos, err := h.Video.ListByAuthor(id, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": videos})
}

// ---- Likes ----

// ToggleLike: POST /videos/:id/like  (requires auth)
func (h *Handler) ToggleLike(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	total, liked, err := h.Like.Toggle(uid, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"likes": total, "liked": liked})
}

// ---- Favorites ----

// ToggleFavorite: POST /videos/:id/favorite  (requires auth)
func (h *Handler) ToggleFavorite(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	faved, err := h.Favorite.Toggle(uid, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorited": faved})
}

// FavoriteVideos: GET /users/me/favorites
func (h *Handler) FavoriteVideos(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	videos, err := h.Video.ListFavorites(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": videos})
}

// ---- Comments ----

// ListComments: GET /videos/:id/comments
func (h *Handler) ListComments(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	comments, err := h.Comment.ListByVideo(id, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": comments})
}

// CreateComment: POST /comments  (requires auth)
func (h *Handler) CreateComment(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	var req model.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	u, _ := h.User.Get(uid)
	cm := &model.Comment{VideoID: req.VideoID, UserID: uid, Content: req.Content, ParentID: req.ParentID}
	if u != nil {
		cm.Username = u.Nickname
		cm.Avatar = u.Avatar
	}
	if err := h.Comment.Create(cm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cm})
}

// ---- Follows ----

// ToggleFollow: POST /users/:id/follow  (requires auth)
func (h *Handler) ToggleFollow(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	following, err := h.Follow.Toggle(uid, targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delta := 1
	if !following {
		delta = -1
	}
	_ = h.User.UpdateFollowingCount(uid, delta)
	_ = h.User.UpdateFollowersCount(targetID, delta)
	c.JSON(http.StatusOK, gin.H{"following": following})
}

// Followers: GET /users/:id/followers
func (h *Handler) Followers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	users, err := h.Follow.ListFollowers(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// Following: GET /users/:id/following
func (h *Handler) Following(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	users, err := h.Follow.ListFollowing(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// ---- Admin video upload ----

// UploadVideo: POST /admin/videos  (requires auth; any logged-in user can post)
func (h *Handler) UploadVideo(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	var req model.VideoInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	id, err := h.Video.Create(&req, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发布失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "发布成功"})
}

// DeleteVideo: DELETE /admin/videos/:id
func (h *Handler) DeleteVideo(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.Video.Delete(id, uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
