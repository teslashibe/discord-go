package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	discord "github.com/teslashibe/discord-go"
)

type ListChannelsInput struct {
	GuildID string `json:"guild_id" jsonschema:"description=Discord guild snowflake id"`
}

func listChannels(ctx context.Context, c *discord.Client, in ListChannelsInput) (any, error) {
	return c.ListChannels(ctx, discord.ChannelListParams{GuildID: in.GuildID})
}

type GetChannelInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=Discord channel snowflake id"`
}

func getChannel(ctx context.Context, c *discord.Client, in GetChannelInput) (any, error) {
	return c.GetChannel(ctx, in.ChannelID)
}

var channelTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, ListChannelsInput](
		"discord_list_channels",
		"List all channels in a guild (text/voice/category/thread/forum)",
		"ListChannels",
		listChannels,
	),
	mcptool.Define[*discord.Client, GetChannelInput](
		"discord_get_channel",
		"Fetch a single channel by id (type, topic, NSFW, parent, position)",
		"GetChannel",
		getChannel,
	),
}
