package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ===================== Private messages (私信) =====================

// SendDM: POST /dm/:userId — send a direct message (requires auth).
func (h *Handler) SendDM(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	receiverID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}
	m, err := h.Messages.Send(uid, receiverID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发送失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m})
}

// GetConversation: GET /dm/:userId — message thread with a user (requires auth).
func (h *Handler) GetConversation(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	otherID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}
	list, err := h.Messages.Conversation(uid, otherID, 50)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	// Mark messages from the other user as read.
	_ = h.Messages.MarkRead(uid, otherID)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ConversationList: GET /dm — list of conversations (requires auth).
func (h *Handler) ConversationList(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	list, err := h.Messages.ConversationList(uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ===================== Video analytics (视频数据统计) =====================

// CreatorStats: GET /creator/stats — aggregate stats for the current user's videos.
func (h *Handler) CreatorStats(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	var stats struct {
		VideoCount    int `json:"video_count"`
		TotalPlays    int `json:"total_plays"`
		TotalLikes    int `json:"total_likes"`
		TotalComments int `json:"total_comments"`
		TotalShares   int `json:"total_shares"`
	}
	_ = h.Video.GetDB().QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(plays),0), COALESCE(SUM(likes),0), COALESCE(SUM(comments_count),0), COALESCE(SUM(shares),0)
		 FROM videos WHERE author_id=?`, uid,
	).Scan(&stats.VideoCount, &stats.TotalPlays, &stats.TotalLikes, &stats.TotalComments, &stats.TotalShares)
	c.JSON(http.StatusOK, gin.H{"data": stats})
}
