package discord

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
)

// installEventHandlers registers gateway listeners that mirror live
// MessageCreate / MessageUpdate / MessageDelete events into the local
// SQLite log so Watch can serve them.
func (c *Client) installEventHandlers(sess *discordgo.Session) {
	sess.AddHandler(func(s *discordgo.Session, e *discordgo.MessageCreate) {
		c.recordMessage(context.Background(), e.Message)
	})
	sess.AddHandler(func(s *discordgo.Session, e *discordgo.MessageUpdate) {
		// MessageUpdate may arrive without Author when the embed-only
		// path is taken; recordMessage tolerates a nil author.
		c.recordMessage(context.Background(), e.Message)
	})
	sess.AddHandler(func(s *discordgo.Session, e *discordgo.MessageDelete) {
		c.deleteMessageRow(context.Background(), e.ChannelID, e.ID)
	})
	sess.AddHandler(func(s *discordgo.Session, e *discordgo.MessageDeleteBulk) {
		for _, id := range e.Messages {
			c.deleteMessageRow(context.Background(), e.ChannelID, id)
		}
	})
}

// recordMessage upserts a discordgo.Message into the local log.
func (c *Client) recordMessage(ctx context.Context, dm *discordgo.Message) {
	if dm == nil {
		return
	}
	m := convertMessage(dm)
	c.mu.RLock()
	selfID := ""
	if c.selfUser != nil {
		selfID = c.selfUser.ID
	}
	c.mu.RUnlock()
	if selfID != "" && m.Author.ID == selfID {
		m.IsFromMe = true
	}
	if err := c.upsertMessage(ctx, m); err != nil {
		c.logger.Warn("upsertMessage failed for live event")
	}
}

// convertMessage maps discordgo.Message → our Message.
func convertMessage(dm *discordgo.Message) Message {
	m := Message{
		ID:         dm.ID,
		ChannelID:  dm.ChannelID,
		GuildID:    dm.GuildID,
		Body:       dm.Content,
		Pinned:     dm.Pinned,
		Timestamp:  dm.Timestamp,
		EditedAt:   timeOrZero(dm.EditedTimestamp),
		HasMedia:   len(dm.Attachments) > 0,
	}
	if dm.Author != nil {
		m.Author = User{
			ID:            dm.Author.ID,
			Username:      dm.Author.Username,
			GlobalName:    dm.Author.GlobalName,
			Discriminator: dm.Author.Discriminator,
			Bot:           dm.Author.Bot,
			System:        dm.Author.System,
			AvatarURL:     dm.Author.AvatarURL(""),
		}
	}
	if dm.MessageReference != nil {
		m.ReplyToID = dm.MessageReference.MessageID
	}
	for _, a := range dm.Attachments {
		if a == nil {
			continue
		}
		m.Attachments = append(m.Attachments, a.URL)
	}
	for _, u := range dm.Mentions {
		if u != nil {
			m.Mentions = append(m.Mentions, u.ID)
		}
	}
	for _, r := range dm.Reactions {
		if r == nil || r.Emoji == nil {
			continue
		}
		m.Reactions = append(m.Reactions, Reaction{
			Emoji: emojiKey(r.Emoji),
			Count: r.Count,
			Me:    r.Me,
		})
	}
	return m
}

func emojiKey(e *discordgo.Emoji) string {
	if e == nil {
		return ""
	}
	if e.ID == "" {
		return e.Name
	}
	return e.Name + ":" + e.ID
}

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
