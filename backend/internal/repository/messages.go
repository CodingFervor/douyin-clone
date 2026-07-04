package repository

import (
	"database/sql"
	"time"
)

// ===================== Private messages (私信) =====================

// PrivateMessage is a 1-to-1 direct message.
type PrivateMessage struct {
	ID           int64     `json:"id"`
	SenderID     int64     `json:"sender_id"`
	ReceiverID   int64     `json:"receiver_id"`
	Content      string    `json:"content"`
	IsRead       int       `json:"is_read"`
	SenderName   string    `json:"sender_name"` // joined
	SenderAvatar string    `json:"sender_avatar"`
	CreatedAt    time.Time `json:"created_at"`
}

type MessageRepo struct{ db *sql.DB }

func NewMessageRepo(db *sql.DB) *MessageRepo { return &MessageRepo{db: db} }

// Send records a direct message from sender to receiver.
func (r *MessageRepo) Send(senderID, receiverID int64, content string) (*PrivateMessage, error) {
	res, err := r.db.Exec(
		`INSERT INTO private_messages (sender_id, receiver_id, content) VALUES (?,?,?)`,
		senderID, receiverID, content)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &PrivateMessage{ID: id, SenderID: senderID, ReceiverID: receiverID, Content: content}, nil
}

// Conversation returns the message thread between two users (both directions),
// oldest-first within the window.
func (r *MessageRepo) Conversation(a, b int64, limit int) ([]PrivateMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT pm.id, pm.sender_id, pm.receiver_id, pm.content, pm.is_read, pm.created_at, u.nickname, u.avatar
		 FROM private_messages pm JOIN users u ON u.id = pm.sender_id
		 WHERE (pm.sender_id=? AND pm.receiver_id=?) OR (pm.sender_id=? AND pm.receiver_id=?)
		 ORDER BY pm.id DESC LIMIT ?`, a, b, b, a, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PrivateMessage{}
	for rows.Next() {
		var m PrivateMessage
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.IsRead, &m.CreatedAt, &m.SenderName, &m.SenderAvatar); err == nil {
			out = append(out, m)
		}
	}
	// Reverse to chronological.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// MarkRead marks all messages from `other` to `me` as read.
func (r *MessageRepo) MarkRead(me, other int64) error {
	_, err := r.db.Exec(`UPDATE private_messages SET is_read=1 WHERE receiver_id=? AND sender_id=?`, me, other)
	return err
}

// ConversationList returns the user's conversation partners, newest message first.
type ConversationSummary struct {
	OtherID     int64     `json:"other_id"`
	OtherName   string    `json:"other_name"`
	OtherAvatar string    `json:"other_avatar"`
	LastMessage string    `json:"last_message"`
	Unread      int       `json:"unread"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (r *MessageRepo) ConversationList(userID int64) ([]ConversationSummary, error) {
	rows, err := r.db.Query(
		`SELECT
		    CASE WHEN sender_id=? THEN receiver_id ELSE sender_id END AS other_id,
		    u.nickname, u.avatar,
		    (SELECT content FROM private_messages pm2
		     WHERE (pm2.sender_id=? AND pm2.receiver_id = CASE WHEN sender_id=? THEN receiver_id ELSE sender_id END)
		        OR (pm2.receiver_id=? AND pm2.sender_id = CASE WHEN sender_id=? THEN receiver_id ELSE sender_id END)
		     ORDER BY pm2.id DESC LIMIT 1) AS last_msg,
		    (SELECT MAX(id) FROM private_messages pm3
		     WHERE (pm3.sender_id=? AND pm3.receiver_id = CASE WHEN sender_id=? THEN receiver_id ELSE sender_id END)
		        OR (pm3.receiver_id=? AND pm3.sender_id = CASE WHEN sender_id=? THEN receiver_id ELSE sender_id END)) AS max_id,
		    COUNT(CASE WHEN receiver_id=? AND is_read=0 THEN 1 END) AS unread
		 FROM private_messages pm
		 JOIN users u ON u.id = (CASE WHEN pm.sender_id=? THEN pm.receiver_id ELSE pm.sender_id END)
		 WHERE pm.sender_id=? OR pm.receiver_id=?
		 GROUP BY other_id
		 ORDER BY max_id DESC`, userID, userID, userID, userID, userID, userID, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConversationSummary{}
	for rows.Next() {
		var s ConversationSummary
		if err := rows.Scan(&s.OtherID, &s.OtherName, &s.OtherAvatar, &s.LastMessage, &s.UpdatedAt, &s.Unread); err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

var _ = sql.ErrNoRows
