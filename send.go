package discord

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/teslashibe/discord-go/internal/transport"
)

// SendMessage posts a text message into a channel. Confirm-gated +
// channel-allowlist-checked.
func (c *Client) SendMessage(ctx context.Context, params SendParams) error {
	if params.ChannelID == "" {
		return fmt.Errorf("%w: ChannelID required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.Body) == "" {
		return ErrMessageEmpty
	}
	if err := c.requireConfirm(params.Confirm); err != nil {
		return err
	}
	if err := c.channelAllowed(params.ChannelID); err != nil {
		return err
	}
	if c.dryRun {
		c.logger.Info("dry-run: would SendMessage")
		return nil
	}
	body := map[string]any{"content": params.Body}
	if params.ReplyToID != "" {
		body["message_reference"] = map[string]any{
			"message_id": params.ReplyToID,
			"channel_id": params.ChannelID,
		}
	}
	if params.Silent {
		body["flags"] = 4096 // SUPPRESS_NOTIFICATIONS
	}
	return c.withDoer(func(d *transport.Doer) error {
		var out rawMessage
		if err := d.JSON(ctx, http.MethodPost,
			"/api/v9/channels/"+url.PathEscape(params.ChannelID)+"/messages",
			body, &out, nil); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		_ = c.upsertMessage(ctx, convertMessage(out))
		return nil
	})
}

// EditMessage replaces the body of one of YOUR messages. Discord
// rejects edits to other users' messages with 403.
func (c *Client) EditMessage(ctx context.Context, params EditParams) error {
	if params.ChannelID == "" || params.MessageID == "" {
		return fmt.Errorf("%w: ChannelID and MessageID required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.Body) == "" {
		return ErrMessageEmpty
	}
	if err := c.requireConfirm(params.Confirm); err != nil {
		return err
	}
	if err := c.channelAllowed(params.ChannelID); err != nil {
		return err
	}
	if c.dryRun {
		c.logger.Info("dry-run: would EditMessage")
		return nil
	}
	body := map[string]any{"content": params.Body}
	return c.withDoer(func(d *transport.Doer) error {
		var out rawMessage
		if err := d.JSON(ctx, http.MethodPatch,
			"/api/v9/channels/"+url.PathEscape(params.ChannelID)+"/messages/"+url.PathEscape(params.MessageID),
			body, &out, nil); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		_ = c.upsertMessage(ctx, convertMessage(out))
		return nil
	})
}

// DeleteMessage deletes a single message. You can always delete your
// own messages; deleting others requires Manage Messages in the guild.
func (c *Client) DeleteMessage(ctx context.Context, params DeleteParams) error {
	if params.ChannelID == "" || params.MessageID == "" {
		return fmt.Errorf("%w: ChannelID and MessageID required", ErrInvalidParams)
	}
	if err := c.requireConfirm(params.Confirm); err != nil {
		return err
	}
	if err := c.channelAllowed(params.ChannelID); err != nil {
		return err
	}
	if c.dryRun {
		c.logger.Info("dry-run: would DeleteMessage")
		return nil
	}
	return c.withDoer(func(d *transport.Doer) error {
		if err := d.JSON(ctx, http.MethodDelete,
			"/api/v9/channels/"+url.PathEscape(params.ChannelID)+"/messages/"+url.PathEscape(params.MessageID),
			nil, nil, nil); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		c.deleteMessageRow(ctx, params.ChannelID, params.MessageID)
		return nil
	})
}

// React adds an emoji reaction. Emoji is unicode ("👍") or "name:id"
// for a custom guild emoji (URL-encoded internally).
func (c *Client) React(ctx context.Context, params ReactParams) error {
	if params.ChannelID == "" || params.MessageID == "" || params.Emoji == "" {
		return fmt.Errorf("%w: ChannelID, MessageID, Emoji required", ErrInvalidParams)
	}
	if err := c.requireConfirm(params.Confirm); err != nil {
		return err
	}
	if err := c.channelAllowed(params.ChannelID); err != nil {
		return err
	}
	if c.dryRun {
		c.logger.Info("dry-run: would React")
		return nil
	}
	emoji := url.PathEscape(params.Emoji)
	path := "/api/v9/channels/" + url.PathEscape(params.ChannelID) +
		"/messages/" + url.PathEscape(params.MessageID) +
		"/reactions/" + emoji + "/@me"
	return c.withDoer(func(d *transport.Doer) error {
		if err := d.JSON(ctx, http.MethodPut, path, nil, nil, nil); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return nil
	})
}

// Unreact removes a reaction. UserID="" removes your own reaction.
func (c *Client) Unreact(ctx context.Context, params UnreactParams) error {
	if params.ChannelID == "" || params.MessageID == "" || params.Emoji == "" {
		return fmt.Errorf("%w: ChannelID, MessageID, Emoji required", ErrInvalidParams)
	}
	if err := c.requireConfirm(params.Confirm); err != nil {
		return err
	}
	if err := c.channelAllowed(params.ChannelID); err != nil {
		return err
	}
	if c.dryRun {
		c.logger.Info("dry-run: would Unreact")
		return nil
	}
	uid := params.UserID
	if uid == "" {
		uid = "@me"
	}
	emoji := url.PathEscape(params.Emoji)
	path := "/api/v9/channels/" + url.PathEscape(params.ChannelID) +
		"/messages/" + url.PathEscape(params.MessageID) +
		"/reactions/" + emoji + "/" + url.PathEscape(uid)
	return c.withDoer(func(d *transport.Doer) error {
		if err := d.JSON(ctx, http.MethodDelete, path, nil, nil, nil); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return nil
	})
}
