package discord

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/teslashibe/discord-go/internal/transport"
)

// GetMessages returns up to params.Limit messages from a channel.
// Default = REST + write-through to local log; FromLocal=true reads
// from SQLite only.
func (c *Client) GetMessages(ctx context.Context, params MessageListParams) ([]Message, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("%w: ChannelID required", ErrInvalidParams)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100 // Discord cap
	}
	if params.FromLocal {
		return c.getMessagesLocal(ctx, params.ChannelID, limit, params.BeforeID)
	}
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	if params.BeforeID != "" {
		q.Set("before", params.BeforeID)
	}
	if params.AfterID != "" {
		q.Set("after", params.AfterID)
	}
	if params.AroundID != "" {
		q.Set("around", params.AroundID)
	}
	var raw []rawMessage
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet,
			"/api/v9/channels/"+url.PathEscape(params.ChannelID)+"/messages",
			nil, &raw, q)
	})
	if err != nil {
		return nil, err
	}
	c.mu.RLock()
	selfID := ""
	if c.selfUser != nil {
		selfID = c.selfUser.ID
	}
	c.mu.RUnlock()
	out := make([]Message, 0, len(raw))
	for _, r := range raw {
		m := convertMessage(r)
		if selfID != "" && m.Author.ID == selfID {
			m.IsFromMe = true
		}
		out = append(out, m)
		_ = c.upsertMessage(ctx, m)
	}
	return out, nil
}

// GetMessage fetches a single message by id.
func (c *Client) GetMessage(ctx context.Context, channelID, messageID string) (Message, error) {
	if channelID == "" || messageID == "" {
		return Message{}, fmt.Errorf("%w: channelId and messageId required", ErrInvalidParams)
	}
	var r rawMessage
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet,
			"/api/v9/channels/"+url.PathEscape(channelID)+"/messages/"+url.PathEscape(messageID),
			nil, &r, nil)
	})
	if err != nil {
		return Message{}, err
	}
	m := convertMessage(r)
	c.mu.RLock()
	selfID := ""
	if c.selfUser != nil {
		selfID = c.selfUser.ID
	}
	c.mu.RUnlock()
	if selfID != "" && m.Author.ID == selfID {
		m.IsFromMe = true
	}
	_ = c.upsertMessage(ctx, m)
	return m, nil
}

func (c *Client) getMessagesLocal(ctx context.Context, channelID string, limit int, beforeID string) ([]Message, error) {
	c.mu.RLock()
	db := c.logDB
	c.mu.RUnlock()
	if db == nil {
		return nil, ErrNotConnected
	}
	q := `SELECT rowid, msg_id, channel_id, COALESCE(guild_id,''),
              COALESCE(author_id,''), COALESCE(author_name,''), is_from_me,
              COALESCE(body,''), has_media, COALESCE(reply_to_id,''),
              timestamp, COALESCE(edited_at, 0)
         FROM messages WHERE channel_id = ?`
	args := []any{channelID}
	if beforeID != "" {
		// Numeric compare on snowflakes (lex compare breaks across lengths).
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

// rawMessage is the shape Discord returns.
type rawMessage struct {
	ID               string     `json:"id"`
	ChannelID        string     `json:"channel_id"`
	GuildID          string     `json:"guild_id,omitempty"`
	Author           User       `json:"author"`
	Content          string     `json:"content"`
	Timestamp        time.Time  `json:"timestamp"`
	EditedTimestamp  *time.Time `json:"edited_timestamp"`
	Attachments      []struct {
		URL string `json:"url"`
	} `json:"attachments"`
	Mentions         []User     `json:"mentions"`
	Pinned           bool       `json:"pinned"`
	MessageReference *struct {
		MessageID string `json:"message_id"`
		ChannelID string `json:"channel_id"`
		GuildID   string `json:"guild_id,omitempty"`
	} `json:"message_reference,omitempty"`
	Reactions []struct {
		Count int  `json:"count"`
		Me    bool `json:"me"`
		Emoji struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"emoji"`
	} `json:"reactions,omitempty"`
}

func convertMessage(r rawMessage) Message {
	m := Message{
		ID:        r.ID,
		ChannelID: r.ChannelID,
		GuildID:   r.GuildID,
		Author:    r.Author,
		Body:      r.Content,
		Pinned:    r.Pinned,
		HasMedia:  len(r.Attachments) > 0,
		Timestamp: r.Timestamp,
	}
	if r.EditedTimestamp != nil {
		m.EditedAt = *r.EditedTimestamp
	}
	for _, a := range r.Attachments {
		m.Attachments = append(m.Attachments, a.URL)
	}
	for _, u := range r.Mentions {
		m.Mentions = append(m.Mentions, u.ID)
	}
	if r.MessageReference != nil {
		m.ReplyToID = r.MessageReference.MessageID
	}
	for _, rx := range r.Reactions {
		key := rx.Emoji.Name
		if rx.Emoji.ID != "" {
			key = rx.Emoji.Name + ":" + rx.Emoji.ID
		}
		m.Reactions = append(m.Reactions, Reaction{Emoji: key, Count: rx.Count, Me: rx.Me})
	}
	return m
}
