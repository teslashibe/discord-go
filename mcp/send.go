package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	discord "github.com/teslashibe/discord-go"
)

type SendMessageInput struct {
	ChannelID string `json:"channel_id"`
	Body      string `json:"body" jsonschema:"description=plain-text message body"`
	ReplyToID string `json:"reply_to_id,omitempty" jsonschema:"description=optional message id to reply to"`
	Silent    bool   `json:"silent,omitempty" jsonschema:"description=suppress @mention notifications"`
	TTS       bool   `json:"tts,omitempty"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func sendMessage(ctx context.Context, c *discord.Client, in SendMessageInput) (any, error) {
	return nil, c.SendMessage(ctx, discord.SendParams{
		ChannelID: in.ChannelID, Body: in.Body, ReplyToID: in.ReplyToID,
		Silent: in.Silent, TTS: in.TTS, Confirm: in.Confirm,
	})
}

type EditMessageInput struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
	Body      string `json:"body"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func editMessage(ctx context.Context, c *discord.Client, in EditMessageInput) (any, error) {
	return nil, c.EditMessage(ctx, discord.EditParams{
		ChannelID: in.ChannelID, MessageID: in.MessageID, Body: in.Body, Confirm: in.Confirm,
	})
}

type DeleteMessageInput struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func deleteMessage(ctx context.Context, c *discord.Client, in DeleteMessageInput) (any, error) {
	return nil, c.DeleteMessage(ctx, discord.DeleteParams{
		ChannelID: in.ChannelID, MessageID: in.MessageID, Confirm: in.Confirm,
	})
}

type BulkDeleteInput struct {
	ChannelID  string   `json:"channel_id"`
	MessageIDs []string `json:"message_ids" jsonschema:"description=2..100 message ids; must be <14 days old (Discord limitation)"`
	Confirm    bool     `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func bulkDelete(ctx context.Context, c *discord.Client, in BulkDeleteInput) (any, error) {
	return nil, c.BulkDeleteMessages(ctx, discord.BulkDeleteParams{
		ChannelID: in.ChannelID, MessageIDs: in.MessageIDs, Confirm: in.Confirm,
	})
}

type SendDMInput struct {
	UserID  string `json:"user_id"`
	Body    string `json:"body"`
	Confirm bool   `json:"confirm" jsonschema:"description=must be true (write operation)"`
}

func sendDM(ctx context.Context, c *discord.Client, in SendDMInput) (any, error) {
	return nil, c.SendDM(ctx, discord.SendDMParams{
		UserID: in.UserID, Body: in.Body, Confirm: in.Confirm,
	})
}

var sendTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, SendMessageInput](
		"discord_send_message",
		"Post a text message into a channel (supports reply_to_id\\, silent\\, tts); confirm-gated and allowlist-checked",
		"SendMessage",
		sendMessage,
	),
	mcptool.Define[*discord.Client, EditMessageInput](
		"discord_edit_message",
		"Edit a message previously sent by the bot (Discord forbids editing other users' messages); confirm-gated",
		"EditMessage",
		editMessage,
	),
	mcptool.Define[*discord.Client, DeleteMessageInput](
		"discord_delete_message",
		"Delete a single message; needs ManageMessages perm to delete others' messages",
		"DeleteMessage",
		deleteMessage,
	),
	mcptool.Define[*discord.Client, BulkDeleteInput](
		"discord_bulk_delete_messages",
		"Bulk-delete 2..100 messages in a channel (Discord rejects items >14 days old)",
		"BulkDeleteMessages",
		bulkDelete,
	),
	mcptool.Define[*discord.Client, SendDMInput](
		"discord_send_dm",
		"Open a DM channel to a user and post a message (needs shared guild + DMs enabled)",
		"SendDM",
		sendDM,
	),
}
