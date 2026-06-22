package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// LiveRoom is the DTO for a live-streaming room.
type LiveRoom struct {
	ID         int64     `json:"id"`
	HostID     int64     `json:"host_id"`
	HostName   string    `json:"host_name"`
	HostAvatar string    `json:"host_avatar"`
	Title      string    `json:"title"`
	CoverURL   string    `json:"cover_url"`
	StreamURL  string    `json:"stream_url"`
	Viewers    int       `json:"viewers"`
	Likes      int       `json:"likes"`
	Status     string    `json:"status"`
	Category   string    `json:"category"`
	CreatedAt  time.Time `json:"created_at"`
}

// LiveProduct is an item pinned in a live room (小黄车).
type LiveProduct struct {
	ID        int64   `json:"id"`
	LiveID    int64   `json:"live_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Image     string  `json:"image"`
	Sales     int     `json:"sales"`
	SortOrder int     `json:"sort_order"`
}

type LiveRepo struct{ db *sql.DB }

func NewLiveRepo(db *sql.DB) *LiveRepo { return &LiveRepo{db: db} }

// ListLive returns all currently-live rooms, hottest first.
func (r *LiveRepo) ListLive(limit int) ([]LiveRoom, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, host_id, host_name, host_avatar, title, cover_url, stream_url, viewers, likes, status, category, created_at
		 FROM live_rooms WHERE status='live' ORDER BY viewers DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveRoom{}
	for rows.Next() {
		var lr LiveRoom
		if err := scanLive(rows, &lr); err == nil {
			out = append(out, lr)
		}
	}
	return out, nil
}

// Get returns a single live room by id.
func (r *LiveRepo) Get(id int64) (*LiveRoom, error) {
	lr := &LiveRoom{}
	err := r.db.QueryRow(
		`SELECT id, host_id, host_name, host_avatar, title, cover_url, stream_url, viewers, likes, status, category, created_at
		 FROM live_rooms WHERE id=?`, id).Scan(
		&lr.ID, &lr.HostID, &lr.HostName, &lr.HostAvatar, &lr.Title, &lr.CoverURL, &lr.StreamURL,
		&lr.Viewers, &lr.Likes, &lr.Status, &lr.Category, &lr.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return lr, err
}

// ListProducts returns the pinned products (小黄车) for a live room.
func (r *LiveRepo) ListProducts(liveID int64) ([]LiveProduct, error) {
	rows, err := r.db.Query(
		`SELECT id, live_id, name, price, image, sales, sort_order FROM live_products WHERE live_id=? ORDER BY sort_order, id`, liveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveProduct{}
	for rows.Next() {
		var lp LiveProduct
		if err := rows.Scan(&lp.ID, &lp.LiveID, &lp.Name, &lp.Price, &lp.Image, &lp.Sales, &lp.SortOrder); err == nil {
			out = append(out, lp)
		}
	}
	return out, nil
}

// IncrementViewers bumps the viewer count when someone enters a room.
func (r *LiveRepo) IncrementViewers(id int64) {
	_, _ = r.db.Exec(`UPDATE live_rooms SET viewers = viewers + 1 WHERE id=?`, id)
}

// IncrementLikes bumps the like count.
func (r *LiveRepo) IncrementLikes(id int64) {
	_, _ = r.db.Exec(`UPDATE live_rooms SET likes = likes + 1 WHERE id=?`, id)
}

// SeedLive populates live rooms + products if empty (idempotent).
func (r *LiveRepo) SeedLive() {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM live_rooms`).Scan(&n)
	if n > 0 {
		return
	}
	// Use a public HLS test stream as the mock live feed.
	const hls = "https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8"
	rooms := []LiveRoom{
		{HostID: 2, HostName: "旅行的意义", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=travel", Title: "冰岛极光旅行直播中！", CoverURL: "https://picsum.photos/seed/live1/750/1000", StreamURL: hls, Viewers: 12800, Likes: 89000, Status: "live", Category: "旅行"},
		{HostID: 3, HostName: "吃货日记", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=food", Title: "深夜食堂·家常红烧肉教学", CoverURL: "https://picsum.photos/seed/live2/750/1000", StreamURL: hls, Viewers: 25600, Likes: 150000, Status: "live", Category: "美食"},
		{HostID: 4, HostName: "舞动青春", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=dance", Title: "热门舞蹈教学直播", CoverURL: "https://picsum.photos/seed/live3/750/1000", StreamURL: hls, Viewers: 8900, Likes: 56000, Status: "live", Category: "舞蹈"},
		{HostID: 5, HostName: "萌宠乐园", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=pet", Title: "金毛在海边撒欢啦", CoverURL: "https://picsum.photos/seed/live4/750/1000", StreamURL: hls, Viewers: 15600, Likes: 98000, Status: "live", Category: "萌宠"},
	}
	for _, lr := range rooms {
		res, _ := r.db.Exec(
			`INSERT INTO live_rooms (host_id, host_name, host_avatar, title, cover_url, stream_url, viewers, likes, status, category) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			lr.HostID, lr.HostName, lr.HostAvatar, lr.Title, lr.CoverURL, lr.StreamURL, lr.Viewers, lr.Likes, lr.Status, lr.Category)
		id, _ := res.LastInsertId()
		// Seed pinned products per room.
		products := []LiveProduct{
			{Name: fmt.Sprintf("%s同款好物A", lr.HostName), Price: 99.0, Image: "https://picsum.photos/seed/p" + fmt.Sprint(id) + "a/200/200", Sales: 1000},
			{Name: fmt.Sprintf("%s同款好物B", lr.HostName), Price: 199.0, Image: "https://picsum.photos/seed/p" + fmt.Sprint(id) + "b/200/200", Sales: 800},
			{Name: fmt.Sprintf("%s推荐好物C", lr.HostName), Price: 299.0, Image: "https://picsum.photos/seed/p" + fmt.Sprint(id) + "c/200/200", Sales: 500},
		}
		for i, p := range products {
			_, _ = r.db.Exec(
				`INSERT INTO live_products (live_id, name, price, image, sales, sort_order) VALUES (?,?,?,?,?,?)`,
				id, p.Name, p.Price, p.Image, p.Sales, i)
		}
	}
}

func scanLive(rows *sql.Rows, lr *LiveRoom) error {
	return rows.Scan(&lr.ID, &lr.HostID, &lr.HostName, &lr.HostAvatar, &lr.Title, &lr.CoverURL, &lr.StreamURL,
		&lr.Viewers, &lr.Likes, &lr.Status, &lr.Category, &lr.CreatedAt)
}
