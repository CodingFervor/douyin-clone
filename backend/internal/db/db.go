package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create data dir: %w", err)
		}
	}
	var err error
	DB, err = sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(1)
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	if err = createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func createTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			nickname TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			bio TEXT NOT NULL DEFAULT '',
			following_count INTEGER NOT NULL DEFAULT 0,
			followers_count INTEGER NOT NULL DEFAULT 0,
			likes_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS videos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			author_id INTEGER NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			video_url TEXT NOT NULL,
			cover_url TEXT NOT NULL DEFAULT '',
			duration INTEGER NOT NULL DEFAULT 0,
			plays INTEGER NOT NULL DEFAULT 0,
			likes INTEGER NOT NULL DEFAULT 0,
			comments_count INTEGER NOT NULL DEFAULT 0,
			shares INTEGER NOT NULL DEFAULT 0,
			tags TEXT NOT NULL DEFAULT '',
			music TEXT NOT NULL DEFAULT '原声',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_author ON videos(author_id)`,
		`CREATE TABLE IF NOT EXISTS likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			video_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, video_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_likes_video ON likes(video_id)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			video_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			likes INTEGER NOT NULL DEFAULT 0,
			parent_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_video ON comments(video_id)`,
		`CREATE TABLE IF NOT EXISTS follows (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			follower_id INTEGER NOT NULL,
			followee_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(follower_id, followee_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_follows_follower ON follows(follower_id)`,
		`CREATE INDEX IF NOT EXISTS idx_follows_followee ON follows(followee_id)`,
		`CREATE TABLE IF NOT EXISTS favorites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			video_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, video_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id)`,
		// Play records: track how much of each video a user watched (completion ratio),
		// used by the recommendation engine.
		`CREATE TABLE IF NOT EXISTS play_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			video_id INTEGER NOT NULL,
			completion REAL NOT NULL DEFAULT 0,  -- 0.0 ~ 1.0 ratio watched
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, video_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_plays_user ON play_records(user_id)`,
		// Notifications: like/comment/follow/system events directed at a user.
		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,            -- recipient
			actor_id INTEGER NOT NULL,           -- who performed the action
			actor_name TEXT NOT NULL DEFAULT '',
			actor_avatar TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL,                  -- like, comment, follow, system
			video_id INTEGER NOT NULL DEFAULT 0,
			content TEXT NOT NULL DEFAULT '',
			is_read INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id)`,
		// Live rooms: a mock live-streaming room (no real RTMP; serves an HLS sample).
		`CREATE TABLE IF NOT EXISTS live_rooms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER NOT NULL,
			host_name TEXT NOT NULL DEFAULT '',
			host_avatar TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			cover_url TEXT NOT NULL DEFAULT '',
			stream_url TEXT NOT NULL DEFAULT '',
			viewers INTEGER NOT NULL DEFAULT 0,
			likes INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'live',
			category TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_live_status ON live_rooms(status)`,
		// Live products: items pinned in a live room (小黄车).
		`CREATE TABLE IF NOT EXISTS live_products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			live_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			price REAL NOT NULL,
			image TEXT NOT NULL DEFAULT '',
			sales INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_live_products_live ON live_products(live_id)`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return fmt.Errorf("exec: %w", err)
		}
	}
	return nil
}
