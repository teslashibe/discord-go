package discord

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// initLogStore opens (or creates) <storeDir>/messages.db. Idempotent.
func (c *Client) initLogStore(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.logDB != nil {
		return nil
	}
	if err := os.MkdirAll(c.storeDir, 0o700); err != nil {
		return fmt.Errorf("%w: mkdir %s: %v", ErrStoreInit, c.storeDir, err)
	}
	dsn := "file:" + filepath.Join(c.storeDir, "messages.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(2000)&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("%w: open: %v", ErrStoreInit, err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("%w: ping: %v", ErrStoreInit, err)
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return fmt.Errorf("%w: migrate: %v", ErrStoreInit, err)
	}
	c.logDB = db
	return nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS messages (
    rowid       INTEGER PRIMARY KEY AUTOINCREMENT,
    msg_id      TEXT NOT NULL,
    channel_id  TEXT NOT NULL,
    guild_id    TEXT,
    author_id   TEXT,
    author_name TEXT,
    is_from_me  INTEGER NOT NULL DEFAULT 0,
    body        TEXT,
    has_media   INTEGER NOT NULL DEFAULT 0,
    reply_to_id TEXT,
    timestamp   INTEGER NOT NULL,
    edited_at   INTEGER,
    created_at  INTEGER NOT NULL,
    UNIQUE(channel_id, msg_id)
);
CREATE INDEX IF NOT EXISTS idx_msg_chan_ts ON messages(channel_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_msg_guild   ON messages(guild_id);
CREATE INDEX IF NOT EXISTS idx_msg_body    ON messages(body);

CREATE TABLE IF NOT EXISTS channels (
    id          TEXT PRIMARY KEY,
    guild_id    TEXT,
    name        TEXT,
    type        TEXT,
    last_seen   INTEGER
);`
	_, err := db.ExecContext(ctx, schema)
	return err
}

// upsertMessage inserts (or refreshes) a message, idempotent on
// (channel_id, msg_id).
func (c *Client) upsertMessage(ctx context.Context, m Message) error {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return nil
	}
	const q = `
INSERT INTO messages
  (msg_id, channel_id, guild_id, author_id, author_name, is_from_me,
   body, has_media, reply_to_id, timestamp, edited_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(channel_id, msg_id) DO UPDATE SET
  body        = excluded.body,
  has_media   = excluded.has_media,
  edited_at   = excluded.edited_at,
  reply_to_id = excluded.reply_to_id`
	var editedAt any
	if !m.EditedAt.IsZero() {
		editedAt = m.EditedAt.UnixMilli()
	}
	_, err := db.ExecContext(ctx, q,
		m.ID, m.ChannelID, nullable(m.GuildID),
		nullable(m.Author.ID), nullable(displayNameOf(m.Author)),
		boolToInt(m.IsFromMe), nullable(m.Body), boolToInt(m.HasMedia),
		nullable(m.ReplyToID), m.Timestamp.UnixMilli(), editedAt,
		time.Now().UnixMilli(),
	)
	return err
}

// deleteMessageRow removes a row by (channel_id, msg_id).
func (c *Client) deleteMessageRow(ctx context.Context, channelID, msgID string) {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return
	}
	_, _ = db.ExecContext(ctx, `DELETE FROM messages WHERE channel_id = ? AND msg_id = ?`, channelID, msgID)
}

// upsertChannelMeta records channel-level info (used by ListChannels /
// ListDMChannels) so Watch can present last-seen times.
func (c *Client) upsertChannelMeta(ctx context.Context, ch Channel) {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return
	}
	_, _ = db.ExecContext(ctx, `
INSERT INTO channels (id, guild_id, name, type, last_seen)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  guild_id = excluded.guild_id,
  name     = excluded.name,
  type     = excluded.type,
  last_seen = MAX(COALESCE(channels.last_seen, 0), excluded.last_seen)`,
		ch.ID, nullable(ch.GuildID), ch.Name, ch.Type, time.Now().UnixMilli(),
	)
}

func (c *Client) storedMessageCount(ctx context.Context) int64 {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return 0
	}
	var n int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages`).Scan(&n); err != nil {
		return 0
	}
	return n
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func displayNameOf(u User) string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}
