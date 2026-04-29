package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"

	discord "github.com/teslashibe/discord-go"
)

type ListChannelsInput struct {
	GuildID string `json:"guild_id" jsonschema:"description=guild snowflake id"`
}

func listChannels(ctx context.Context, c *discord.Client, in ListChannelsInput) (any, error) {
	return c.ListChannels(ctx, discord.ChannelListParams{GuildID: in.GuildID})
}

type GetChannelInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=channel snowflake id"`
}

func getChannel(ctx context.Context, c *discord.Client, in GetChannelInput) (any, error) {
	return c.GetChannel(ctx, in.ChannelID)
}

type ListDMChannelsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=default 50,minimum=1,maximum=200"`
}

func listDMChannels(ctx context.Context, c *discord.Client, in ListDMChannelsInput) (any, error) {
	return c.ListDMChannels(ctx, discord.DMListParams{Limit: in.Limit})
}

var channelTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, ListChannelsInput](
		"discord_list_channels",
		"List all channels in a guild (text/voice/category/forum/thread/etc.)",
		"ListChannels",
		listChannels,
	),
	mcptool.Define[*discord.Client, GetChannelInput](
		"discord_get_channel",
		"Fetch a single channel by snowflake id (works for guild channels and DMs)",
		"GetChannel",
		getChannel,
	),
	mcptool.Define[*discord.Client, ListDMChannelsInput](
		"discord_list_dm_channels",
		"List your direct-message and group-DM channels (most recent activity first)",
		"ListDMChannels",
		listDMChannels,
	),
}
