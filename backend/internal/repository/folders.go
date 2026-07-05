package repository

import (
	"database/sql"
	"time"
)

// ===================== Favorite folders (收藏夹分组) =====================

// FavoriteFolder is a user-created collection.
type FavoriteFolder struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	CoverURL  string    `json:"cover_url"`
	Count     int       `json:"count"` // number of videos in this folder
	CreatedAt time.Time `json:"created_at"`
}

type FolderRepo struct{ db *sql.DB }

func NewFolderRepo(db *sql.DB) *FolderRepo { return &FolderRepo{db: db} }

// Create makes a new folder.
func (r *FolderRepo) Create(f *FavoriteFolder) error {
	res, err := r.db.Exec(`INSERT INTO favorite_folders (user_id, name, cover_url) VALUES (?,?,?)`, f.UserID, f.Name, f.CoverURL)
	if err != nil {
		return err
	}
	f.ID, _ = res.LastInsertId()
	return nil
}

// ListByUser returns a user's folders, with the count of favorited videos.
func (r *FolderRepo) ListByUser(userID int64) ([]FavoriteFolder, error) {
	rows, err := r.db.Query(
		`SELECT f.id, f.user_id, f.name, f.cover_url, f.created_at,
		        (SELECT COUNT(*) FROM favorites fav WHERE fav.user_id = f.user_id) AS cnt
		 FROM favorite_folders f WHERE f.user_id=? ORDER BY f.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FavoriteFolder{}
	for rows.Next() {
		var f FavoriteFolder
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.CoverURL, &f.CreatedAt, &f.Count); err == nil {
			out = append(out, f)
		}
	}
	return out, nil
}

// ===================== Suggested follows (关注用户推荐) =====================

// SuggestedUser is a user recommended for following.
type SuggestedUser struct {
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Followers int    `json:"followers_count"`
	Bio       string `json:"bio"`
}

// SuggestFollows returns popular users the current user doesn't follow yet.
func (r *UserRepo) SuggestFollows(userID int64, limit int) ([]SuggestedUser, error) {
	if limit <= 0 {
		limit = 6
	}
	rows, err := r.db.Query(
		`SELECT u.id, u.nickname, u.avatar, u.followers_count, u.bio
		 FROM users u
		 WHERE u.id != ? AND u.id NOT IN (SELECT followee_id FROM follows WHERE follower_id=?)
		 ORDER BY u.followers_count DESC LIMIT ?`, userID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SuggestedUser{}
	for rows.Next() {
		var s SuggestedUser
		if err := rows.Scan(&s.ID, &s.Nickname, &s.Avatar, &s.Followers, &s.Bio); err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}
