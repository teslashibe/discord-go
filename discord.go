// Package discord provides a Go client + MCP tool surface for Discord
// (bot only — selfbots violate Discord's ToS), built on
// github.com/bwmarrin/discordgo.
//
// All write-style methods (SendMessage, EditMessage, DeleteMessage,
// BulkDeleteMessages, React, Unreact, SendDM) honour two cross-cutting
// safety knobs:
//
//   - [WithRequireConfirm] (default true) requires an explicit Confirm
//     flag from the caller.
//   - [WithAllowedChannels] / [WithAllowedGuilds] deny writes to peers
//     outside an allowlist.
package discord

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Intent re-exports discordgo.Intent so callers don't need to import
// discordgo directly.
type Intent = discordgo.Intent

const (
	IntentsGuilds                Intent = discordgo.IntentsGuilds
	IntentsGuildMembers          Intent = discordgo.IntentsGuildMembers
	IntentsGuildMessages         Intent = discordgo.IntentsGuildMessages
	IntentsGuildMessageReactions Intent = discordgo.IntentsGuildMessageReactions
	IntentsDirectMessages        Intent = discordgo.IntentsDirectMessages
	IntentsMessageContent        Intent = discordgo.IntentsMessageContent
)

// Config configures a Client.
type Config struct {
	// Token is the bot token from
	// https://discord.com/developers/applications. Required.
	Token string

	// Intents is a bitfield of the gateway intents your bot needs.
	// Common defaults are bundled in [DefaultIntents].
	Intents Intent

	// StoreDir holds the local messages.db cache. Defaults to a
	// per-user data dir.
	StoreDir string
}

// DefaultIntents covers the read+write surface this package exposes.
// Note that IntentsMessageContent and IntentsGuildMembers are
// "privileged" intents that must be enabled in the developer portal.
const DefaultIntents Intent = IntentsGuilds |
	IntentsGuildMessages |
	IntentsMessageContent |
	IntentsGuildMessageReactions |
	IntentsDirectMessages

// Client is a Discord bot client. Methods are safe for concurrent use
// after Connect returns. Call Close to release resources.
type Client struct {
	token   string
	intents Intent

	storeDir string

	confirmWrites   bool
	dryRun          bool
	allowedChannels map[string]struct{}
	allowedGuilds   map[string]struct{}

	logger *zap.Logger

	mu        sync.RWMutex
	session   *discordgo.Session
	connected bool
	closed    bool
	logDB     *sql.DB
	selfUser  *discordgo.User
}

// New constructs a Client. Connect must be called before any read or
// write tool.
func New(cfg Config, opts ...Option) *Client {
	storeDir := cfg.StoreDir
	if storeDir == "" {
		storeDir = defaultStoreDir()
	}
	intents := cfg.Intents
	if intents == 0 {
		intents = DefaultIntents
	}
	c := &Client{
		token:         cfg.Token,
		intents:       intents,
		storeDir:      storeDir,
		confirmWrites: true,
		logger:        zap.NewNop(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Option configures a Client.
type Option func(*Client)

// WithRequireConfirm controls whether write methods require an explicit
// confirm flag. Default: true.
func WithRequireConfirm(require bool) Option {
	return func(c *Client) { c.confirmWrites = require }
}

// WithDryRun makes write methods log + return without hitting Discord.
func WithDryRun(dry bool) Option {
	return func(c *Client) { c.dryRun = dry }
}

// WithAllowedChannels restricts write methods to a fixed set of channel
// IDs (Discord snowflakes). Pass nil/empty to disable the allowlist.
func WithAllowedChannels(ids []string) Option {
	return func(c *Client) {
		c.allowedChannels = idSet(ids)
	}
}

// WithAllowedGuilds restricts write methods to a fixed set of guild IDs.
func WithAllowedGuilds(ids []string) Option {
	return func(c *Client) {
		c.allowedGuilds = idSet(ids)
	}
}

// WithLogger installs a zap logger; default is no-op.
func WithLogger(l *zap.Logger) Option {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// Close disconnects (if connected) and releases resources. Safe to call
// multiple times.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	sess := c.session
	logDB := c.logDB
	c.session = nil
	c.logDB = nil
	c.connected = false
	c.mu.Unlock()

	var firstErr error
	if sess != nil {
		if err := sess.Close(); err != nil {
			firstErr = err
		}
	}
	if logDB != nil {
		if err := logDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// channelAllowed enforces WithAllowedChannels. Returns nil when no
// allowlist is configured.
func (c *Client) channelAllowed(id string) error {
	if len(c.allowedChannels) == 0 {
		return nil
	}
	if _, ok := c.allowedChannels[id]; ok {
		return nil
	}
	return ErrChannelNotAllowed
}

// guildAllowed enforces WithAllowedGuilds.
func (c *Client) guildAllowed(id string) error {
	if id == "" || len(c.allowedGuilds) == 0 {
		return nil
	}
	if _, ok := c.allowedGuilds[id]; ok {
		return nil
	}
	return ErrGuildNotAllowed
}

// requireConfirm returns ErrConfirmRequired when the host enforces
// confirm-on-write and the caller didn't pass confirm=true.
func (c *Client) requireConfirm(confirm bool) error {
	if !c.confirmWrites || confirm {
		return nil
	}
	return ErrConfirmRequired
}

// withSession runs fn with the active *discordgo.Session under a read
// lock so callers don't need to hold the mutex across the API call.
func (c *Client) withSession(fn func(s *discordgo.Session) error) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClosed
	}
	sess := c.session
	connected := c.connected
	c.mu.RUnlock()
	if sess == nil || !connected {
		return ErrNotConnected
	}
	return fn(sess)
}

func (c *Client) validateConfig() error {
	if c.token == "" {
		return fmt.Errorf("%w: Config.Token is required (bot token from https://discord.com/developers/applications)", ErrInvalidParams)
	}
	return nil
}

// idSet builds a set of non-empty IDs from a slice.
func idSet(ids []string) map[string]struct{} {
	if len(ids) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		m[id] = struct{}{}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func defaultStoreDir() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "teslashibe", "discord-go")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".discord-go")
	}
	return filepath.Join(home, "Library", "Application Support", "teslashibe", "discord-go")
}

// retained for symmetry with sibling packages; lifecycle helper.
var _ = time.Now
