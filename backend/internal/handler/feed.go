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

// FollowingFeed: GET /videos/following — videos by followed authors (关注 tab).
func (h *Handler) FollowingFeed(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	videos, err := h.Video.ListFollowingFeed(uid, limit)
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

// ListDuets: GET /videos/:id/duets — videos that duetted the original (合拍).
func (h *Handler) ListDuets(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	uid, _ := h.currentUserID(c, true)
	list, err := h.Video.ListDuets(id, 20, uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
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
	// Notify the video author when a like is added.
	if liked && h.Notify != nil {
		v, _ := h.Video.Get(id, 0)
		actor, _ := h.User.Get(uid)
		if v != nil && actor != nil {
			_ = h.Notify.Notify(v.AuthorID, uid, actor.Nickname, actor.Avatar, "like", id, "赞了你的视频")
		}
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
	uid, _ := h.currentUserID(c, true)
	comments, err := h.Comment.ListByVideoAnnotated(id, 100, uid)
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
	// Notify the video author about the new comment.
	if h.Notify != nil {
		v, _ := h.Video.Get(req.VideoID, 0)
		actor, _ := h.User.Get(uid)
		if v != nil && actor != nil {
			_ = h.Notify.Notify(v.AuthorID, uid, actor.Nickname, actor.Avatar, "comment", req.VideoID, "评论了你的视频: "+truncateStr(req.Content, 40))
		}
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
	// Notify the target user when followed.
	if following && h.Notify != nil {
		actor, _ := h.User.Get(uid)
		if actor != nil {
			_ = h.Notify.Notify(targetID, uid, actor.Nickname, actor.Avatar, "follow", 0, "关注了你")
		}
	}
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

// CreateDuet: POST /videos/:id/duet — create a duet (合拍) of an existing video.
// For the demo the duet reuses the parent's media URL (no real compositing).
func (h *Handler) CreateDuet(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	parentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	parent, err := h.Video.Get(parentID, 0)
	if err != nil || parent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "原视频不存在"})
		return
	}
	var req struct {
		Title string `json:"title"`
		Tags  string `json:"tags"`
	}
	_ = c.ShouldBindJSON(&req)
	title := req.Title
	if title == "" {
		title = "合拍 @" + parent.AuthorName
	}
	vid, err := h.Video.CreateDuet(uid, parentID, title, "合拍作品", parent.VideoURL, parent.CoverURL, req.Tags, parent.Music)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "合拍失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": vid, "message": "合拍成功"})
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

// ReportVideo: POST /videos/:id/report — report a video (requires auth).
func (h *Handler) ReportVideo(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.Video.Report(id, uid, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "举报失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "举报已提交"})
}

// truncateStr caps s to n runes for short notification previews.
func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
