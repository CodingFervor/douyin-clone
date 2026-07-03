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
	City       string    `json:"city"`
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
		`SELECT id, host_id, host_name, host_avatar, title, cover_url, stream_url, viewers, likes, status, category, city, created_at
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
		`SELECT id, host_id, host_name, host_avatar, title, cover_url, stream_url, viewers, likes, status, category, city, created_at
		 FROM live_rooms WHERE id=?`, id).Scan(
		&lr.ID, &lr.HostID, &lr.HostName, &lr.HostAvatar, &lr.Title, &lr.CoverURL, &lr.StreamURL,
		&lr.Viewers, &lr.Likes, &lr.Status, &lr.Category, &lr.City, &lr.CreatedAt)
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
		{HostID: 2, HostName: "旅行的意义", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=travel", Title: "冰岛极光旅行直播中！", CoverURL: "https://picsum.photos/seed/live1/750/1000", StreamURL: hls, Viewers: 12800, Likes: 89000, Status: "live", Category: "旅行", City: "北京"},
		{HostID: 3, HostName: "吃货日记", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=food", Title: "深夜食堂·家常红烧肉教学", CoverURL: "https://picsum.photos/seed/live2/750/1000", StreamURL: hls, Viewers: 25600, Likes: 150000, Status: "live", Category: "美食", City: "成都"},
		{HostID: 4, HostName: "舞动青春", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=dance", Title: "热门舞蹈教学直播", CoverURL: "https://picsum.photos/seed/live3/750/1000", StreamURL: hls, Viewers: 8900, Likes: 56000, Status: "live", Category: "舞蹈", City: "上海"},
		{HostID: 5, HostName: "萌宠乐园", HostAvatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=pet", Title: "金毛在海边撒欢啦", CoverURL: "https://picsum.photos/seed/live4/750/1000", StreamURL: hls, Viewers: 15600, Likes: 98000, Status: "live", Category: "萌宠", City: "广州"},
	}
	for _, lr := range rooms {
		res, _ := r.db.Exec(
			`INSERT INTO live_rooms (host_id, host_name, host_avatar, title, cover_url, stream_url, viewers, likes, status, category, city) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			lr.HostID, lr.HostName, lr.HostAvatar, lr.Title, lr.CoverURL, lr.StreamURL, lr.Viewers, lr.Likes, lr.Status, lr.Category, lr.City)
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
		&lr.Viewers, &lr.Likes, &lr.Status, &lr.Category, &lr.City, &lr.CreatedAt)
}

