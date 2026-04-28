package discord

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/teslashibe/discord-go/internal/transport"
)

// ListGuilds returns the guilds you are in.
func (c *Client) ListGuilds(ctx context.Context, params GuildListParams) ([]Guild, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	q := url.Values{}
	q.Set("with_counts", "true")
	var raw []struct {
		ID                       string `json:"id"`
		Name                     string `json:"name"`
		OwnerID                  string `json:"owner_id"`
		Icon                     string `json:"icon"`
		ApproximateMemberCount   int    `json:"approximate_member_count"`
	}
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet, "/api/v9/users/@me/guilds", nil, &raw, q)
	})
	if err != nil {
		return nil, err
	}
	if limit < len(raw) {
		raw = raw[:limit]
	}
	out := make([]Guild, 0, len(raw))
	for _, g := range raw {
		out = append(out, Guild{
			ID: g.ID, Name: g.Name, OwnerID: g.OwnerID,
			IconURL:     guildIconURL(g.ID, g.Icon),
			MemberCount: g.ApproximateMemberCount,
		})
	}
	return out, nil
}

// GetGuild fetches a single guild by id.
func (c *Client) GetGuild(ctx context.Context, guildID string) (Guild, error) {
	if guildID == "" {
		return Guild{}, fmt.Errorf("%w: guildId required", ErrInvalidParams)
	}
	q := url.Values{}
	q.Set("with_counts", "true")
	var raw struct {
		ID                     string `json:"id"`
		Name                   string `json:"name"`
		OwnerID                string `json:"owner_id"`
		Icon                   string `json:"icon"`
		ApproximateMemberCount int    `json:"approximate_member_count"`
	}
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet, "/api/v9/guilds/"+url.PathEscape(guildID), nil, &raw, q)
	})
	if err != nil {
		return Guild{}, err
	}
	return Guild{
		ID: raw.ID, Name: raw.Name, OwnerID: raw.OwnerID,
		IconURL:     guildIconURL(raw.ID, raw.Icon),
		MemberCount: raw.ApproximateMemberCount,
	}, nil
}

// ListChannels returns all channels in a guild.
func (c *Client) ListChannels(ctx context.Context, params ChannelListParams) ([]Channel, error) {
	if params.GuildID == "" {
		return nil, fmt.Errorf("%w: GuildID required", ErrInvalidParams)
	}
	var raw []rawChannel
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet,
			"/api/v9/guilds/"+url.PathEscape(params.GuildID)+"/channels",
			nil, &raw, nil)
	})
	if err != nil {
		return nil, err
	}
	out := make([]Channel, 0, len(raw))
	for _, r := range raw {
		ch := convertChannel(r)
		out = append(out, ch)
		c.upsertChannelMeta(ctx, ch)
	}
	return out, nil
}

// GetChannel fetches a single channel by id.
func (c *Client) GetChannel(ctx context.Context, channelID string) (Channel, error) {
	if channelID == "" {
		return Channel{}, fmt.Errorf("%w: channelId required", ErrInvalidParams)
	}
	var r rawChannel
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet,
			"/api/v9/channels/"+url.PathEscape(channelID),
			nil, &r, nil)
	})
	if err != nil {
		return Channel{}, err
	}
	ch := convertChannel(r)
	c.upsertChannelMeta(ctx, ch)
	return ch, nil
}

// ListDMChannels returns your DM and group-DM channels (ordered most-
// recent activity first by Discord).
func (c *Client) ListDMChannels(ctx context.Context, params DMListParams) ([]Channel, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	var raw []rawChannel
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet, "/api/v9/users/@me/channels", nil, &raw, nil)
	})
	if err != nil {
		return nil, err
	}
	if limit < len(raw) {
		raw = raw[:limit]
	}
	out := make([]Channel, 0, len(raw))
	for _, r := range raw {
		ch := convertChannel(r)
		// For 1:1 DMs Discord doesn't set Name; derive from recipient.
		if ch.Name == "" && len(ch.Recipients) > 0 {
			ch.Name = displayNameOf(ch.Recipients[0])
		}
		out = append(out, ch)
		c.upsertChannelMeta(ctx, ch)
	}
	return out, nil
}

// rawChannel is the shape Discord returns for /channels endpoints.
type rawChannel struct {
	ID            string `json:"id"`
	Type          int    `json:"type"`
	GuildID       string `json:"guild_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Topic         string `json:"topic,omitempty"`
	NSFW          bool   `json:"nsfw,omitempty"`
	ParentID      string `json:"parent_id,omitempty"`
	Position      int    `json:"position,omitempty"`
	LastMessageID string `json:"last_message_id,omitempty"`
	Recipients    []User `json:"recipients,omitempty"`
}

func convertChannel(r rawChannel) Channel {
	return Channel{
		ID:            r.ID,
		GuildID:       r.GuildID,
		Name:          r.Name,
		Type:          channelTypeName(r.Type),
		Topic:         r.Topic,
		NSFW:          r.NSFW,
		ParentID:      r.ParentID,
		Position:      r.Position,
		LastMessageID: r.LastMessageID,
		Recipients:    r.Recipients,
	}
}

// channelTypeName maps Discord's int type ids to readable strings.
func channelTypeName(t int) string {
	switch t {
	case 0:
		return "text"
	case 1:
		return "dm"
	case 2:
		return "voice"
	case 3:
		return "group_dm"
	case 4:
		return "category"
	case 5:
		return "news"
	case 10:
		return "news_thread"
	case 11:
		return "public_thread"
	case 12:
		return "private_thread"
	case 13:
		return "stage_voice"
	case 15:
		return "forum"
	case 16:
		return "media"
	}
	return "type_" + strconv.Itoa(t)
}

func guildIconURL(guildID, icon string) string {
	if icon == "" || guildID == "" {
		return ""
	}
	ext := "png"
	if len(icon) > 2 && icon[:2] == "a_" {
		ext = "gif"
	}
	return "https://cdn.discordapp.com/icons/" + guildID + "/" + icon + "." + ext
}
