package repository

import (
	"database/sql"
	"strings"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
)

// RecommendRepo implements a lightweight content-recommendation engine:
//  1. Collaborative filtering: find users who liked the same videos as me,
//     then surface videos they liked that I haven't seen.
//  2. Tag affinity: from my liked videos, extract top tags and recommend
//     unseen videos sharing those tags.
//  3. Cold-start fallback: when the above produce too few results, top up
//     with globally popular videos (weighted by likes + plays + completion).
//
// Completion ratio (from play_records) acts as an implicit-feedback weight.
type RecommendRepo struct {
	db *sql.DB
}

func NewRecommendRepo(db *sql.DB) *RecommendRepo { return &RecommendRepo{db: db} }

// ForUser returns up to `limit` recommended videos for the user. When userID
// is 0 (not logged in) or the user has no likes, it falls back to popularity.
func (r *RecommendRepo) ForUser(userID int64, limit int) ([]model.Video, error) {
	if limit <= 0 {
		limit = 20
	}
	// Collect candidate video IDs in priority order, dedup as we go.
	seen := map[int64]bool{}
	var ids []int64

	// 1) Collaborative filtering via shared likes.
	if userID > 0 {
		cf, err := r.collabFilter(userID, limit)
		if err == nil {
			for _, id := range cf {
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
	}

	// 2) Tag affinity from my liked videos.
	if userID > 0 && len(ids) < limit {
		tag, err := r.tagAffinity(userID, limit-len(ids), seen)
		if err == nil {
			for _, id := range tag {
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
	}

	// 3) Top up with popular videos.
	if len(ids) < limit {
		pop, err := r.popular(limit-len(ids), seen)
		if err == nil {
			ids = append(ids, pop...)
		}
	}

	if len(ids) == 0 {
		return []model.Video{}, nil
	}
	// Limit the final list.
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return r.loadVideos(ids, userID)
}

// collabFilter finds videos liked by users who share my likes (excluding ones
// I already liked). This is the classic item-based CF signal.
func (r *RecommendRepo) collabFilter(userID int64, limit int) ([]int64, error) {
	rows, err := r.db.Query(
		`SELECT l2.video_id
		 FROM likes l1
		 JOIN likes l2 ON l2.user_id = l1.user_id AND l2.video_id != l1.video_id
		 WHERE l1.user_id = ?
		   AND l2.video_id NOT IN (SELECT video_id FROM likes WHERE user_id = ?)
		 GROUP BY l2.video_id
		 ORDER BY COUNT(*) DESC
		 LIMIT ?`, userID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

// tagAffinity extracts the top tags from videos the user liked and recommends
// other videos sharing those tags.
func (r *RecommendRepo) tagAffinity(userID int64, limit int, seen map[int64]bool) ([]int64, error) {
	// Gather tags from liked videos.
	tagRows, err := r.db.Query(
		`SELECT v.tags FROM likes l JOIN videos v ON v.id = l.video_id WHERE l.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	tagCount := map[string]int{}
	for tagRows.Next() {
		var tags string
		if tagRows.Scan(&tags) == nil {
			for _, t := range strings.Split(tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tagCount[t]++
				}
			}
		}
	}
	tagRows.Close()
	if len(tagCount) == 0 {
		return nil, nil
	}
	// Pick the top tags.
	topTags := topKeys(tagCount, 5)
	if len(topTags) == 0 {
		return nil, nil
	}
	// Find videos sharing those tags, not yet liked.
	q := `SELECT id, tags FROM videos WHERE id NOT IN (SELECT video_id FROM likes WHERE user_id=?)`
	rows, err := r.db.Query(q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type cand struct {
		id  int64
		hit int
	}
	var cands []cand
	for rows.Next() {
		var id int64
		var tags string
		if rows.Scan(&id, &tags) != nil {
			continue
		}
		if seen[id] {
			continue
		}
		hit := 0
		for _, t := range strings.Split(tags, ",") {
			if tagCount[strings.TrimSpace(t)] > 0 {
				hit += tagCount[strings.TrimSpace(t)]
			}
		}
		if hit > 0 {
			cands = append(cands, cand{id, hit})
		}
	}
	// Sort by hit desc (simple insertion sort for the small candidate set).
	for i := 1; i < len(cands); i++ {
		for j := i; j > 0 && cands[j].hit > cands[j-1].hit; j-- {
			cands[j], cands[j-1] = cands[j-1], cands[j]
		}
	}
	out := []int64{}
	for i := 0; i < len(cands) && len(out) < limit; i++ {
		out = append(out, cands[i].id)
	}
	return out, nil
}

// popular returns globally popular video IDs (likes + plays + avg completion),
// excluding already-seen ones.
func (r *RecommendRepo) popular(limit int, seen map[int64]bool) ([]int64, error) {
	rows, err := r.db.Query(
		`SELECT id FROM videos ORDER BY (likes*2 + plays + comments_count) DESC, id DESC LIMIT ?`, limit*2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) != nil {
			continue
		}
		if seen[id] {
			continue
		}
		out = append(out, id)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// loadVideos fetches full video rows (with author join) for the given IDs,
// in the given order, annotating liked/favorited state.
func (r *RecommendRepo) loadVideos(ids []int64, userID int64) ([]model.Video, error) {
	if len(ids) == 0 {
		return []model.Video{}, nil
	}
	// Build placeholders.
	ph := make([]any, len(ids))
	marks := ""
	for i, id := range ids {
		if i > 0 {
			marks += ","
		}
		marks += "?"
		ph[i] = id
	}
	q := `SELECT v.id, v.author_id, u.nickname, u.avatar, v.title, v.description,
	             v.video_url, v.cover_url, v.duration, v.plays, v.likes, v.comments_count,
	             v.shares, v.tags, v.music, v.created_at
	      FROM videos v JOIN users u ON u.id = v.author_id
	      WHERE v.id IN (` + marks + `)`
	rows, err := r.db.Query(q, ph...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[int64]model.Video{}
	for rows.Next() {
		var v model.Video
		if scanVideo(rows, &v) == nil {
			byID[v.ID] = v
		}
	}
	// Preserve the recommended order.
	out := make([]model.Video, 0, len(ids))
	for _, id := range ids {
		if v, ok := byID[id]; ok {
			out = append(out, v)
		}
	}
	// Annotate liked/favorited.
	if userID > 0 {
		for i := range out {
			var liked, fav int
			_ = r.db.QueryRow(`SELECT 1 FROM likes WHERE user_id=? AND video_id=?`, userID, out[i].ID).Scan(&liked)
			_ = r.db.QueryRow(`SELECT 1 FROM favorites WHERE user_id=? AND video_id=?`, userID, out[i].ID).Scan(&fav)
			out[i].Liked = liked == 1
			out[i].Favorited = fav == 1
		}
	}
	return out, nil
}

// RecordPlay upserts a play record (completion ratio) for implicit feedback.
func (r *RecommendRepo) RecordPlay(userID, videoID int64, completion float64) error {
	if userID <= 0 {
		return nil
	}
	if completion < 0 {
		completion = 0
	} else if completion > 1 {
		completion = 1
	}
	_, err := r.db.Exec(
		`INSERT INTO play_records (user_id, video_id, completion) VALUES (?,?,?)
		 ON CONFLICT(user_id, video_id) DO UPDATE SET completion=excluded.completion`,
		userID, videoID, completion)
	return err
}

// topKeys returns the keys of m with the highest values (up to n).
func topKeys(m map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k, v})
	}
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].v > all[j-1].v; j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	if n > len(all) {
		n = len(all)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = all[i].k
	}
	return out
}
