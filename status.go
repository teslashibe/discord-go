package discord

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/teslashibe/discord-go/internal/transport"
)

// Status returns a snapshot of the client's auth/connection state.
// Returns (StatusReport{...}, nil) even when not authorised — callers
// inspect the booleans to decide what to do.
func (c *Client) Status(ctx context.Context) (StatusReport, error) {
	rep := StatusReport{StoreDir: c.storeDir}
	if !filepath.IsAbs(rep.StoreDir) {
		if abs, err := filepath.Abs(rep.StoreDir); err == nil {
			rep.StoreDir = abs
		}
	}
	c.mu.RLock()
	rep.Connected = c.connected
	self := c.selfUser
	c.mu.RUnlock()

	if self != nil {
		rep.Authorized = true
		rep.SelfID = self.ID
		rep.SelfUsername = self.Username
		rep.SelfGlobalName = self.GlobalName
	}
	rep.StoredMessages = c.storedMessageCount(ctx)

	if !rep.Connected {
		rep.HelpAuth = "set Config.Token (extract from web client devtools, see README) and call Connect"
		return rep, nil
	}
	if !rep.Authorized {
		// Re-probe — Connect's first call may have been transient.
		_ = c.withDoer(func(d *transport.Doer) error {
			var u User
			if err := d.JSON(ctx, http.MethodGet, "/api/v9/users/@me", nil, &u, nil); err == nil {
				c.mu.Lock()
				c.selfUser = &u
				c.mu.Unlock()
				rep.Authorized = true
				rep.SelfID = u.ID
				rep.SelfUsername = u.Username
				rep.SelfGlobalName = u.GlobalName
			}
			return nil
		})
	}
	return rep, nil
}
