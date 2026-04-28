package discord

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/teslashibe/discord-go/internal/transport"
)

// SearchMessages runs Discord's user-only search endpoint. Either
// GuildID or ChannelID (DM-scoped) must be provided. Discord returns
// matches in pages of 25; pass Offset to paginate.
//
// The endpoint is rate-limited per-guild and the index lags real-time
// by a few seconds (Discord uses Elasticsearch behind it). For
// near-real-time matches, fall back to GetMessages + filter.
func (c *Client) SearchMessages(ctx context.Context, p SearchParams) (SearchResult, error) {
	if p.GuildID == "" && p.ChannelID == "" {
		return SearchResult{}, fmt.Errorf("%w: GuildID or ChannelID required", ErrInvalidParams)
	}
	if p.Query == "" && p.AuthorID == "" && !p.HasLink && !p.HasFile && !p.HasImage && !p.HasVideo {
		return SearchResult{}, fmt.Errorf("%w: at least one of Query / AuthorID / Has* required", ErrInvalidParams)
	}
	q := url.Values{}
	if p.Query != "" {
		q.Set("content", p.Query)
	}
	if p.AuthorID != "" {
		q.Set("author_id", p.AuthorID)
	}
	if p.ChannelID != "" {
		q.Set("channel_id", p.ChannelID)
	}
	if p.Offset > 0 {
		q.Set("offset", strconv.Itoa(p.Offset))
	}
	if p.HasLink {
		q.Add("has", "link")
	}
	if p.HasFile {
		q.Add("has", "file")
	}
	if p.HasImage {
		q.Add("has", "image")
	}
	if p.HasVideo {
		q.Add("has", "video")
	}

	// path = /api/v9/guilds/{id}/messages/search OR
	//        /api/v9/channels/{id}/messages/search (DM scope)
	var path string
	if p.GuildID != "" {
		path = "/api/v9/guilds/" + url.PathEscape(p.GuildID) + "/messages/search"
	} else {
		path = "/api/v9/channels/" + url.PathEscape(p.ChannelID) + "/messages/search"
	}

	// Discord's search response wraps matches in [[hit]] arrays.
	var raw struct {
		TotalResults     int            `json:"total_results"`
		Messages         [][]rawMessage `json:"messages"`
		AnalyticsID      string         `json:"analytics_id,omitempty"`
		DocLimitExceeded bool           `json:"doc_limit_exceeded,omitempty"`
	}
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet, path, nil, &raw, q)
	})
	if err != nil {
		return SearchResult{}, err
	}

	c.mu.RLock()
	selfID := ""
	if c.selfUser != nil {
		selfID = c.selfUser.ID
	}
	c.mu.RUnlock()

	out := SearchResult{TotalResults: raw.TotalResults}
	limit := p.Limit
	if limit <= 0 {
		limit = len(raw.Messages)
	}
	for _, group := range raw.Messages {
		if limit <= 0 {
			break
		}
		// Each hit-group is the matched message + context; the hit is
		// always the first element. Lower the noise by surfacing just
		// the hit.
		if len(group) == 0 {
			continue
		}
		m := convertMessage(group[0])
		if selfID != "" && m.Author.ID == selfID {
			m.IsFromMe = true
		}
		_ = c.upsertMessage(ctx, m)
		out.Messages = append(out.Messages, m)
		limit--
	}
	return out, nil
}
