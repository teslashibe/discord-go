package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	discord "github.com/teslashibe/discord-go"
)

type StatusInput struct{}

func status(ctx context.Context, c *discord.Client, _ StatusInput) (any, error) {
	return c.Status(ctx)
}

var statusTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, StatusInput](
		"discord_status",
		"Report client state: store path, connection + auth status, bot identity, guild count, intents",
		"Status",
		status,
	),
}
