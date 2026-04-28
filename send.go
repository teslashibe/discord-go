package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SendMessage posts a text message into a channel. Confirm-gated and
// allowlist-checked.
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
	return c.withSession(func(s *discordgo.Session) error {
		send := &discordgo.MessageSend{Content: params.Body, TTS: params.TTS}
		if params.ReplyToID != "" {
			send.Reference = &discordgo.MessageReference{
				MessageID: params.ReplyToID,
				ChannelID: params.ChannelID,
			}
		}
		if params.Silent {
			send.Flags = discordgo.MessageFlagsSuppressNotifications
		}
		_, err := s.ChannelMessageSendComplex(params.ChannelID, send)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return nil
	})
}

// EditMessage replaces the body of a message previously sent by the bot.
// Discord does not allow editing other users' messages.
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
	return c.withSession(func(s *discordgo.Session) error {
		_, err := s.ChannelMessageEdit(params.ChannelID, params.MessageID, params.Body)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return nil
	})
}

// DeleteMessage deletes a single message. Bots need the ManageMessages
// permission to delete other users' messages.
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
	return c.withSession(func(s *discordgo.Session) error {
		if err := s.ChannelMessageDelete(params.ChannelID, params.MessageID); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		c.deleteMessageRow(ctx, params.ChannelID, params.MessageID)
		return nil
	})
}

// BulkDeleteMessages deletes 2–100 messages in a single call. Discord
// rejects messages older than 14 days; we let the API surface that.
func (c *Client) BulkDeleteMessages(ctx context.Context, params BulkDeleteParams) error {
	if params.ChannelID == "" {
		return fmt.Errorf("%w: ChannelID required", ErrInvalidParams)
	}
	if len(params.MessageIDs) < 2 || len(params.MessageIDs) > 100 {
		return fmt.Errorf("%w: BulkDelete requires 2..100 message ids (got %d)", ErrInvalidParams, len(params.MessageIDs))
	}
	if err := c.requireConfirm(params.Confirm); err != nil {
		return err
	}
	if err := c.channelAllowed(params.ChannelID); err != nil {
		return err
	}
	if c.dryRun {
		c.logger.Info("dry-run: would BulkDeleteMessages")
		return nil
	}
	return c.withSession(func(s *discordgo.Session) error {
		if err := s.ChannelMessagesBulkDelete(params.ChannelID, params.MessageIDs); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		for _, id := range params.MessageIDs {
			c.deleteMessageRow(ctx, params.ChannelID, id)
		}
		return nil
	})
}

// React adds an emoji reaction. Emoji is the unicode emoji ("👍") or
// "name:id" for a custom guild emoji.
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
	return c.withSession(func(s *discordgo.Session) error {
		if err := s.MessageReactionAdd(params.ChannelID, params.MessageID, params.Emoji); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return nil
	})
}

// Unreact removes a reaction. UserID="" removes the bot's own reaction;
// any other id removes that user's reaction (requires ManageMessages).
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
	return c.withSession(func(s *discordgo.Session) error {
		if err := s.MessageReactionRemove(params.ChannelID, params.MessageID, params.Emoji, uid); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return nil
	})
}

// SendDM opens a DM channel to a user and posts a message. Requires
// the user to share a guild with the bot AND have DMs from server
// members enabled.
func (c *Client) SendDM(ctx context.Context, params SendDMParams) error {
	if params.UserID == "" {
		return fmt.Errorf("%w: UserID required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.Body) == "" {
		return ErrMessageEmpty
	}
	if err := c.requireConfirm(params.Confirm); err != nil {
		return err
	}
	if c.dryRun {
		c.logger.Info("dry-run: would SendDM")
		return nil
	}
	return c.withSession(func(s *discordgo.Session) error {
		ch, err := s.UserChannelCreate(params.UserID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDMNotAllowed, err)
		}
		if _, err := s.ChannelMessageSend(ch.ID, params.Body); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return nil
	})
}
