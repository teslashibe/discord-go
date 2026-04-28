package discord

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewDefaults(t *testing.T) {
	c := New(Config{Token: "tok"})
	if c == nil {
		t.Fatal("New returned nil")
	}
	if !c.confirmWrites {
		t.Error("WithRequireConfirm default should be true")
	}
	if c.intents == 0 {
		t.Error("intents should default to DefaultIntents")
	}
	if !filepath.IsAbs(c.storeDir) {
		t.Errorf("storeDir %q should be absolute", c.storeDir)
	}
}

func TestRequireConfirm(t *testing.T) {
	c := New(Config{Token: "t"})
	if err := c.requireConfirm(false); !errors.Is(err, ErrConfirmRequired) {
		t.Errorf("expected ErrConfirmRequired, got %v", err)
	}
	if err := c.requireConfirm(true); err != nil {
		t.Errorf("expected nil with confirm=true, got %v", err)
	}
	c2 := New(Config{Token: "t"}, WithRequireConfirm(false))
	if err := c2.requireConfirm(false); err != nil {
		t.Errorf("expected nil with WithRequireConfirm(false), got %v", err)
	}
}

func TestAllowedChannels(t *testing.T) {
	c := New(Config{Token: "t"}, WithAllowedChannels([]string{"123", "456"}))
	if err := c.channelAllowed("123"); err != nil {
		t.Errorf("123 should be allowed: %v", err)
	}
	if err := c.channelAllowed("999"); !errors.Is(err, ErrChannelNotAllowed) {
		t.Errorf("999 should be denied, got %v", err)
	}
	// All-empty list disables the allowlist.
	c2 := New(Config{Token: "t"}, WithAllowedChannels([]string{"", ""}))
	if err := c2.channelAllowed("123"); err != nil {
		t.Errorf("expected disabled allowlist, got %v", err)
	}
}

func TestAllowedGuilds(t *testing.T) {
	c := New(Config{Token: "t"}, WithAllowedGuilds([]string{"g1"}))
	if err := c.guildAllowed("g1"); err != nil {
		t.Errorf("g1 should be allowed: %v", err)
	}
	if err := c.guildAllowed("g2"); !errors.Is(err, ErrGuildNotAllowed) {
		t.Errorf("g2 should be denied")
	}
	// Empty guildID is always allowed (DM scope).
	if err := c.guildAllowed(""); err != nil {
		t.Errorf("empty guild should be allowed")
	}
}

func TestValidateConfig(t *testing.T) {
	c := New(Config{}) // missing Token
	if err := c.validateConfig(); !errors.Is(err, ErrInvalidParams) {
		t.Errorf("expected ErrInvalidParams, got %v", err)
	}
}

func TestStatusBeforeConnect(t *testing.T) {
	c := New(Config{Token: "t"})
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
	c := New(Config{Token: "t"})
	if err := c.withSession(func(_ *discordgo.Session) error { return nil }); err == nil {
		t.Error("expected ErrNotConnected when called before Connect")
	}
}

func TestStripMentionAndSnowflake(t *testing.T) {
	cases := map[string]string{
		"<@123456789012345678>":  "123456789012345678",
		"<@!123456789012345678>": "123456789012345678",
		"123456789012345678":     "123456789012345678",
		"foo#1234":               "foo#1234",
	}
	for in, want := range cases {
		if got := stripMention(in); got != want {
			t.Errorf("stripMention(%q) = %q, want %q", in, got, want)
		}
	}
	if !isSnowflake("123456789012345678") {
		t.Error("expected isSnowflake true for 18 digits")
	}
	if isSnowflake("abc") {
		t.Error("expected isSnowflake false for letters")
	}
	if isSnowflake("12345") {
		t.Error("expected isSnowflake false for too short")
	}
}

func TestSplitHandle(t *testing.T) {
	u, d := splitHandle("alice")
	if u != "alice" || d != "" {
		t.Errorf("splitHandle(alice) = %q,%q", u, d)
	}
	u, d = splitHandle("alice#1234")
	if u != "alice" || d != "1234" {
		t.Errorf("splitHandle(alice#1234) = %q,%q", u, d)
	}
}
