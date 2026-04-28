package discord

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Watch returns messages newer than params.SinceRowID from the local
// SQLite log. Use result.Cursor as the next call's SinceRowID for
// stable polling.
//
// When Poll=true, Watch first refreshes the listed channels via REST
// (so genuinely-new messages get into the local log before we read it
// out). Use this for "I want fresh messages right now"; leave Poll=false
// for "give me everything I've already pulled since last time".
//
// PollChannels caps how many channels Watch will poll in one call —
// each poll is one REST round trip, so don't pass your entire DM list
// without a budget.
//
// When ChannelID/GuildID is set the cursor still advances against the
// global rowid space, so a second call with a different filter may
// silently skip rows; keep filters stable for a gap-free per-channel
// feed. Watch never blocks.
func (c *Client) Watch(ctx context.Context, params WatchParams) (WatchResult, error) {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return WatchResult{Messages: []Message{}}, ErrNotConnected
	}

	out := WatchResult{Messages: []Message{}, Cursor: params.SinceRowID}

	if params.Poll && len(params.PollChannels) > 0 {
		// Bound the work we'll do; user can call again to keep going.
		const maxPoll = 25
		channels := params.PollChannels
		if len(channels) > maxPoll {
			channels = channels[:maxPoll]
		}
		for _, ch := range channels {
			if ch == "" {
				continue
			}
			// Use after_id = highest snowflake we've already stored for
			// this channel so a busy channel with >25 new messages
			// doesn't silently drop the older ones (Discord paginates
			// from newest backwards by default).
			plp := MessageListParams{ChannelID: ch, Limit: 100}
			if last := c.lastSeenMessageID(ctx, ch); last != "" {
				plp.AfterID = last
			}
			if _, err := c.GetMessages(ctx, plp); err != nil {
				c.logger.Warn("watch: poll channel failed",
					zap.String("channel_id", ch),
					zap.Error(err))
				// Don't fail the whole Watch — the local cursor still
				// advances over anything we DID manage to fetch.
				continue
			}
			out.Polled++
		}
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
		return out, err
	}
	defer rows.Close()
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
