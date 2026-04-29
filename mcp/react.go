package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"

	discord "github.com/teslashibe/discord-go"
)

type ReactInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=channel snowflake id"`
	MessageID string `json:"message_id" jsonschema:"description=message snowflake id"`
	Emoji     string `json:"emoji" jsonschema:"description=unicode emoji (e.g. 👍) or 'name:id' for a custom guild emoji"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func react(ctx context.Context, c *discord.Client, in ReactInput) (any, error) {
	return nil, c.React(ctx, discord.ReactParams{
		ChannelID: in.ChannelID, MessageID: in.MessageID, Emoji: in.Emoji, Confirm: in.Confirm,
	})
}

type UnreactInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=channel snowflake id"`
	MessageID string `json:"message_id" jsonschema:"description=message snowflake id"`
	Emoji     string `json:"emoji" jsonschema:"description=unicode emoji or 'name:id'"`
	UserID    string `json:"user_id,omitempty" jsonschema:"description=optional: remove a specific user's reaction (default: yours). Removing other users' reactions requires Manage Messages."`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func unreact(ctx context.Context, c *discord.Client, in UnreactInput) (any, error) {
	return nil, c.Unreact(ctx, discord.UnreactParams{
		ChannelID: in.ChannelID, MessageID: in.MessageID, Emoji: in.Emoji, UserID: in.UserID, Confirm: in.Confirm,
	})
}

var reactTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, ReactInput](
		"discord_react",
		"Add an emoji reaction to a message (confirm-gated + channel-allowlist-checked)",
		"React",
		react,
	),
	mcptool.Define[*discord.Client, UnreactInput](
		"discord_unreact",
		"Remove an emoji reaction (default: your own)",
		"Unreact",
		unreact,
	),
}
