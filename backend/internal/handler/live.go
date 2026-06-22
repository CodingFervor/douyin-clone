package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/repository"
)

// SetLive attaches the LiveRepo.
func (h *Handler) SetLive(live *repository.LiveRepo) {
	h.Live = live
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
	c.JSON(http.StatusOK, gin.H{"room": room, "products": products})
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
