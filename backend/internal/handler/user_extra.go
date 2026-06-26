package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
)

// ===================== Video search (FTS5) =====================

// SearchVideos: GET /videos/search?q=...
func (h *Handler) SearchVideos(c *gin.Context) {
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	uid, _ := h.currentUserID(c, true)
	// Log the search for the hot-search ranking (best-effort).
	if h.SearchLog != nil && q != "" {
		h.SearchLog.Log(q, uid)
	}
	results, err := h.Video.Search(q, limit, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": results, "total": len(results)})
}

// SearchSuggest: GET /videos/search/suggest?q=...
func (h *Handler) SearchSuggest(c *gin.Context) {
	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
		return
	}
	out, err := h.Video.Suggest(prefix, 10)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// ===================== Comment likes =====================

// LikeComment: POST /comments/:id/like  (requires auth)
func (h *Handler) LikeComment(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID"})
		return
	}
	liked, err := h.Comment.ToggleLike(uid, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

// ===================== Edit profile =====================

// UpdateProfile: PUT /auth/profile  (requires auth)
func (h *Handler) UpdateProfile(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	var req model.ProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	u, err := h.User.Get(uid)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if req.Nickname != "" {
		u.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		u.Avatar = req.Avatar
	}
	if req.Bio != "" {
		u.Bio = req.Bio
	}
	if err := h.User.UpdateProfile(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": u})
}
