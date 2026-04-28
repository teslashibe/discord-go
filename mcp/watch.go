package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	discord "github.com/teslashibe/discord-go"
)

type WatchInput struct {
	SinceRowID int64  `json:"since_row_id,omitempty" jsonschema:"description=cursor returned by previous Watch call"`
	ChannelID  string `json:"channel_id,omitempty" jsonschema:"description=optional channel filter"`
	GuildID    string `json:"guild_id,omitempty" jsonschema:"description=optional guild filter"`
	Limit      int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=500"`
}

func watch(ctx context.Context, c *discord.Client, in WatchInput) (any, error) {
	return c.Watch(ctx, discord.WatchParams{
		SinceRowID: in.SinceRowID, ChannelID: in.ChannelID,
		GuildID: in.GuildID, Limit: in.Limit,
	})
}

var watchTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, WatchInput](
		"discord_watch",
		"Poll the local message log for new gateway events; returns a cursor for the next call (never blocks)",
		"Watch",
		watch,
	),
}
