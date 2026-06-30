package repository

import (
	"database/sql"
	"strings"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
)

// ===================== Hashtags (#话题) =====================

type HashtagRepo struct{ db *sql.DB }

func NewHashtagRepo(db *sql.DB) *HashtagRepo { return &HashtagRepo{db: db} }

// Rebuild scans all videos' tags columns and rebuilds the hashtag usage counts.
// Called once at startup so the hot-hashtag list has data on a fresh seed.
func (r *HashtagRepo) Rebuild() {
	rows, err := r.db.Query(`SELECT tags FROM videos WHERE tags != ''`)
	if err != nil {
		return
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var tags string
		if rows.Scan(&tags) != nil {
			continue
		}
		for _, t := range splitTags(tags) {
			counts[t]++
		}
	}
	for name, n := range counts {
		_, _ = r.db.Exec(
			`INSERT INTO hashtags (name, uses) VALUES (?,?)
			 ON CONFLICT(name) DO UPDATE SET uses=excluded.uses`, name, n)
	}
}

// Top returns the most-used hashtags (ranked desc).
func (r *HashtagRepo) Top(limit int) ([]model.Hashtag, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, name, uses FROM hashtags ORDER BY uses DESC, name LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Hashtag{}
	for rows.Next() {
		var h model.Hashtag
		if rows.Scan(&h.ID, &h.Name, &h.Uses) == nil {
			out = append(out, h)
		}
	}
	return out, nil
}

// splitTags parses a comma- or space-separated tag string into clean tokens.
func splitTags(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '#' || r == '、' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ===================== Live gifts (礼物) =====================

type GiftRepo struct{ db *sql.DB }

func NewGiftRepo(db *sql.DB) *GiftRepo { return &GiftRepo{db: db} }

// List returns the live-room gift catalog (by sort order).
func (r *GiftRepo) List() ([]model.LiveGift, error) {
	rows, err := r.db.Query(
		`SELECT id, name, icon, price, sort_order FROM live_gifts ORDER BY sort_order, price`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.LiveGift{}
	for rows.Next() {
		var g model.LiveGift
		if rows.Scan(&g.ID, &g.Name, &g.Icon, &g.Price, &g.SortOrder) == nil {
			out = append(out, g)
		}
	}
	return out, nil
}

// SeedGifts populates a demo gift catalog if empty.
func (r *GiftRepo) SeedGifts() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM live_gifts`).Scan(&n)
	if n > 0 {
		return
	}
	gifts := []struct {
		name  string
		icon  string
		price int
	}{
		{"点赞", "👍", 1},
		{"玫瑰", "🌹", 10},
		{"爱心", "❤️", 52},
		{"蛋糕", "🎂", 199},
		{"皇冠", "👑", 520},
		{"跑车", "🏎️", 3000},
		{"火箭", "🚀", 10000},
	}
	for i, g := range gifts {
		_, _ = r.db.Exec(
			`INSERT INTO live_gifts (name, icon, price, sort_order) VALUES (?,?,?,?)`,
			g.name, g.icon, g.price, i)
	}
}

var _ = sql.ErrNoRows
