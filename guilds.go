package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// ListGuilds returns the guilds the bot is in.
func (c *Client) ListGuilds(ctx context.Context, params GuildListParams) ([]Guild, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	var out []Guild
	err := c.withSession(func(s *discordgo.Session) error {
		gs, err := s.UserGuilds(limit, "", "", false)
		if err != nil {
			return fmt.Errorf("UserGuilds: %w", err)
		}
		for _, g := range gs {
			out = append(out, Guild{
				ID:      g.ID,
				Name:    g.Name,
				IconURL: guildIconURL(g.ID, g.Icon),
			})
		}
		return nil
	})
	return out, err
}

// GetGuild returns a single guild by id, including member count.
func (c *Client) GetGuild(ctx context.Context, guildID string) (Guild, error) {
	if guildID == "" {
		return Guild{}, fmt.Errorf("%w: guildId required", ErrInvalidParams)
	}
	var out Guild
	err := c.withSession(func(s *discordgo.Session) error {
		g, err := s.GuildWithCounts(guildID)
		if err != nil {
			return fmt.Errorf("Guild: %w", err)
		}
		out = Guild{
			ID:          g.ID,
			Name:        g.Name,
			OwnerID:     g.OwnerID,
			IconURL:     g.IconURL(""),
			MemberCount: g.ApproximateMemberCount,
		}
		return nil
	})
	return out, err
}

// ListChannels returns all channels in a guild (text/voice/category/thread).
func (c *Client) ListChannels(ctx context.Context, params ChannelListParams) ([]Channel, error) {
	if params.GuildID == "" {
		return nil, fmt.Errorf("%w: GuildID required", ErrInvalidParams)
	}
	var out []Channel
	err := c.withSession(func(s *discordgo.Session) error {
		chs, err := s.GuildChannels(params.GuildID)
		if err != nil {
			return fmt.Errorf("GuildChannels: %w", err)
		}
		for _, ch := range chs {
			out = append(out, convertChannel(ch))
		}
		return nil
	})
	return out, err
}

// GetChannel returns a single channel by id.
func (c *Client) GetChannel(ctx context.Context, channelID string) (Channel, error) {
	if channelID == "" {
		return Channel{}, fmt.Errorf("%w: channelId required", ErrInvalidParams)
	}
	var out Channel
	err := c.withSession(func(s *discordgo.Session) error {
		ch, err := s.Channel(channelID)
		if err != nil {
			return fmt.Errorf("Channel: %w", err)
		}
		out = convertChannel(ch)
		return nil
	})
	return out, err
}

func convertChannel(ch *discordgo.Channel) Channel {
	if ch == nil {
		return Channel{}
	}
	return Channel{
		ID:       ch.ID,
		GuildID:  ch.GuildID,
		Name:     ch.Name,
		Type:     channelTypeName(ch.Type),
		Topic:    ch.Topic,
		NSFW:     ch.NSFW,
		ParentID: ch.ParentID,
		Position: ch.Position,
	}
}

// guildIconURL builds the standard CDN URL for a guild icon hash.
// Returns empty when the guild has no icon set.
func guildIconURL(guildID, icon string) string {
	if icon == "" || guildID == "" {
		return ""
	}
	ext := "png"
	if len(icon) > 2 && icon[:2] == "a_" {
		ext = "gif"
	}
	return fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.%s", guildID, icon, ext)
}

func channelTypeName(t discordgo.ChannelType) string {
	switch t {
	case discordgo.ChannelTypeGuildText:
		return "text"
	case discordgo.ChannelTypeDM:
		return "dm"
	case discordgo.ChannelTypeGuildVoice:
		return "voice"
	case discordgo.ChannelTypeGroupDM:
		return "group_dm"
	case discordgo.ChannelTypeGuildCategory:
		return "category"
	case discordgo.ChannelTypeGuildNews:
		return "news"
	case discordgo.ChannelTypeGuildStore:
		return "store"
	case discordgo.ChannelTypeGuildNewsThread:
		return "news_thread"
	case discordgo.ChannelTypeGuildPublicThread:
		return "public_thread"
	case discordgo.ChannelTypeGuildPrivateThread:
		return "private_thread"
	case discordgo.ChannelTypeGuildStageVoice:
		return "stage_voice"
	case discordgo.ChannelTypeGuildForum:
		return "forum"
	}
	return fmt.Sprintf("type_%d", int(t))
}
