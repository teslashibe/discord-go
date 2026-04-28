package discord

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teslashibe/discord-go/internal/transport"
)

func TestNewDefaults(t *testing.T) {
	c := New(Config{Token: "abc.def.ghi"})
	if c == nil {
		t.Fatal("New returned nil")
	}
	if !c.confirmWrites {
		t.Error("WithRequireConfirm default should be true")
	}
	if c.storeDir == "" {
		t.Error("StoreDir default should be set")
	}
	if !filepath.IsAbs(c.storeDir) {
		t.Errorf("storeDir should be absolute, got %q", c.storeDir)
	}
}

func TestRequireConfirm(t *testing.T) {
	c := New(Config{Token: "abc.def.ghi"})
	if err := c.requireConfirm(false); !errors.Is(err, ErrConfirmRequired) {
		t.Errorf("expected ErrConfirmRequired, got %v", err)
	}
	if err := c.requireConfirm(true); err != nil {
		t.Errorf("expected nil with confirm=true, got %v", err)
	}
	c2 := New(Config{Token: "abc.def.ghi"}, WithRequireConfirm(false))
	if err := c2.requireConfirm(false); err != nil {
		t.Errorf("expected nil with WithRequireConfirm(false), got %v", err)
	}
}

func TestAllowedChannels(t *testing.T) {
	c := New(Config{Token: "abc.def.ghi"}, WithAllowedChannels([]string{"123", "456"}))
	if err := c.channelAllowed("123"); err != nil {
		t.Errorf("123 should be allowed: %v", err)
	}
	if err := c.channelAllowed("999"); !errors.Is(err, ErrChannelNotAllowed) {
		t.Errorf("999 should be denied, got %v", err)
	}
	// All-empty allowlist disables the gate.
	c2 := New(Config{Token: "abc.def.ghi"}, WithAllowedChannels([]string{"", ""}))
	if err := c2.channelAllowed("999"); err != nil {
		t.Errorf("empty allowlist should disable the gate, got %v", err)
	}
}

func TestAllowedGuilds(t *testing.T) {
	c := New(Config{Token: "abc.def.ghi"}, WithAllowedGuilds([]string{"g1"}))
	if err := c.guildAllowed("g1"); err != nil {
		t.Errorf("g1 should be allowed: %v", err)
	}
	if err := c.guildAllowed("g2"); !errors.Is(err, ErrGuildNotAllowed) {
		t.Errorf("g2 should be denied, got %v", err)
	}
	// Empty guildID skips the check (we don't always have a guild context).
	if err := c.guildAllowed(""); err != nil {
		t.Errorf("empty guildID should pass through, got %v", err)
	}
}

func TestValidateConfig(t *testing.T) {
	c := New(Config{}) // missing Token
	if err := c.validateConfig(); !errors.Is(err, ErrInvalidParams) {
		t.Errorf("expected ErrInvalidParams for empty token, got %v", err)
	}
	c2 := New(Config{Token: "Bot abc.def.ghi"}) // bot token rejected
	if err := c2.validateConfig(); !errors.Is(err, ErrInvalidParams) {
		t.Errorf("expected ErrInvalidParams for Bot-prefixed token, got %v", err)
	}
	c3 := New(Config{Token: "Bearer xyz"}) // bearer token rejected
	if err := c3.validateConfig(); !errors.Is(err, ErrInvalidParams) {
		t.Errorf("expected ErrInvalidParams for Bearer-prefixed token, got %v", err)
	}
	c4 := New(Config{Token: "abc.def.ghi"})
	if err := c4.validateConfig(); err != nil {
		t.Errorf("user token should validate, got %v", err)
	}
}

func TestStatusReturnsHelpAndIsCallableBeforeConnect(t *testing.T) {
	c := New(Config{Token: "abc.def.ghi"})
	rep, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("Status before Connect should not error: %v", err)
	}
	if rep.Connected {
		t.Errorf("Connected should be false before Connect")
	}
	if rep.Authorized {
		t.Errorf("Authorized should be false before Connect")
	}
	if rep.HelpAuth == "" {
		t.Errorf("HelpAuth should be set when not connected")
	}
}

func TestNotConnectedReturnsErr(t *testing.T) {
	c := New(Config{Token: "abc.def.ghi"})
	if err := c.withDoer(func(_ *transport.Doer) error { return nil }); err == nil {
		t.Error("expected ErrNotConnected when called before Connect")
	}
}

func TestStripMention(t *testing.T) {
	cases := map[string]string{
		"<@123456789012345678>":  "123456789012345678",
		"<@!123456789012345678>": "123456789012345678",
		"<@&987>":                "987", // role mentions are stripped too
		"123456789012345678":     "123456789012345678",
		"  alice  ":              "alice",
	}
	for in, want := range cases {
		if got := stripMention(in); got != want {
			t.Errorf("stripMention(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsSnowflake(t *testing.T) {
	cases := map[string]bool{
		"123456789012345678":    true,
		"12345":                 false, // too short
		"123456789012345678901": false, // too long
		"123abc456789012345":    false, // non-numeric
		"":                      false,
	}
	for in, want := range cases {
		if got := isSnowflake(in); got != want {
			t.Errorf("isSnowflake(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSplitHandle(t *testing.T) {
	u, d := splitHandle("alice#1234")
	if u != "alice" || d != "1234" {
		t.Errorf("splitHandle alice#1234 = (%q,%q)", u, d)
	}
	u, d = splitHandle("alice")
	if u != "alice" || d != "" {
		t.Errorf("splitHandle alice = (%q,%q)", u, d)
	}
}

func TestDefaultStoreDirIsAbsolute(t *testing.T) {
	if got := defaultStoreDir(); !filepath.IsAbs(got) && !strings.HasPrefix(got, "./") {
		t.Errorf("defaultStoreDir() = %q, want absolute or ./-relative", got)
	}
}

func TestGuildIconURL(t *testing.T) {
	if got := guildIconURL("123", ""); got != "" {
		t.Errorf("empty icon should yield empty URL, got %q", got)
	}
	if got := guildIconURL("", "abc"); got != "" {
		t.Errorf("empty guildID should yield empty URL, got %q", got)
	}
	if got := guildIconURL("123", "abc"); got != "https://cdn.discordapp.com/icons/123/abc.png" {
		t.Errorf("png icon URL wrong: %q", got)
	}
	if got := guildIconURL("123", "a_anim"); got != "https://cdn.discordapp.com/icons/123/a_anim.gif" {
		t.Errorf("animated icon should be .gif, got %q", got)
	}
}