// ListByCity returns live rooms in a given city (城市频道).
func (r *LiveRepo) ListByCity(city string, limit int) ([]LiveRoom, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, host_id, host_name, host_avatar, title, cover_url, stream_url, viewers, likes, status, category, city, created_at
		 FROM live_rooms WHERE status='live' AND city=? ORDER BY viewers DESC LIMIT ?`, city, limit)
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

// ListCities returns the distinct cities that have live rooms, with counts.
type CityCount struct {
	City  string `json:"city"`
	Count int    `json:"count"`
}

func (r *LiveRepo) ListCities() ([]CityCount, error) {
	rows, err := r.db.Query(
		`SELECT city, COUNT(*) AS cnt FROM live_rooms WHERE status='live' AND city != '' GROUP BY city ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CityCount{}
	for rows.Next() {
		var c CityCount
		if rows.Scan(&c.City, &c.Count) == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

// ===================== PK battles (直播PK) =====================

// PKBattle is a live-versus-live contest scored by gifts/likes.
type PKBattle struct {
	ID     int64  `json:"id"`
	RoomA  int64  `json:"room_a"`
	RoomB  int64  `json:"room_b"`
	ScoreA int    `json:"score_a"`
	ScoreB int    `json:"score_b"`
	Status string `json:"status"` // live, ended
	// Joined room names for display.
	RoomAName string `json:"room_a_name"`
	RoomBName string `json:"room_b_name"`
}

// StartPK creates a battle between two rooms (random pairing if room_b is 0).
func (r *LiveRepo) StartPK(roomA, roomB int64) (*PKBattle, error) {
	if roomB == 0 {
		// Pick a random other live room.
		_ = r.db.QueryRow(`SELECT id FROM live_rooms WHERE id != ? AND status='live' ORDER BY RANDOM() LIMIT 1`, roomA).Scan(&roomB)
	}
	if roomB == 0 {
		return nil, fmt.Errorf("没有可PK的直播间")
	}
	res, err := r.db.Exec(`INSERT INTO pk_battles (room_a, room_b) VALUES (?,?)`, roomA, roomB)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetPK(id)
}

// GetPK returns a battle with joined room names.
func (r *LiveRepo) GetPK(id int64) (*PKBattle, error) {
	pk := &PKBattle{}
	var aName, bName sql.NullString
	err := r.db.QueryRow(
		`SELECT pb.id, pb.room_a, pb.room_b, pb.score_a, pb.score_b, pb.status, a.host_name, b.host_name
		 FROM pk_battles pb
		 JOIN live_rooms a ON a.id = pb.room_a
		 JOIN live_rooms b ON b.id = pb.room_b
		 WHERE pb.id=?`, id,
	).Scan(&pk.ID, &pk.RoomA, &pk.RoomB, &pk.ScoreA, &pk.ScoreB, &pk.Status, &aName, &bName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	pk.RoomAName = aName.String
	pk.RoomBName = bName.String
	return pk, err
}

// GetActivePK returns the in-progress PK for a room (if any).
func (r *LiveRepo) GetActivePK(roomID int64) (*PKBattle, error) {
	pk := &PKBattle{}
	var aName, bName sql.NullString
	err := r.db.QueryRow(
		`SELECT pb.id, pb.room_a, pb.room_b, pb.score_a, pb.score_b, pb.status, a.host_name, b.host_name
		 FROM pk_battles pb
		 JOIN live_rooms a ON a.id = pb.room_a
		 JOIN live_rooms b ON b.id = pb.room_b
		 WHERE pb.status='live' AND (pb.room_a=? OR pb.room_b=?) ORDER BY pb.id DESC LIMIT 1`, roomID, roomID,
	).Scan(&pk.ID, &pk.RoomA, &pk.RoomB, &pk.ScoreA, &pk.ScoreB, &pk.Status, &aName, &bName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	pk.RoomAName = aName.String
	pk.RoomBName = bName.String
	return pk, err
}

// ScorePK adds points to one side of a battle.
func (r *LiveRepo) ScorePK(id int64, side string, points int) (*PKBattle, error) {
	if side == "b" {
		_, _ = r.db.Exec(`UPDATE pk_battles SET score_b = score_b + ? WHERE id=?`, points, id)
	} else {
		_, _ = r.db.Exec(`UPDATE pk_battles SET score_a = score_a + ? WHERE id=?`, points, id)
	}
	return r.GetPK(id)
}

// EndPK finalizes a battle.
func (r *LiveRepo) EndPK(id int64) error {
	_, err := r.db.Exec(`UPDATE pk_battles SET status='ended' WHERE id=?`, id)
	return err
}

// ===================== Fan guards (粉丝勋章/守护) =====================

// FanGuard is a user who "守护" a host.
type FanGuard struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	HostID     int64     `json:"host_id"`
	BadgeLevel int       `json:"badge_level"`
	Nickname   string    `json:"nickname"`
	Avatar     string    `json:"avatar"`
	CreatedAt  time.Time `json:"created_at"`
}

// Guard toggles a user's 守护 status for a host. Returns true if now guarding.
func (r *LiveRepo) Guard(userID, hostID int64) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`DELETE FROM fan_guards WHERE user_id=? AND host_id=?`, userID, hostID)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return false, tx.Commit()
	}
	if _, err := tx.Exec(`INSERT INTO fan_guards (user_id, host_id) VALUES (?,?)`, userID, hostID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// IsGuarding reports whether a user guards a host.
func (r *LiveRepo) IsGuarding(userID, hostID int64) bool {
	var n int
	_ = r.db.QueryRow(`SELECT 1 FROM fan_guards WHERE user_id=? AND host_id=?`, userID, hostID).Scan(&n)
	return n == 1
}

// GuardCount returns how many users guard a host.
func (r *LiveRepo) GuardCount(hostID int64) int {
	var n int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM fan_guards WHERE host_id=?`, hostID).Scan(&n)
	return n
}

// ListGuards returns the top fans guarding a host.
func (r *LiveRepo) ListGuards(hostID int64, limit int) ([]FanGuard, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT g.id, g.user_id, g.host_id, g.badge_level, u.nickname, u.avatar, g.created_at
		 FROM fan_guards g JOIN users u ON u.id = g.user_id
		 WHERE g.host_id=? ORDER BY g.badge_level DESC, g.id LIMIT ?`, hostID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FanGuard{}
	for rows.Next() {
		var g FanGuard
		if err := rows.Scan(&g.ID, &g.UserID, &g.HostID, &g.BadgeLevel, &g.Nickname, &g.Avatar, &g.CreatedAt); err == nil {
			out = append(out, g)
		}
	}
	return out, nil
}

// ===================== Red packets (红包雨) =====================

// RedPacket is a grab-bag of coins a host drops in a live room.
type RedPacket struct {
	ID        int64  `json:"id"`
	RoomID    int64  `json:"room_id"`
	Total     int    `json:"total"`
	Remaining int    `json:"remaining"`
	AmountPer int    `json:"amount_per"`
	Status    string `json:"status"` // active, ended
}

// DropPacket creates a red packet in a room.
func (r *LiveRepo) DropPacket(roomID int64, total, amountPer int) (*RedPacket, error) {
	if total <= 0 {
		total = 10
	}
	if amountPer <= 0 {
		amountPer = 10
	}
	res, err := r.db.Exec(
		`INSERT INTO red_packets (room_id, total, remaining, amount_per) VALUES (?,?,?,?)`,
		roomID, total, total, amountPer)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &RedPacket{ID: id, RoomID: roomID, Total: total, Remaining: total, AmountPer: amountPer, Status: "active"}, nil
}

// ActivePacket returns the in-progress red packet for a room (if any).
func (r *LiveRepo) ActivePacket(roomID int64) (*RedPacket, error) {
	p := &RedPacket{}
	err := r.db.QueryRow(
		`SELECT id, room_id, total, remaining, amount_per, status FROM red_packets
		 WHERE room_id=? AND status='active' AND remaining > 0 ORDER BY id DESC LIMIT 1`, roomID,
	).Scan(&p.ID, &p.RoomID, &p.Total, &p.Remaining, &p.AmountPer, &p.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// GrabPacket lets a user claim one share of a red packet (transactional: dedup
// via UNIQUE + decrement remaining). Returns the amount won, or 0 if sold out.
func (r *LiveRepo) GrabPacket(packetID, userID int64) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var remaining, amountPer int
	err = tx.QueryRow(`SELECT remaining, amount_per FROM red_packets WHERE id=? AND status='active'`, packetID).Scan(&remaining, &amountPer)
	if err != nil {
		return 0, fmt.Errorf("红包不存在")
	}
	if remaining <= 0 {
		return 0, fmt.Errorf("红包已抢完")
	}
	if _, err := tx.Exec(`INSERT INTO red_packet_claims (packet_id, user_id, amount) VALUES (?,?,?)`, packetID, userID, amountPer); err != nil {
		return 0, fmt.Errorf("您已抢过该红包")
	}
	if _, err := tx.Exec(`UPDATE red_packets SET remaining = remaining - 1 WHERE id=?`, packetID); err != nil {
		return 0, err
	}
	if remaining-1 <= 0 {
		if _, err := tx.Exec(`UPDATE red_packets SET status='ended' WHERE id=?`, packetID); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return amountPer, nil
}
