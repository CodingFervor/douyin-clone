package repository

import (
	"database/sql"
	"time"
)

// ===================== Live danmaku (chat messages) =====================

// LiveMessage is a chat/danmaku message in a live room.
type LiveMessage struct {
	ID        int64     `json:"id"`
	LiveID    int64     `json:"live_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Avatar    string    `json:"avatar"`
	Content   string    `json:"content"`
	IsSystem  int       `json:"is_system"`
	CreatedAt time.Time `json:"created_at"`
}

type DanmakuRepo struct{ db *sql.DB }

func NewDanmakuRepo(db *sql.DB) *DanmakuRepo { return &DanmakuRepo{db: db} }

// Send stores a chat message and returns it.
func (r *DanmakuRepo) Send(liveID, userID int64, username, avatar, content string) (*LiveMessage, error) {
	m := &LiveMessage{LiveID: liveID, UserID: userID, Username: username, Avatar: avatar, Content: content}
	res, err := r.db.Exec(
		`INSERT INTO live_messages (live_id, user_id, username, avatar, content) VALUES (?,?,?,?,?)`,
		liveID, userID, username, avatar, content)
	if err != nil {
		return nil, err
	}
	m.ID, _ = res.LastInsertId()
	return m, nil
}

// ListByLive returns recent messages for a room (oldest-first within the window).
func (r *DanmakuRepo) ListByLive(liveID int64, limit int) ([]LiveMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT id, live_id, user_id, username, avatar, content, is_system, created_at
		 FROM live_messages WHERE live_id=? ORDER BY id DESC LIMIT ?`, liveID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveMessage{}
	for rows.Next() {
		var m LiveMessage
		if err := rows.Scan(&m.ID, &m.LiveID, &m.UserID, &m.Username, &m.Avatar, &m.Content, &m.IsSystem, &m.CreatedAt); err == nil {
			out = append(out, m)
		}
	}
	// Reverse so newest are at the tail (chronological order).
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// ===================== Search logs / hot search =====================

type SearchLogRepo struct{ db *sql.DB }

func NewSearchLogRepo(db *sql.DB) *SearchLogRepo { return &SearchLogRepo{db: db} }

// Log records a user search (best-effort).
func (r *SearchLogRepo) Log(keyword string, userID int64) {
	kw := trim(keyword)
	if kw == "" {
		return
	}
	_, _ = r.db.Exec(`INSERT INTO search_logs (keyword, user_id) VALUES (?,?)`, kw, userID)
}

// HotSearch returns the top-N most-searched keywords in the recent window,
// with their search counts (ranked by count desc).
func (r *SearchLogRepo) HotSearch(limit int) ([]HotSearchItem, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(
		`SELECT keyword, COUNT(*) AS cnt FROM search_logs
		 WHERE created_at >= datetime('now', '-7 days')
		 GROUP BY keyword ORDER BY cnt DESC, keyword LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HotSearchItem{}
	for rows.Next() {
		var h HotSearchItem
		if err := rows.Scan(&h.Keyword, &h.Count); err == nil {
			out = append(out, h)
		}
	}
	return out, nil
}

// HotSearchItem is one ranked hot-search entry.
type HotSearchItem struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

// trim is a tiny helper to avoid importing strings here.
func trim(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

var _ = sql.ErrNoRows
