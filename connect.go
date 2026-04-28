package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Connect opens the gateway websocket and verifies the bot token.
// Subsequent calls before Close return ErrAlreadyConnected.
func (c *Client) Connect(ctx context.Context) error {
	if err := c.validateConfig(); err != nil {
		return err
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClosed
	}
	if c.connected {
		c.mu.Unlock()
		return ErrAlreadyConnected
	}
	c.mu.Unlock()

	if err := c.initLogStore(ctx); err != nil {
		return err
	}

	sess, err := discordgo.New("Bot " + c.token)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}
	sess.Identify.Intents = c.intents
	sess.ShouldReconnectOnError = true
	sess.StateEnabled = true

	c.installEventHandlers(sess)

	// Audit fix (H2): publish the session to c BEFORE Open() so that a
	// concurrent Close() can tear it down even if Open's goroutines
	// have already started consuming events. We mark connected=false
	// during the open window; methods that need the session block on
	// the connected flag.
	c.mu.Lock()
	c.session = sess
	c.connected = false
	c.mu.Unlock()

	if err := sess.Open(); err != nil {
		c.mu.Lock()
		c.session = nil
		c.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}
	// Verify the token by hitting /users/@me. This catches revoked /
	// invalid tokens before the first user-visible call.
	self, err := sess.User("@me")
	if err != nil {
		_ = sess.Close()
		c.mu.Lock()
		c.session = nil
		c.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}

	c.mu.Lock()
	c.connected = true
	c.selfUser = self
	c.mu.Unlock()
	return nil
}

// Disconnect closes the gateway websocket but keeps logDB / config so
// the client can be Connect-ed again.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	sess := c.session
	c.session = nil
	c.connected = false
	c.mu.Unlock()
	if sess == nil {
		return nil
	}
	return sess.Close()
}
