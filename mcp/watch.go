package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"

	discord "github.com/teslashibe/discord-go"
)

type WatchInput struct {
	SinceRowID   int64    `json:"since_row_id,omitempty" jsonschema:"description=local cursor; use the previous call's cursor (0 = from start)"`
	ChannelID    string   `json:"channel_id,omitempty" jsonschema:"description=optional: only return messages for this channel"`
	GuildID      string   `json:"guild_id,omitempty" jsonschema:"description=optional: only return messages for this guild"`
	Limit        int      `json:"limit,omitempty" jsonschema:"minimum=1,maximum=500"`
	Poll         bool     `json:"poll,omitempty" jsonschema:"description=true = refresh poll_channels via REST before returning"`
	PollChannels []string `json:"poll_channels,omitempty" jsonschema:"description=channels to poll when poll=true (capped at 25 per call)"`
}

func watch(ctx context.Context, c *discord.Client, in WatchInput) (any, error) {
	return c.Watch(ctx, discord.WatchParams{
		SinceRowID:   in.SinceRowID,
		ChannelID:    in.ChannelID,
		GuildID:      in.GuildID,
		Limit:        in.Limit,
		Poll:         in.Poll,
		PollChannels: in.PollChannels,
	})
}

var watchTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, WatchInput](
		"discord_watch",
		"Tail the local message log; with poll=true also refreshes the listed channels first",
		"Watch",
		watch,
	),
}
