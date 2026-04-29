package discord

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestLocalPaginationIsNumeric guards the snowflake-comparison fix in
// getMessagesLocal. With BeforeID=largeSnowflake we should still return
// messages whose snowflake is shorter (numerically smaller) — a lex
// compare would falsely include them since shorter strings sort
// "smaller" in some buckets.
func TestLocalPaginationIsNumeric(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	// Two messages: one "old" (small snowflake) and one "new" (big).
	// In lex order, "999...9" (18 chars) sorts before "1000...0" (19 chars).
	old := Message{
		ID:        "999999999999999999",
		ChannelID: "ch",
		Author:    User{ID: "u1", Username: "alice"},
		Body:      "old message",
		Timestamp: time.Unix(1000, 0).UTC(),
	}
	newer := Message{
		ID:        "1000000000000000000",
		ChannelID: "ch",
		Author:    User{ID: "u1", Username: "alice"},
		Body:      "new message",
		Timestamp: time.Unix(2000, 0).UTC(),
	}
	if err := c.upsertMessage(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := c.upsertMessage(ctx, newer); err != nil {
		t.Fatal(err)
	}

	// before_id = newer; want only the older message.
	got, err := c.getMessagesLocal(ctx, "ch", 10, newer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 message before %s, got %d", newer.ID, len(got))
	}
	if got[0].ID != old.ID {
		t.Errorf("want id=%s, got %s", old.ID, got[0].ID)
	}

	// before_id = older; want zero results.
	got, err = c.getMessagesLocal(ctx, "ch", 10, old.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 messages before %s, got %d", old.ID, len(got))
	}
}

// TestStripMentionRoleAndChannel verifies stripMention handles the
// shapes Discord emits in user inputs.
func TestStripMentionRoleAndChannel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"<@123>", "123"},
		{"<@!123>", "123"},
		{"<@&456>", "456"},
		{"plain", "plain"},
		{"  spaced  ", "spaced"},
	}
	for _, tc := range cases {
		if got := stripMention(tc.in); got != tc.want {
			t.Errorf("stripMention(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestUserAvatarURL covers both static and animated avatars.
func TestUserAvatarURL(t *testing.T) {
	u := User{ID: "1", Avatar: "abc"}
	if got := u.AvatarURL(); !strings.HasSuffix(got, ".png") {
		t.Errorf("static avatar should be .png, got %q", got)
	}
	a := User{ID: "1", Avatar: "a_anim"}
	if got := a.AvatarURL(); !strings.HasSuffix(got, ".gif") {
		t.Errorf("animated avatar should be .gif, got %q", got)
	}
	empty := User{}
	if got := empty.AvatarURL(); got != "" {
		t.Errorf("empty avatar should yield empty URL, got %q", got)
	}
}

// TestChannelTypeName covers the readable mapping.
func TestChannelTypeName(t *testing.T) {
	cases := map[int]string{
		0:   "text",
		1:   "dm",
		3:   "group_dm",
		11:  "public_thread",
		15:  "forum",
		999: "type_999",
	}
	for in, want := range cases {
		if got := channelTypeName(in); got != want {
			t.Errorf("channelTypeName(%d) = %q, want %q", in, got, want)
		}
	}
}

// newTestClient returns a Connected-looking client with a fresh
// SQLite store in t.TempDir().
func newTestClient(t *testing.T) *Client {
	t.Helper()
	dir := t.TempDir()
	c := New(Config{Token: "abc.def.ghi", StoreDir: dir})
	if err := c.initLogStore(context.Background()); err != nil {
		t.Fatalf("initLogStore: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestStatusClearsAuthorizedAfterDisconnect is the M1 audit regression.
// After Disconnect(), Status must not report stale Authorized=true.
func TestStatusClearsAuthorizedAfterDisconnect(t *testing.T) {
	c := newTestClient(t)
	// Simulate a successful connect by hand (no live token to probe).
	c.mu.Lock()
	c.connected = true
	c.selfUser = &User{ID: "1", Username: "alice"}
	c.mu.Unlock()

	rep, err := c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Authorized || !rep.Connected {
		t.Fatalf("baseline: should be Authorized+Connected, got %+v", rep)
	}

	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}
	rep, err = c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Authorized {
		t.Error("Status reports Authorized=true after Disconnect (stale selfUser)")
	}
	if rep.Connected {
		t.Error("Status reports Connected=true after Disconnect")
	}
	if rep.SelfID != "" {
		t.Error("Status reports stale SelfID after Disconnect")
	}
}

// TestCloseClearsAuthorized is the M1 audit regression for Close.
func TestCloseClearsAuthorized(t *testing.T) {
	c := newTestClient(t)
	c.mu.Lock()
	c.connected = true
	c.selfUser = &User{ID: "1", Username: "alice"}
	c.mu.Unlock()

	_ = c.Close()
	rep, _ := c.Status(context.Background())
	if rep.Authorized {
		t.Error("Status reports Authorized=true after Close")
	}
	if rep.Connected {
		t.Error("Status reports Connected=true after Close")
	}
}

// TestInitLogStoreOnClosedClientReturnsErrClosed is the M2 audit
// regression: opening the local DB on a closed client must fail.
func TestInitLogStoreOnClosedClientReturnsErrClosed(t *testing.T) {
	dir := t.TempDir()
	c := New(Config{Token: "abc.def.ghi", StoreDir: dir})
	_ = c.Close()
	if err := c.initLogStore(context.Background()); !errors.Is(err, ErrClosed) {
		t.Errorf("expected ErrClosed, got %v", err)
	}
}

// TestValidateConfigCaseVariantPrefixes is the M4 audit regression.
// "Bot ", "BOT ", "bearer " etc. should all be rejected.
func TestValidateConfigCaseVariantPrefixes(t *testing.T) {
	for _, tok := range []string{"Bot abc", "BOT abc", "bot abc", "Bearer xyz", "BEARER xyz", "bearer xyz"} {
		c := New(Config{Token: tok})
		if err := c.validateConfig(); !errors.Is(err, ErrInvalidParams) {
			t.Errorf("token %q should be rejected, got %v", tok, err)
		}
	}
	c := New(Config{Token: "real.user.token"})
	if err := c.validateConfig(); err != nil {
		t.Errorf("clean user token rejected: %v", err)
	}
}

// TestLastSeenMessageID is the H3 audit regression — Watch's poll path
// uses this to set after_id so >25 new messages don't get dropped.
func TestLastSeenMessageID(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	if got := c.lastSeenMessageID(ctx, "ch"); got != "" {
		t.Errorf("empty channel should yield empty id, got %q", got)
	}
	now := time.Unix(1000, 0).UTC()
	older := Message{ID: "100", ChannelID: "ch", Author: User{ID: "u"}, Body: "a", Timestamp: now}
	newer := Message{ID: "1000000000000000000", ChannelID: "ch", Author: User{ID: "u"}, Body: "b", Timestamp: now}
	if err := c.upsertMessage(ctx, older); err != nil {
		t.Fatal(err)
	}
	if err := c.upsertMessage(ctx, newer); err != nil {
		t.Fatal(err)
	}
	if got := c.lastSeenMessageID(ctx, "ch"); got != newer.ID {
		t.Errorf("lastSeenMessageID = %q, want %q (numeric, not lex compare)", got, newer.ID)
	}
}

// TestConcurrentInitLogStore is the M2 audit regression for the lost
// race when two goroutines call Connect concurrently — only one DB
// should land on the client.
func TestConcurrentInitLogStore(t *testing.T) {
	dir := t.TempDir()
	c := New(Config{Token: "abc.def.ghi", StoreDir: dir})
	t.Cleanup(func() { _ = c.Close() })
	const n = 8
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() { errs <- c.initLogStore(context.Background()) }()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent initLogStore: %v", err)
		}
	}
	if c.logDB == nil {
		t.Fatal("expected logDB to be set")
	}
	// Sanity: store works.
	if err := c.upsertMessage(context.Background(), Message{
		ID: "1", ChannelID: "ch", Author: User{ID: "u"}, Body: "x", Timestamp: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
}
