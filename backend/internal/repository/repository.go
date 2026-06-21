package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
)

// ===================== User =====================

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(u *model.User) error {
	res, err := r.db.Exec(
		`INSERT INTO users (username, password, nickname, avatar, bio) VALUES (?,?,?,?,?)`,
		u.Username, u.Password, defaultStr(u.Nickname, u.Username), u.Avatar, u.Bio)
	if err != nil {
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password, nickname, avatar, bio, following_count, followers_count, likes_count, created_at FROM users WHERE username=?`, username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Avatar, &u.Bio, &u.FollowingCount, &u.FollowersCount, &u.LikesCount, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) Get(id int64) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password, nickname, avatar, bio, following_count, followers_count, likes_count, created_at FROM users WHERE id=?`, id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Avatar, &u.Bio, &u.FollowingCount, &u.FollowersCount, &u.LikesCount, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *UserRepo) Exists(username string) bool {
	var n int
	_ = r.db.QueryRow(`SELECT 1 FROM users WHERE username=? LIMIT 1`, username).Scan(&n)
	return n == 1
}

func (r *UserRepo) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *UserRepo) UpdateFollowingCount(id int64, delta int) error {
	_, err := r.db.Exec(`UPDATE users SET following_count = MAX(0, following_count + ?) WHERE id=?`, delta, id)
	return err
}
func (r *UserRepo) UpdateFollowersCount(id int64, delta int) error {
	_, err := r.db.Exec(`UPDATE users SET followers_count = MAX(0, followers_count + ?) WHERE id=?`, delta, id)
	return err
}

// ===================== Video =====================

type VideoRepo struct{ db *sql.DB }

func NewVideoRepo(db *sql.DB) *VideoRepo { return &VideoRepo{db: db} }

// Feed returns a feed of videos ordered newest-first, joined with author info,
// and annotated with whether the current user liked/favorited each.
func (r *VideoRepo) Feed(limit int, currentUserID int64) ([]model.Video, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT v.id, v.author_id, u.nickname, u.avatar, v.title, v.description,
		        v.video_url, v.cover_url, v.duration, v.plays, v.likes, v.comments_count,
		        v.shares, v.tags, v.music, v.created_at
		 FROM videos v JOIN users u ON u.id = v.author_id
		 ORDER BY v.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Video{}
	for rows.Next() {
		var v model.Video
		if err := scanVideo(rows, &v); err == nil {
			out = append(out, v)
		}
	}
	r.annotateUserState(out, currentUserID)
	return out, nil
}

