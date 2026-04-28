// Package discord provides a Go client + MCP tool surface for Discord
// that authenticates as a real user account using a user token (the
// same one your browser session uses), built on a browser-faithful
// HTTP transport in internal/transport.
//
// READ THE README FIRST. This is a "selfbot" pattern. Discord's ToS
// forbids it and the punishment is account permaban. Use this only on
// your own personal account, with the safety knobs left on.
//
// All write methods (SendMessage, EditMessage, DeleteMessage, React,
// Unreact) honour:
//   - [WithRequireConfirm] (default true) requires Confirm=true.
//   - [WithAllowedChannels] / [WithAllowedGuilds] deny off-list writes.
package discord

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"go.uber.org/zap"

	"github.com/teslashibe/discord-go/internal/transport"
)

// Config configures a Client.
type Config struct {
	// Token is your Discord user-account token. Required. See README
	// for the JS snippet to extract it from the web client.
	Token string

	// Locale is sent as X-Discord-Locale + system_locale in
	// X-Super-Properties. Defaults to "en-US".
	Locale string

	// Timezone is sent as X-Discord-Timezone. Defaults to
	// "America/Los_Angeles".
	Timezone string

	// StoreDir holds the local messages.db. Defaults to a per-user dir.
	StoreDir string

	// MinGap is the floor between consecutive HTTP calls to keep
	// pacing human-like. Defaults to 500ms; bump it higher if you're
	// being defensive.
	MinGap time.Duration

	// BuildNumber overrides the client_build_number baked into
	// X-Super-Properties. The default tracks the Discord web build at
	// the time this package was published; if Discord's web client
	// updates and you want to match it, override here.
	BuildNumber int

	// UserAgent overrides the User-Agent header. Defaults to a recent
	// Chrome-on-macOS string that matches the headers we send.
	UserAgent string
}

// Client is a Discord user-account client. Methods are safe for
// concurrent use after Connect returns. Call Close to release resources.
type Client struct {
	token    string
	storeDir string
	cfg      Config

	confirmWrites   bool
	dryRun          bool
	allowedChannels map[string]struct{}
	allowedGuilds   map[string]struct{}

	logger *zap.Logger

	httpClient *http.Client

	mu        sync.RWMutex
	doer      *transport.Doer
	connected bool
	closed    bool
	logDB     *sql.DB
	selfUser  *User
}

// New constructs a Client. Connect must be called before any read or
// write tool.
func New(cfg Config, opts ...Option) *Client {
	storeDir := cfg.StoreDir
	if storeDir == "" {
		storeDir = defaultStoreDir()
	}
	cfg.StoreDir = storeDir
	c := &Client{
		token:         cfg.Token,
		storeDir:      storeDir,
		cfg:           cfg,
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
// IDs. Pass nil/empty to disable the allowlist.
func WithAllowedChannels(ids []string) Option {
	return func(c *Client) { c.allowedChannels = idSet(ids) }
}

// WithAllowedGuilds restricts write methods to a fixed set of guild IDs.
func WithAllowedGuilds(ids []string) Option {
	return func(c *Client) { c.allowedGuilds = idSet(ids) }
}

// WithMinGap overrides the inter-call floor. Lower = riskier.
func WithMinGap(d time.Duration) Option {
	return func(c *Client) { c.cfg.MinGap = d }
}

// WithBuildNumber overrides the X-Super-Properties build number.
func WithBuildNumber(n int) Option {
	return func(c *Client) { c.cfg.BuildNumber = n }
}

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(s string) Option {
	return func(c *Client) { c.cfg.UserAgent = s }
}

// WithLogger installs a zap logger; default is no-op.
func WithLogger(l *zap.Logger) Option {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithHTTPClient injects a custom http.Client (e.g. for tests).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// Connect verifies the token by hitting /users/@me. Subsequent calls
// before Close return ErrAlreadyConnected.
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

	doer, err := transport.New(transport.Options{
		Token:       c.token,
		UserAgent:   c.cfg.UserAgent,
		Locale:      c.cfg.Locale,
		Timezone:    c.cfg.Timezone,
		BuildNumber: c.cfg.BuildNumber,
		MinGap:      c.cfg.MinGap,
		HTTPClient:  c.httpClient,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	c.mu.Lock()
	c.doer = doer
	c.mu.Unlock()

	// Verify the token. If this fails the client stays "not connected"
	// and doer is nil-ed out to avoid follow-up calls hitting a doomed
	// session.
	var u User
	if err := doer.JSON(ctx, http.MethodGet, "/api/v9/users/@me", nil, &u, nil); err != nil {
		c.mu.Lock()
		c.doer = nil
		c.mu.Unlock()
		if transport.IsUnauthorized(err) {
			return fmt.Errorf("%w: %v", ErrUnauthorized, err)
		}
		if transport.IsCloudflareBlocked(err) {
			return fmt.Errorf("%w: %v", ErrCloudflareBlocked, err)
		}
		return fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}

	c.mu.Lock()
	c.selfUser = &u
	c.connected = true
	c.mu.Unlock()
	return nil
}

// Disconnect zeroes the live transport but keeps the store + config so
// the client can Connect again.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	c.doer = nil
	c.connected = false
	c.mu.Unlock()
	return nil
}

// Close releases all resources.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	logDB := c.logDB
	c.logDB = nil
	c.doer = nil
	c.connected = false
	c.mu.Unlock()
	if logDB != nil {
		return logDB.Close()
	}
	return nil
}

// RateLimit returns a snapshot of the adaptive rate-limit state.
func (c *Client) RateLimit() transport.RateLimitState {
	c.mu.RLock()
	d := c.doer
	c.mu.RUnlock()
	if d == nil {
		return transport.RateLimitState{}
	}
	return d.RateLimit()
}

// channelAllowed enforces WithAllowedChannels.
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

// requireConfirm gates writes.
func (c *Client) requireConfirm(confirm bool) error {
	if !c.confirmWrites || confirm {
		return nil
	}
	return ErrConfirmRequired
}

// withDoer runs fn with the active doer under a read lock.
func (c *Client) withDoer(fn func(d *transport.Doer) error) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClosed
	}
	d := c.doer
	connected := c.connected
	c.mu.RUnlock()
	if d == nil || !connected {
		return ErrNotConnected
	}
	return fn(d)
}

func (c *Client) validateConfig() error {
	tok := strings.TrimSpace(c.token)
	if tok == "" {
		return fmt.Errorf("%w: Config.Token is required", ErrInvalidParams)
	}
	// Discord user tokens are <id>.<ts>.<sig> base64 chunks.
	// Bot tokens start with "Bot " or are clearly bot-shaped — refuse
	// them so callers don't try to drive this client with one (they'd
	// hit user-only endpoints and fail in confusing ways).
	if strings.HasPrefix(tok, "Bot ") || strings.HasPrefix(tok, "Bearer ") {
		return fmt.Errorf("%w: pass the raw user token (no Bot/Bearer prefix). For bots use the official bot API instead", ErrInvalidParams)
	}
	return nil
}

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

var _ = errors.Is // reserved for future error-classification helpers
