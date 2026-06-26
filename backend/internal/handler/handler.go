package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
	"github.com/CodingFervor/douyin-clone/backend/internal/repository"
)

type Handler struct {
	User      *repository.UserRepo
	Video     *repository.VideoRepo
	Like      *repository.LikeRepo
	Comment   *repository.CommentRepo
	Follow    *repository.FollowRepo
	Favorite  *repository.FavoriteRepo
	Recommend *repository.RecommendRepo
	Notify    *repository.NotifyRepo
	Live      *repository.LiveRepo
	Danmaku   *repository.DanmakuRepo
	SearchLog *repository.SearchLogRepo
	jwtKey    []byte
}

func New(jwtSecret string, u *repository.UserRepo, v *repository.VideoRepo, l *repository.LikeRepo,
	c *repository.CommentRepo, f *repository.FollowRepo, fv *repository.FavoriteRepo,
	rec *repository.RecommendRepo, n *repository.NotifyRepo) *Handler {
	return &Handler{User: u, Video: v, Like: l, Comment: c, Follow: f, Favorite: fv, Recommend: rec, Notify: n, jwtKey: []byte(jwtSecret)}
}

// ---- JWT ----

func (h *Handler) signToken(userID int64, username string) string {
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := `{"user_id":` + strconv.FormatInt(userID, 10) + `,"exp":` + strconv.FormatInt(time.Now().Add(168*time.Hour).Unix(), 10) + `}`
	return b64(header) + "." + b64(payload) + "." + h.sig(header, payload)
}

func (h *Handler) parseToken(token string) (int64, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || h.sig(b64d(parts[0]), b64d(parts[1])) != parts[2] {
		return 0, false
	}
	uid := extractInt(b64d(parts[1]), "user_id")
	exp := extractInt(b64d(parts[1]), "exp")
	if exp > 0 && time.Now().Unix() > exp {
		return 0, false
	}
	return uid, true
}

func (h *Handler) sig(header, payload string) string {
	sum := sha256.Sum256([]byte(header + "." + payload + "." + string(h.jwtKey)))
	return hex.EncodeToString(sum[:])
}

// currentUserID returns the user id from the bearer token; optional=true allows
// unauthenticated browsing (the feed is public) while still identifying the user.
func (h *Handler) currentUserID(c *gin.Context, optional bool) (int64, bool) {
	auth := c.GetHeader("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		if optional {
			return 0, true
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return 0, false
	}
	uid, ok := h.parseToken(strings.TrimPrefix(auth, "Bearer "))
	if !ok {
		if optional {
			return 0, true
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "登录已过期"})
		return 0, false
	}
	return uid, true
}

func hashPassword(plain string) string {
	sum := sha256.Sum256([]byte(plain + "dy-salt"))
	return hex.EncodeToString(sum[:])
}

// ---- Auth ----

func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码必填"})
		return
	}
	u, err := h.User.FindByUsername(req.Username)
	if err != nil || u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if u.Password != req.Password && u.Password != hashPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": h.signToken(u.ID, u.Username), "user": u})
}

func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合法"})
		return
	}
	if h.User.Exists(req.Username) {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	u := &model.User{Username: req.Username, Password: hashPassword(req.Password), Nickname: req.Nickname, Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=" + req.Username}
	if u.Nickname == "" {
		u.Nickname = req.Username
	}
	if err := h.User.Create(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "注册失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": h.signToken(u.ID, u.Username), "user": u})
}

func (h *Handler) Profile(c *gin.Context) {
	uid, ok := h.currentUserID(c, false)
	if !ok {
		return
	}
	u, err := h.User.Get(uid)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": u})
}

func (h *Handler) GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	u, err := h.User.Get(id)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	curID, _ := h.currentUserID(c, true)
	u.IsFollowing = h.Follow.IsFollowing(curID, id)
	c.JSON(http.StatusOK, gin.H{"user": u})
}
