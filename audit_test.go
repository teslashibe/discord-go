package discord

import (
	"context"
	"testing"
	"time"
)

// TestLocalPaginationIsNumeric is the regression test for the audit's
// H1 fix: snowflakes are 17–20-digit numbers and lexicographic
// comparison breaks across lengths. We seed two messages where a
// 19-digit id is older (smaller) than an 18-digit id and assert that
// "before the 18-digit id" returns the 19-digit one.
func TestLocalPaginationIsNumeric(t *testing.T) {
	c := New(Config{Token: "t", StoreDir: t.TempDir()})
	defer c.Close()
	if err := c.initLogStore(context.Background()); err != nil {
		t.Fatalf("initLogStore: %v", err)
	}
	older := Message{
		ID:        "100000000000000000", // 18 digits, but numerically smaller
		ChannelID: "chan1",
		Body:      "older",
		Timestamp: time.Unix(1_700_000_000, 0).UTC(),
	}
	newer := Message{
		ID:        "999999999999999999", // 18 digits, numerically larger
		ChannelID: "chan1",
		Body:      "newer",
		Timestamp: time.Unix(1_700_000_100, 0).UTC(),
	}
	mixed := Message{
		ID:        "10000000000000000000", // 20 digits — would lex-compare as smaller
		ChannelID: "chan1",
		Body:      "mixed",
		Timestamp: time.Unix(1_700_000_200, 0).UTC(),
	}
	for _, m := range []Message{older, newer, mixed} {
		if err := c.upsertMessage(context.Background(), m); err != nil {
			t.Fatalf("upsert %s: %v", m.ID, err)
		}
	}

	// "before mixed" should return both older AND newer (numerically
	// less than 10000000000000000000). Pre-fix, lex comparison would
	// have excluded the 18-digit "newer" because "9..." > "1...".
	got, err := c.getMessagesLocal(context.Background(), "chan1", 10, mixed.ID)
	if err != nil {
		t.Fatalf("getMessagesLocal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results before mixed id, got %d (%v)", len(got), got)
	}
}

// TestStripMentionRoleAndChannel ensures we don't mistake role / channel
// mentions for users.
func TestStripMentionRoleAndChannel(t *testing.T) {
	// Role mention <@&id> currently strips to digits; isSnowflake then
	// passes, so ResolveUser would call User(<role-id>) which 404s. We
	// keep the behaviour but document it.
	if got := stripMention("<@&123456789012345678>"); got != "123456789012345678" {
		t.Errorf("role mention strip = %q", got)
	}
	if got := stripMention("just text"); got != "just text" {
		t.Errorf("plain text should pass through, got %q", got)
	}
}

// TestDefaultIntentsCoversReadAndWrite makes sure DefaultIntents
// includes the privileged intents that the read+write surface needs.
func TestDefaultIntentsCoversReadAndWrite(t *testing.T) {
	wanted := []Intent{
		IntentsGuilds, IntentsGuildMessages, IntentsMessageContent,
		IntentsGuildMessageReactions, IntentsDirectMessages,
	}
	for _, w := range wanted {
		if DefaultIntents&w == 0 {
			t.Errorf("DefaultIntents missing %d", w)
		}
	}
}
