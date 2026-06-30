package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/repository"
)

// SetLive attaches the LiveRepo + the danmaku/search-log/hashtag/gift repos.
func (h *Handler) SetLive(live *repository.LiveRepo, dm *repository.DanmakuRepo, sl *repository.SearchLogRepo, ht *repository.HashtagRepo, gf *repository.GiftRepo) {
	h.Live = live
	h.Danmaku = dm
	h.SearchLog = sl
	h.Hashtag = ht
	h.Gift = gf
}

// ListLive: GET /live  — list currently-live rooms.
func (h *Handler) ListLive(c *gin.Context) {
	rooms, err := h.Live.ListLive(20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rooms})
}

// GetLive: GET /live/:id — room detail + pinned products (小黄车).
func (h *Handler) GetLive(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	room, err := h.Live.Get(id)
	if err != nil || room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "直播间不存在"})
		return
	}
	products, _ := h.Live.ListProducts(id)
	// Count the viewer in.
	h.Live.IncrementViewers(id)
	room.Viewers++
	// Include recent danmaku so the room renders with chat history.
	messages := []repository.LiveMessage{}
	if h.Danmaku != nil {
		messages, _ = h.Danmaku.ListByLive(id, 30)
	}
	c.JSON(http.StatusOK, gin.H{"room": room, "products": products, "messages": messages})
}

// LikeLive: POST /live/:id/like — bump the like counter.
func (h *Handler) LikeLive(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	h.Live.IncrementLikes(id)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SendDanmaku: POST /live/:id/messages — send a chat/danmaku message (requires auth).
func (h *Handler) SendDanmaku(c *gin.Context) {
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
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}
	u, _ := h.User.Get(uid)
	name, avatar := "", ""
	if u != nil {
		name, avatar = u.Nickname, u.Avatar
	}
	m, err := h.Danmaku.Send(id, uid, name, avatar, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发送失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m})
}

// ListDanmaku: GET /live/:id/messages — recent chat for a room.
func (h *Handler) ListDanmaku(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.Danmaku.ListByLive(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// HotSearch: GET /videos/hot-search — top searched keywords (ranked).
func (h *Handler) HotSearch(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if h.SearchLog == nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	list, err := h.SearchLog.HotSearch(limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ListGifts: GET /live/gifts — the live-room gift catalog.
func (h *Handler) ListGifts(c *gin.Context) {
	if h.Gift == nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	list, err := h.Gift.List()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
