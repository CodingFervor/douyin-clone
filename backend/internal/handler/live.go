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

// SetMessages attaches the private-message repo.
func (h *Handler) SetMessages(m *repository.MessageRepo) {
	h.Messages = m
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

// ListCities: GET /live/cities — distinct cities with live room counts (城市频道).
func (h *Handler) ListCities(c *gin.Context) {
	cities, err := h.Live.ListCities()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cities})
}

// ListLiveByCity: GET /live/city/:city — live rooms in a city.
func (h *Handler) ListLiveByCity(c *gin.Context) {
	city := c.Param("city")
	rooms, err := h.Live.ListByCity(city, 20)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
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
	// Count the like as a small contribution to the contribution board.
	if uid, ok := h.currentUserID(c, true); ok && uid > 0 {
		_ = h.Live.AddContribution(id, uid, 1)
	}
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

// ===================== PK battles (直播PK) =====================

// StartPK: POST /live/:id/pk — start a PK against a random opponent (requires auth).
func (h *Handler) StartPK(c *gin.Context) {
	_, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	pk, err := h.Live.StartPK(roomID, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pk})
}

// GetActivePK: GET /live/:id/pk — the in-progress PK for a room (if any).
func (h *Handler) GetActivePK(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	pk, err := h.Live.GetActivePK(roomID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pk})
}

// ScorePK: POST /live/:id/pk/score — add points to side a or b (requires auth).
func (h *Handler) ScorePK(c *gin.Context) {
	_, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	pkID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req struct {
		Side   string `json:"side"`
		Points int    `json:"points"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Points <= 0 {
		req.Points = 10
	}
	pk, err := h.Live.ScorePK(pkID, req.Side, req.Points)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "加分失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pk})
}

// EndPK: POST /live/:id/pk/end — finalize a PK (requires auth).
func (h *Handler) EndPK(c *gin.Context) {
	_, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	pkID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	_ = h.Live.EndPK(pkID)
	c.JSON(http.StatusOK, gin.H{"message": "PK已结束"})
}

// ===================== Fan guards (粉丝勋章/守护) =====================

// GuardHost: POST /live/:id/guard — toggle 守护 a host (requires auth).
func (h *Handler) GuardHost(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	room, _ := h.Live.Get(roomID)
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "直播间不存在"})
		return
	}
	guarding, err := h.Live.Guard(uid, room.HostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"guarding": guarding, "count": h.Live.GuardCount(room.HostID)})
}

// GuardStatus: GET /live/:id/guard — guard count + whether the current user guards.
func (h *Handler) GuardStatus(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	room, _ := h.Live.Get(roomID)
	if room == nil {
		c.JSON(http.StatusOK, gin.H{"count": 0, "guarding": false})
		return
	}
	uid, _ := h.currentUserID(c, true)
	c.JSON(http.StatusOK, gin.H{"count": h.Live.GuardCount(room.HostID), "guarding": h.Live.IsGuarding(uid, room.HostID), "guards": mustList(h.Live, room.HostID)})
}

// mustList is a small helper that swallows the guards-list error.
func mustList(r *repository.LiveRepo, hostID int64) []repository.FanGuard {
	list, _ := r.ListGuards(hostID, 10)
	return list
}

// ===================== Red packets (红包雨) =====================

// DropPacket: POST /live/:id/redpacket — host drops a red packet (requires auth).
func (h *Handler) DropPacket(c *gin.Context) {
	_, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req struct {
		Total     int `json:"total"`
		AmountPer int `json:"amount_per"`
	}
	_ = c.ShouldBindJSON(&req)
	p, err := h.Live.DropPacket(roomID, req.Total, req.AmountPer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发送失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

// ActivePacket: GET /live/:id/redpacket — the active red packet for a room (if any).
func (h *Handler) ActivePacket(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	p, err := h.Live.ActivePacket(roomID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

// GrabPacket: POST /live/:id/redpacket/grab — grab a share (requires auth).
func (h *Handler) GrabPacket(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	p, _ := h.Live.ActivePacket(roomID)
	if p == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "暂无红包"})
		return
	}
	amount, err := h.Live.GrabPacket(p.ID, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"amount": amount, "message": "抢到" + strconv.Itoa(amount) + "金币"})
}

// ===================== Contribution board (贡献榜) =====================

// ListContributors: GET /live/:id/contributors — top fans by gift/like value.
func (h *Handler) ListContributors(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	list, err := h.Live.ListContributors(roomID, 20)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// Contribute: POST /live/:id/contribute — record a gift/interaction value (requires auth).
func (h *Handler) Contribute(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req struct {
		Amount int `json:"amount"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Amount <= 0 {
		req.Amount = 10
	}
	if err := h.Live.AddContribution(roomID, uid, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "记录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