// ListByAuthor returns all videos by a user.
func (r *VideoRepo) ListByAuthor(authorID, currentUserID int64) ([]model.Video, error) {
	rows, err := r.db.Query(
		`SELECT v.id, v.author_id, u.nickname, u.avatar, v.title, v.description,
		        v.video_url, v.cover_url, v.duration, v.plays, v.likes, v.comments_count,
		        v.shares, v.tags, v.music, v.created_at
		 FROM videos v JOIN users u ON u.id = v.author_id
		 WHERE v.author_id=? ORDER BY v.id DESC`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Video{}
	for rows.Next() {
		var v model.Video
		if err := scanVideo(rows, &v); err == nil {
			out = append(out, v)
		}
	}
	r.annotateUserState(out, currentUserID)
	return out, nil
}

// ListFavorite returns videos a user has favorited.
func (r *VideoRepo) ListFavorites(userID int64) ([]model.Video, error) {
	rows, err := r.db.Query(
		`SELECT v.id, v.author_id, u.nickname, u.avatar, v.title, v.description,
		        v.video_url, v.cover_url, v.duration, v.plays, v.likes, v.comments_count,
		        v.shares, v.tags, v.music, v.created_at
		 FROM favorites f
		 JOIN videos v ON v.id = f.video_id
		 JOIN users u ON u.id = v.author_id
		 WHERE f.user_id=? ORDER BY f.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Video{}
	for rows.Next() {
		var v model.Video
		v.Favorited = true
		if err := scanVideo(rows, &v); err == nil {
			out = append(out, v)
		}
	}
	return out, nil
}

func (r *VideoRepo) Get(id, currentUserID int64) (*model.Video, error) {
	v := &model.Video{}
	row := r.db.QueryRow(
		`SELECT v.id, v.author_id, u.nickname, u.avatar, v.title, v.description,
		        v.video_url, v.cover_url, v.duration, v.plays, v.likes, v.comments_count,
		        v.shares, v.tags, v.music, v.created_at
		 FROM videos v JOIN users u ON u.id = v.author_id WHERE v.id=?`, id)
	if err := scanVideoRow(row, v); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.annotateUserState([]model.Video{*v}, currentUserID)
	v.Liked = false
	v.Favorited = false
	if currentUserID > 0 {
		var liked, fav int
		_ = r.db.QueryRow(`SELECT 1 FROM likes WHERE user_id=? AND video_id=?`, currentUserID, id).Scan(&liked)
		_ = r.db.QueryRow(`SELECT 1 FROM favorites WHERE user_id=? AND video_id=?`, currentUserID, id).Scan(&fav)
		v.Liked = liked == 1
		v.Favorited = fav == 1
	}
	return v, nil
}

func (r *VideoRepo) Create(v *model.VideoInput, authorID int64) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO videos (author_id, title, description, video_url, cover_url, duration, tags, music)
		 VALUES (?,?,?,?,?,?,?,?)`,
		authorID, v.Title, v.Description, v.VideoURL, v.CoverURL, v.Duration, v.Tags, defaultStr(v.Music, "原声"))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *VideoRepo) Delete(id, authorID int64) error {
	_, err := r.db.Exec(`DELETE FROM videos WHERE id=? AND author_id=?`, id, authorID)
	return err
}

func (r *VideoRepo) IncrementPlays(id int64) {
	_, _ = r.db.Exec(`UPDATE videos SET plays = plays + 1 WHERE id=?`, id)
}

func scanVideo(rows *sql.Rows, v *model.Video) error {
	return rows.Scan(&v.ID, &v.AuthorID, &v.AuthorName, &v.AuthorAvatar, &v.Title, &v.Description,
		&v.VideoURL, &v.CoverURL, &v.Duration, &v.Plays, &v.Likes, &v.CommentsCount, &v.Shares, &v.Tags, &v.Music, &v.CreatedAt)
}

func scanVideoRow(row *sql.Row, v *model.Video) error {
	return row.Scan(&v.ID, &v.AuthorID, &v.AuthorName, &v.AuthorAvatar, &v.Title, &v.Description,
		&v.VideoURL, &v.CoverURL, &v.Duration, &v.Plays, &v.Likes, &v.CommentsCount, &v.Shares, &v.Tags, &v.Music, &v.CreatedAt)
}

// annotateUserState marks liked/favorited flags for the given videos.
func (r *VideoRepo) annotateUserState(vs []model.Video, uid int64) {
	if uid <= 0 {
		return
	}
	for i := range vs {
		var liked, fav int
		_ = r.db.QueryRow(`SELECT 1 FROM likes WHERE user_id=? AND video_id=?`, uid, vs[i].ID).Scan(&liked)
		_ = r.db.QueryRow(`SELECT 1 FROM favorites WHERE user_id=? AND video_id=?`, uid, vs[i].ID).Scan(&fav)
		vs[i].Liked = liked == 1
		vs[i].Favorited = fav == 1
	}
}

// ===================== Likes =====================

type LikeRepo struct{ db *sql.DB }

func NewLikeRepo(db *sql.DB) *LikeRepo { return &LikeRepo{db: db} }

// Toggle adds or removes a like. Returns the new total and whether it's now liked.
func (r *LikeRepo) Toggle(userID, videoID int64) (int, bool, error) {
	var existing int
	err := r.db.QueryRow(`SELECT 1 FROM likes WHERE user_id=? AND video_id=?`, userID, videoID).Scan(&existing)
	if err == nil {
		// already liked -> unlike
		if _, err := r.db.Exec(`DELETE FROM likes WHERE user_id=? AND video_id=?`, userID, videoID); err != nil {
			return 0, false, err
		}
		if _, err := r.db.Exec(`UPDATE videos SET likes = MAX(0, likes - 1) WHERE id=?`, videoID); err != nil {
			return 0, false, err
		}
		var n int
		_ = r.db.QueryRow(`SELECT likes FROM videos WHERE id=?`, videoID).Scan(&n)
		return n, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, err
	}
	// not liked -> like
	if _, err := r.db.Exec(`INSERT INTO likes (user_id, video_id) VALUES (?,?)`, userID, videoID); err != nil {
		return 0, false, err
	}
	if _, err := r.db.Exec(`UPDATE videos SET likes = likes + 1 WHERE id=?`, videoID); err != nil {
		return 0, false, err
	}
	var n int
	_ = r.db.QueryRow(`SELECT likes FROM videos WHERE id=?`, videoID).Scan(&n)
	return n, true, nil
}

// ===================== Comments =====================

type CommentRepo struct{ db *sql.DB }

func NewCommentRepo(db *sql.DB) *CommentRepo { return &CommentRepo{db: db} }

func (r *CommentRepo) ListByVideo(videoID int64, limit int) ([]model.Comment, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Query(
		`SELECT id, video_id, user_id, username, avatar, content, likes, parent_id, created_at
		 FROM comments WHERE video_id=? ORDER BY id DESC LIMIT ?`, videoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Comment{}
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.VideoID, &c.UserID, &c.Username, &c.Avatar, &c.Content, &c.Likes, &c.ParentID, &c.CreatedAt); err == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *CommentRepo) Create(c *model.Comment) error {
	res, err := r.db.Exec(
		`INSERT INTO comments (video_id, user_id, username, avatar, content, parent_id) VALUES (?,?,?,?,?,?)`,
		c.VideoID, c.UserID, c.Username, c.Avatar, c.Content, c.ParentID)
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	_, _ = r.db.Exec(`UPDATE videos SET comments_count = comments_count + 1 WHERE id=?`, c.VideoID)
	return nil
}

func (r *CommentRepo) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM comments`).Scan(&n)
	return n, err
}

// ===================== Follows =====================

type FollowRepo struct{ db *sql.DB }

func NewFollowRepo(db *sql.DB) *FollowRepo { return &FollowRepo{db: db} }

// Toggle follows or unfollows. Returns whether now following.
func (r *FollowRepo) Toggle(followerID, followeeID int64) (bool, error) {
	if followerID == followeeID {
		return false, fmt.Errorf("cannot follow yourself")
	}
	var existing int
	err := r.db.QueryRow(`SELECT 1 FROM follows WHERE follower_id=? AND followee_id=?`, followerID, followeeID).Scan(&existing)
	if err == nil {
		_, err := r.db.Exec(`DELETE FROM follows WHERE follower_id=? AND followee_id=?`, followerID, followeeID)
		return false, err
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	_, err = r.db.Exec(`INSERT INTO follows (follower_id, followee_id) VALUES (?,?)`, followerID, followeeID)
	return true, err
}

func (r *FollowRepo) IsFollowing(followerID, followeeID int64) bool {
	var n int
	_ = r.db.QueryRow(`SELECT 1 FROM follows WHERE follower_id=? AND followee_id=?`, followerID, followeeID).Scan(&n)
	return n == 1
}

func (r *FollowRepo) ListFollowers(userID int64) ([]model.User, error) {
	return r.listUsers(`SELECT u.id, u.username, u.nickname, u.avatar, u.bio, u.following_count, u.followers_count, u.likes_count, u.created_at
		FROM follows f JOIN users u ON u.id = f.follower_id WHERE f.followee_id=? ORDER BY f.id DESC`, userID)
}

func (r *FollowRepo) ListFollowing(userID int64) ([]model.User, error) {
	return r.listUsers(`SELECT u.id, u.username, u.nickname, u.avatar, u.bio, u.following_count, u.followers_count, u.likes_count, u.created_at
		FROM follows f JOIN users u ON u.id = f.followee_id WHERE f.follower_id=? ORDER BY f.id DESC`, userID)
}

func (r *FollowRepo) listUsers(q string, args ...any) ([]model.User, error) {
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Nickname, &u.Avatar, &u.Bio, &u.FollowingCount, &u.FollowersCount, &u.LikesCount, &u.CreatedAt); err == nil {
			out = append(out, u)
		}
	}
	return out, nil
}

// ===================== Favorites =====================

type FavoriteRepo struct{ db *sql.DB }

func NewFavoriteRepo(db *sql.DB) *FavoriteRepo { return &FavoriteRepo{db: db} }

func (r *FavoriteRepo) Toggle(userID, videoID int64) (bool, error) {
	var existing int
	err := r.db.QueryRow(`SELECT 1 FROM favorites WHERE user_id=? AND video_id=?`, userID, videoID).Scan(&existing)
	if err == nil {
		_, err := r.db.Exec(`DELETE FROM favorites WHERE user_id=? AND video_id=?`, userID, videoID)
		return false, err
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	_, err = r.db.Exec(`INSERT INTO favorites (user_id, video_id) VALUES (?,?)`, userID, videoID)
	return true, err
}

// ===================== helpers =====================

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

var _ = time.Now
