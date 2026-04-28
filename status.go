package discord

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
)

// Status returns a snapshot of the client's auth/connection state.
// Returns (StatusReport{...}, nil) even when not authorised — callers
// inspect the booleans to decide what to do.
func (c *Client) Status(ctx context.Context) (StatusReport, error) {
	rep := StatusReport{
		StoreDir: c.storeDir,
		Intents:  int(c.intents),
	}
	c.mu.RLock()
	rep.Connected = c.connected
	self := c.selfUser
	sess := c.session
	c.mu.RUnlock()

	if self != nil {
		rep.Authorized = true
		rep.BotID = self.ID
		rep.BotUsername = self.Username
	}
	if sess != nil {
		// State is populated by the gateway READY event.
		if sess.State != nil {
			rep.GuildCount = len(sess.State.Guilds)
		}
	}
	rep.StoredMessages = c.storedMessageCount(ctx)

	if !rep.Connected {
		rep.HelpAuth = "set Config.Token (bot token from https://discord.com/developers/applications) and call Connect"
	} else if !rep.Authorized {
		// retry self lookup in case Connect's first call was transient
		_ = c.withSession(func(s *discordgo.Session) error {
			if u, err := s.User("@me"); err == nil {
				c.mu.Lock()
				c.selfUser = u
				c.mu.Unlock()
				rep.Authorized = true
				rep.BotID = u.ID
				rep.BotUsername = u.Username
			}
			return nil
		})
	}
	// Make StoreDir absolute for downstream tooling that may chdir.
	if !filepath.IsAbs(rep.StoreDir) {
		if abs, err := filepath.Abs(rep.StoreDir); err == nil {
			rep.StoreDir = abs
		}
	}
	_ = os.Stat // keep os import for parity with sibling packages
	return rep, nil
}
