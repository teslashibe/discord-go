package discord

import (
	"context"
	"time"
)

// Watch returns messages newer than params.SinceRowID from the local
// SQLite log (populated by the gateway's MessageCreate / MessageUpdate
// events while the client is Connect-ed). Use result.Cursor as the
// next call's SinceRowID for stable polling.
//
// When ChannelID or GuildID is set the cursor still advances against
// the global rowid space, so a second call with a different filter may
// silently skip rows — keep filters stable across consecutive Watch
// calls if you want a gap-free per-channel feed. Watch never blocks.
func (c *Client) Watch(ctx context.Context, params WatchParams) (WatchResult, error) {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return WatchResult{Messages: []Message{}}, ErrNotConnected
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	q := `SELECT rowid, msg_id, channel_id, COALESCE(guild_id,''),
              COALESCE(author_id,''), COALESCE(author_name,''), is_from_me,
              COALESCE(body,''), has_media, COALESCE(reply_to_id,''),
              timestamp, COALESCE(edited_at, 0)
         FROM messages WHERE rowid > ?`
	args := []any{params.SinceRowID}
	if params.ChannelID != "" {
		q += ` AND channel_id = ?`
		args = append(args, params.ChannelID)
	}
	if params.GuildID != "" {
		q += ` AND guild_id = ?`
		args = append(args, params.GuildID)
	}
	q += ` ORDER BY rowid ASC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return WatchResult{Messages: []Message{}}, err
	}
	defer rows.Close()
	out := WatchResult{Messages: []Message{}, Cursor: params.SinceRowID}
	for rows.Next() {
		var m Message
		var ts, edited int64
		var hasMedia, isFromMe int
		if err := rows.Scan(&m.RowID, &m.ID, &m.ChannelID, &m.GuildID,
			&m.Author.ID, &m.Author.Username, &isFromMe, &m.Body, &hasMedia,
			&m.ReplyToID, &ts, &edited); err != nil {
			return out, err
		}
		m.IsFromMe = isFromMe == 1
		m.HasMedia = hasMedia == 1
		m.Timestamp = time.UnixMilli(ts).UTC()
		if edited > 0 {
			m.EditedAt = time.UnixMilli(edited).UTC()
		}
		out.Messages = append(out.Messages, m)
		if m.RowID > out.Cursor {
			out.Cursor = m.RowID
		}
	}
	return out, rows.Err()
}
