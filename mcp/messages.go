package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	discord "github.com/teslashibe/discord-go"
)

type GetMessagesInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=channel snowflake id"`
	Limit     int    `json:"limit,omitempty" jsonschema:"description=default 50\\, max 100 (Discord cap),minimum=1,maximum=100"`
	BeforeID  string `json:"before_id,omitempty" jsonschema:"description=pagination: messages older than this id"`
	AfterID   string `json:"after_id,omitempty" jsonschema:"description=pagination: messages newer than this id"`
	AroundID  string `json:"around_id,omitempty" jsonschema:"description=pagination: messages around this id"`
	FromLocal bool   `json:"from_local,omitempty" jsonschema:"description=true = serve from local SQLite log only (no network)"`
}

func getMessages(ctx context.Context, c *discord.Client, in GetMessagesInput) (any, error) {
	return c.GetMessages(ctx, discord.MessageListParams{
		ChannelID: in.ChannelID, Limit: in.Limit,
		BeforeID: in.BeforeID, AfterID: in.AfterID, AroundID: in.AroundID,
		FromLocal: in.FromLocal,
	})
}

type GetMessageInput struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
}

func getMessage(ctx context.Context, c *discord.Client, in GetMessageInput) (any, error) {
	return c.GetMessage(ctx, in.ChannelID, in.MessageID)
}

var messageTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, GetMessagesInput](
		"discord_get_messages",
		"Fetch a channel's recent messages (default REST; set from_local=true for cached log only)",
		"GetMessages",
		getMessages,
	),
	mcptool.Define[*discord.Client, GetMessageInput](
		"discord_get_message",
		"Fetch a single message by channel + message id",
		"GetMessage",
		getMessage,
	),
}
