package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"

	discord "github.com/teslashibe/discord-go"
)

type SendMessageInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=channel snowflake id"`
	Body      string `json:"body" jsonschema:"description=message text (markdown supported)"`
	ReplyToID string `json:"reply_to_id,omitempty" jsonschema:"description=optional message snowflake to reply to"`
	Silent    bool   `json:"silent,omitempty" jsonschema:"description=true = use SUPPRESS_NOTIFICATIONS flag"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func sendMessage(ctx context.Context, c *discord.Client, in SendMessageInput) (any, error) {
	return nil, c.SendMessage(ctx, discord.SendParams{
		ChannelID: in.ChannelID, Body: in.Body, ReplyToID: in.ReplyToID,
		Silent: in.Silent, Confirm: in.Confirm,
	})
}

type EditMessageInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=channel snowflake id"`
	MessageID string `json:"message_id" jsonschema:"description=message snowflake id (must be authored by you)"`
	Body      string `json:"body" jsonschema:"description=replacement text"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func editMessage(ctx context.Context, c *discord.Client, in EditMessageInput) (any, error) {
	return nil, c.EditMessage(ctx, discord.EditParams{
		ChannelID: in.ChannelID, MessageID: in.MessageID, Body: in.Body, Confirm: in.Confirm,
	})
}

type DeleteMessageInput struct {
	ChannelID string `json:"channel_id" jsonschema:"description=channel snowflake id"`
	MessageID string `json:"message_id" jsonschema:"description=message snowflake id"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func deleteMessage(ctx context.Context, c *discord.Client, in DeleteMessageInput) (any, error) {
	return nil, c.DeleteMessage(ctx, discord.DeleteParams{
		ChannelID: in.ChannelID, MessageID: in.MessageID, Confirm: in.Confirm,
	})
}

var sendTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, SendMessageInput](
		"discord_send_message",
		"Post a message to a channel (confirm-gated + channel-allowlist-checked)",
		"SendMessage",
		sendMessage,
	),
	mcptool.Define[*discord.Client, EditMessageInput](
		"discord_edit_message",
		"Edit one of your own messages (confirm-gated + channel-allowlist-checked)",
		"EditMessage",
		editMessage,
	),
	mcptool.Define[*discord.Client, DeleteMessageInput](
		"discord_delete_message",
		"Delete a message (confirm-gated + channel-allowlist-checked)",
		"DeleteMessage",
		deleteMessage,
	),
}
