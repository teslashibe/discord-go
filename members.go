package discord

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/teslashibe/discord-go/internal/transport"
)

// ListMembers returns members of a guild. Discord caps the page size
// at 1000.
func (c *Client) ListMembers(ctx context.Context, params MemberListParams) ([]Member, error) {
	if params.GuildID == "" {
		return nil, fmt.Errorf("%w: GuildID required", ErrInvalidParams)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	if params.AfterID != "" {
		q.Set("after", params.AfterID)
	}
	var raw []rawMember
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet,
			"/api/v9/guilds/"+url.PathEscape(params.GuildID)+"/members",
			nil, &raw, q)
	})
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(raw))
	for _, m := range raw {
		out = append(out, Member{
			User: m.User, GuildID: params.GuildID, Nick: m.Nick,
			JoinedAt: m.JoinedAt, Roles: append([]string(nil), m.Roles...),
		})
	}
	return out, nil
}

// ResolveUser looks up a user. Accepts a snowflake id, "<@id>" or
// "<@!id>" mention, or username/global-name fragment (latter requires
// shared guild and uses guild member search).
//
// Cost note: the username/global-name path is O(N guilds) HTTP calls
// because Discord doesn't expose a global user-search endpoint to user
// accounts; pass a snowflake or mention whenever you can to short-
// circuit. Discord deprecated #discriminator handles in 2023, but the
// "alice#1234" form is still parsed for back-compat.
func (c *Client) ResolveUser(ctx context.Context, ref string) (User, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return User{}, fmt.Errorf("%w: ref required", ErrInvalidParams)
	}
	id := stripMention(ref)
	if isSnowflake(id) {
		var u User
		err := c.withDoer(func(d *transport.Doer) error {
			return d.JSON(ctx, http.MethodGet,
				"/api/v9/users/"+url.PathEscape(id), nil, &u, nil)
		})
		return u, err
	}

	username, disc := splitHandle(ref)
	guilds, err := c.ListGuilds(ctx, GuildListParams{Limit: 200})
	if err != nil {
		return User{}, err
	}
	// Match either the modern unique username (lowercase, no #) or the
	// global_name (display name) — Discord's pomelo migration left
	// both fields populated on member objects.
	wantLow := strings.ToLower(username)
	for _, g := range guilds {
		hits, err := c.guildMembersSearch(ctx, g.ID, username, 5)
		if err != nil {
			continue
		}
		for _, m := range hits {
			if disc != "" && m.User.Discriminator != disc {
				continue
			}
			if strings.ToLower(m.User.Username) != wantLow &&
				!strings.EqualFold(m.User.GlobalName, username) {
				continue
			}
			return m.User, nil
		}
	}
	return User{}, fmt.Errorf("%w: user %q not found in any of your guilds", ErrNotFound, ref)
}

// guildMembersSearch hits Discord's /guilds/{id}/members/search.
func (c *Client) guildMembersSearch(ctx context.Context, guildID, query string, limit int) ([]Member, error) {
	q := url.Values{}
	q.Set("query", query)
	q.Set("limit", strconv.Itoa(limit))
	var raw []rawMember
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet,
			"/api/v9/guilds/"+url.PathEscape(guildID)+"/members/search",
			nil, &raw, q)
	})
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(raw))
	for _, m := range raw {
		out = append(out, Member{
			User: m.User, GuildID: guildID, Nick: m.Nick,
			JoinedAt: m.JoinedAt, Roles: append([]string(nil), m.Roles...),
		})
	}
	return out, nil
}

type rawMember struct {
	User     User      `json:"user"`
	Nick     string    `json:"nick,omitempty"`
	JoinedAt time.Time `json:"joined_at"`
	Roles    []string  `json:"roles,omitempty"`
}

func stripMention(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "<@") || !strings.HasSuffix(s, ">") {
		return s
	}
	inner := s[2 : len(s)-1]
	inner = strings.TrimPrefix(inner, "!") // user-with-nick mention
	inner = strings.TrimPrefix(inner, "&") // role mention — caller's problem if they pass one
	return inner
}

func isSnowflake(s string) bool {
	if len(s) < 17 || len(s) > 20 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func splitHandle(s string) (username, discriminator string) {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
