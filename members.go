package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// ListMembers returns members of a guild. Requires the GuildMembers
// privileged intent. Pagination via AfterID.
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
	var out []Member
	err := c.withSession(func(s *discordgo.Session) error {
		ms, err := s.GuildMembers(params.GuildID, params.AfterID, limit)
		if err != nil {
			return fmt.Errorf("GuildMembers: %w", err)
		}
		for _, m := range ms {
			out = append(out, convertMember(m, params.GuildID))
		}
		return nil
	})
	return out, err
}

// ResolveUser looks up a user. Accepts a snowflake ID, a "<@123>" or
// "<@!123>" mention, or a username#discriminator handle (legacy).
//
// The username form is O(N guilds): the bot must share a guild with
// the user, and we GuildMembersSearch each guild until a match. For
// bots in many guilds prefer passing a snowflake or mention so the
// resolver can short-circuit on the first REST call.
func (c *Client) ResolveUser(ctx context.Context, ref string) (User, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return User{}, fmt.Errorf("%w: ref required", ErrInvalidParams)
	}
	// Mention form
	id := stripMention(ref)
	if isSnowflake(id) {
		var out User
		err := c.withSession(func(s *discordgo.Session) error {
			u, err := s.User(id)
			if err != nil {
				return fmt.Errorf("User: %w", err)
			}
			out = convertUser(u)
			return nil
		})
		return out, err
	}
	// username[#discriminator] form — needs a guild scope; search every
	// guild the bot is in until we find a match.
	username, disc := splitHandle(ref)
	var found User
	err := c.withSession(func(s *discordgo.Session) error {
		guilds, err := s.UserGuilds(200, "", "", false)
		if err != nil {
			return err
		}
		for _, g := range guilds {
			ms, err := s.GuildMembersSearch(g.ID, username, 5)
			if err != nil {
				continue
			}
			for _, m := range ms {
				if m == nil || m.User == nil {
					continue
				}
				if disc != "" && m.User.Discriminator != disc {
					continue
				}
				if !strings.EqualFold(m.User.Username, username) {
					continue
				}
				found = convertUser(m.User)
				return nil
			}
		}
		return fmt.Errorf("%w: user %q not found in any guild the bot is in", ErrNotFound, ref)
	})
	return found, err
}

func convertUser(u *discordgo.User) User {
	if u == nil {
		return User{}
	}
	return User{
		ID:            u.ID,
		Username:      u.Username,
		GlobalName:    u.GlobalName,
		Discriminator: u.Discriminator,
		Bot:           u.Bot,
		System:        u.System,
		AvatarURL:     u.AvatarURL(""),
	}
}

func convertMember(m *discordgo.Member, guildID string) Member {
	if m == nil {
		return Member{GuildID: guildID}
	}
	out := Member{
		GuildID:  guildID,
		Nick:     m.Nick,
		JoinedAt: m.JoinedAt,
		Roles:    append([]string(nil), m.Roles...),
	}
	if m.User != nil {
		out.User = convertUser(m.User)
	}
	return out
}

func stripMention(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "<@") || !strings.HasSuffix(s, ">") {
		return s
	}
	inner := s[2 : len(s)-1]
	inner = strings.TrimPrefix(inner, "!") // user-with-nick mention
	inner = strings.TrimPrefix(inner, "&") // role mention — not a user, but still digits
	return inner
}

// isSnowflake returns true for an all-digit string of typical Discord
// snowflake length (17–20 digits).
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

// splitHandle parses "name" or "name#1234".
func splitHandle(s string) (username, discriminator string) {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

