package repository

import (
	"database/sql"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
)

// Notification is the notification DTO (defined here to keep model.go lean).
type Notification struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	ActorID     int64  `json:"actor_id"`
	ActorName   string `json:"actor_name"`
	ActorAvatar string `json:"actor_avatar"`
	Type        string `json:"type"` // like, comment, follow, system
	VideoID     int64  `json:"video_id"`
	Content     string `json:"content"`
	IsRead      int    `json:"is_read"`
	CreatedAt   string `json:"created_at"`
}

type NotifyRepo struct{ db *sql.DB }

func NewNotifyRepo(db *sql.DB) *NotifyRepo { return &NotifyRepo{db: db} }

// ListByUser returns a user's notifications, newest first.
func (r *NotifyRepo) ListByUser(userID int64, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT id, user_id, actor_id, actor_name, actor_avatar, type, video_id, content, is_read, created_at
		 FROM notifications WHERE user_id=? ORDER BY id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.ActorName, &n.ActorAvatar, &n.Type, &n.VideoID, &n.Content, &n.IsRead, &n.CreatedAt); err == nil {
			out = append(out, n)
		}
	}
	return out, nil
}

// ListByUserAndType returns a user's notifications filtered by type.
func (r *NotifyRepo) ListByUserAndType(userID int64, ntype string, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT id, user_id, actor_id, actor_name, actor_avatar, type, video_id, content, is_read, created_at
		 FROM notifications WHERE user_id=? AND type=? ORDER BY id DESC LIMIT ?`, userID, ntype, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.ActorName, &n.ActorAvatar, &n.Type, &n.VideoID, &n.Content, &n.IsRead, &n.CreatedAt); err == nil {
			out = append(out, n)
		}
	}
	return out, nil
}

// Notify inserts a notification for `userID` from actor `actor`.
func (r *NotifyRepo) Notify(userID, actorID int64, actorName, actorAvatar, ntype string, videoID int64, content string) error {
	if userID == actorID {
		return nil // don't notify yourself
	}
	_, err := r.db.Exec(
		`INSERT INTO notifications (user_id, actor_id, actor_name, actor_avatar, type, video_id, content) VALUES (?,?,?,?,?,?,?)`,
		userID, actorID, actorName, actorAvatar, ntype, videoID, content)
	return err
}

// MarkAllRead sets all of a user's notifications as read.
func (r *NotifyRepo) MarkAllRead(userID int64) error {
	_, err := r.db.Exec(`UPDATE notifications SET is_read=1 WHERE user_id=?`, userID)
	return err
}

// UnreadCount returns the number of unread notifications.
func (r *NotifyRepo) UnreadCount(userID int64) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=? AND is_read=0`, userID).Scan(&n)
	return n, err
}

// Counts returns the per-type unread counts (for the Messages tab badges).
func (r *NotifyRepo) Counts(userID int64) (map[string]int, error) {
	rows, err := r.db.Query(
		`SELECT type, COUNT(*) FROM notifications WHERE user_id=? AND is_read=0 GROUP BY type`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{"like": 0, "comment": 0, "follow": 0, "system": 0}
	for rows.Next() {
		var t string
		var c int
		if rows.Scan(&t, &c) == nil {
			out[t] = c
		}
	}
	return out, nil
}

// _ keeps the unused import guard in case model is referenced later.
var _ = model.User{}
