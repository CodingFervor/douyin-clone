package model

import "time"

type User struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	Password       string    `json:"-"`
	Nickname       string    `json:"nickname"`
	Avatar         string    `json:"avatar"`
	Bio            string    `json:"bio"`
	FollowingCount int       `json:"following_count"`
	FollowersCount int       `json:"followers_count"`
	LikesCount     int       `json:"likes_count"`
	IsFollowing    bool      `json:"is_following"`
	CreatedAt      time.Time `json:"created_at"`
}

type Video struct {
	ID            int64     `json:"id"`
	AuthorID      int64     `json:"author_id"`
	AuthorName    string    `json:"author_name"`
	AuthorAvatar  string    `json:"author_avatar"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	VideoURL      string    `json:"video_url"`
	CoverURL      string    `json:"cover_url"`
	Duration      int       `json:"duration"`
	Plays         int       `json:"plays"`
	Likes         int       `json:"likes"`
	CommentsCount int       `json:"comments_count"`
	Shares        int       `json:"shares"`
	Tags          string    `json:"tags"`
	Music         string    `json:"music"`
	Liked         bool      `json:"liked"`
	Favorited     bool      `json:"favorited"`
	CreatedAt     time.Time `json:"created_at"`
}

type Comment struct {
	ID        int64     `json:"id"`
	VideoID   int64     `json:"video_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Avatar    string    `json:"avatar"`
	Content   string    `json:"content"`
	Likes     int       `json:"likes"`
	Liked     bool      `json:"liked"`
	ParentID  int64     `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ---- Request DTOs ----

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
}

type CreateCommentRequest struct {
	VideoID  int64  `json:"video_id" binding:"required"`
	Content  string `json:"content" binding:"required"`
	ParentID int64  `json:"parent_id"`
}

type VideoInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	VideoURL    string `json:"video_url" binding:"required"`
	CoverURL    string `json:"cover_url"`
	Duration    int    `json:"duration"`
	Tags        string `json:"tags"`
	Music       string `json:"music"`
}

// Hashtag is an aggregated topic tag (#话题) ranked by usage.
type Hashtag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Uses int    `json:"uses"`
}

// LiveGift is a gift in the live-room gift tray.
type LiveGift struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Price     int    `json:"price"`
	SortOrder int    `json:"sort_order"`
}

type ProfileInput struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
}
