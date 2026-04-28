package discord

import (
	"context"
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
