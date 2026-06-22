package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/repository"
)

// ---- Recommendation ----

// RecommendFeed: GET /videos/recommend — personalized feed using CF + tags.
func (h *Handler) RecommendFeed(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	uid, _ := h.currentUserID(c, true)
	videos, err := h.Recommend.ForUser(uid, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "推荐加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": videos})
}

// RecordPlay: POST /videos/:id/play { completion: 0.8 } — implicit feedback.
func (h *Handler) RecordPlay(c *gin.Context) {
	uid, ok := h.currentUserID(c, true)
	if !ok || uid == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req struct {
		Completion float64 `json:"completion"`
	}
	_ = c.ShouldBindJSON(&req)
	_ = h.Recommend.RecordPlay(uid, id, req.Completion)
	h.Video.IncrementPlays(id) // keep play counter in sync
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- Video file upload (multipart) ----

// UploadVideoFile: POST /admin/videos/upload — accepts a multipart file,
// stores it under ./data/uploads/, and creates a video record.
func (h *Handler) UploadVideoFile(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	// Parse multipart form.
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil { // 100MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大或格式错误"})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传视频文件"})
		return
	}
	defer file.Close()

	// Persist the file locally.
	uploadDir := "data/uploads"
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "存储目录创建失败"})
		return
	}
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".mp4"
	}
	filename := fmt.Sprintf("%d_%d%s", uid, time.Now().UnixNano(), ext)
	dst := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(header, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}

	// Optional cover image.
	coverURL := c.PostForm("cover_url")
	title := c.PostForm("title")
	desc := c.PostForm("description")
	tags := c.PostForm("tags")
	music := c.PostForm("music")
	if music == "" {
		music = "原声"
	}
	// Build the public URL (served by the static route /uploads).
	videoURL := "/uploads/" + filename

	id, err := h.Video.CreateRaw(uid, title, desc, videoURL, coverURL, tags, music)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建视频记录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "video_url": videoURL, "message": "上传成功"})
}

// ---- Notifications ----

// ListNotifications: GET /notifications  (auth)
func (h *Handler) ListNotifications(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	filter := c.Query("type") // optional: like/comment/follow/system
	var list []repository.Notification
	var err error
	if filter != "" {
		list, err = h.Notify.ListByUserAndType(uid, filter, 50)
	} else {
		list, err = h.Notify.ListByUser(uid, 50)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// NotificationCounts: GET /notifications/counts  (auth)
func (h *Handler) NotificationCounts(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	counts, err := h.Notify.Counts(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"counts": counts})
}

// MarkNotificationsRead: POST /notifications/read-all  (auth)
func (h *Handler) MarkNotificationsRead(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	if err := h.Notify.MarkAllRead(uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已全部已读"})
}
