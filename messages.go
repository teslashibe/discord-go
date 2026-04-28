package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// GetMessages returns up to params.Limit messages from a channel. When
// params.FromLocal is true the result is served from the local SQLite
// log only (no REST round trip). Otherwise the messages come from the
// Discord API and are written through to the local log.
func (c *Client) GetMessages(ctx context.Context, params MessageListParams) ([]Message, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("%w: ChannelID required", ErrInvalidParams)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		// Discord's REST cap is 100.
		limit = 100
	}
	if params.FromLocal {
		return c.getMessagesLocal(ctx, params.ChannelID, limit, params.BeforeID)
	}

	var out []Message
	err := c.withSession(func(s *discordgo.Session) error {
		msgs, err := s.ChannelMessages(params.ChannelID, limit, params.BeforeID, params.AfterID, params.AroundID)
		if err != nil {
			return fmt.Errorf("ChannelMessages: %w", err)
		}
		c.mu.RLock()
		selfID := ""
		if c.selfUser != nil {
			selfID = c.selfUser.ID
		}
		c.mu.RUnlock()
		for _, dm := range msgs {
			m := convertMessage(dm)
			if selfID != "" && m.Author.ID == selfID {
				m.IsFromMe = true
			}
			out = append(out, m)
			_ = c.upsertMessage(ctx, m)
		}
		return nil
	})
	return out, err
}

// GetMessage fetches a single message by id.
func (c *Client) GetMessage(ctx context.Context, channelID, messageID string) (Message, error) {
	if channelID == "" || messageID == "" {
		return Message{}, fmt.Errorf("%w: channelId and messageId required", ErrInvalidParams)
	}
	var out Message
	err := c.withSession(func(s *discordgo.Session) error {
		dm, err := s.ChannelMessage(channelID, messageID)
		if err != nil {
			return fmt.Errorf("ChannelMessage: %w", err)
		}
		out = convertMessage(dm)
		c.mu.RLock()
		selfID := ""
		if c.selfUser != nil {
			selfID = c.selfUser.ID
		}
		c.mu.RUnlock()
		if selfID != "" && out.Author.ID == selfID {
			out.IsFromMe = true
		}
		_ = c.upsertMessage(ctx, out)
		return nil
	})
	return out, err
}

// getMessagesLocal serves messages from the SQLite log only.
func (c *Client) getMessagesLocal(ctx context.Context, channelID string, limit int, beforeID string) ([]Message, error) {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return nil, ErrNotConnected
	}
	// Audit fix (H1): Snowflakes are 17–20-digit numbers and lexicographic
	// comparison on msg_id breaks across different lengths (older accounts
	// have shorter ids). Cast to INTEGER for the pagination predicate so
	// "before this id" actually means "older than this id".
	q := `SELECT rowid, msg_id, channel_id, COALESCE(guild_id,''),
              COALESCE(author_id,''), COALESCE(author_name,''), is_from_me,
              COALESCE(body,''), has_media, COALESCE(reply_to_id,''),
              timestamp, COALESCE(edited_at, 0)
         FROM messages WHERE channel_id = ?`
	args := []any{channelID}
	if beforeID != "" {
		q += ` AND CAST(msg_id AS INTEGER) < CAST(? AS INTEGER)`
		args = append(args, beforeID)
	}
	q += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		var ts, edited int64
		var hasMedia, isFromMe int
		if err := rows.Scan(&m.RowID, &m.ID, &m.ChannelID, &m.GuildID,
			&m.Author.ID, &m.Author.Username, &isFromMe, &m.Body, &hasMedia,
			&m.ReplyToID, &ts, &edited); err != nil {
			return nil, err
		}
		m.IsFromMe = isFromMe == 1
		m.HasMedia = hasMedia == 1
		m.Timestamp = time.UnixMilli(ts).UTC()
		if edited > 0 {
			m.EditedAt = time.UnixMilli(edited).UTC()
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
