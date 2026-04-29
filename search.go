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
// matches in pages of 25; if Limit > 25 SearchMessages auto-paginates
// (using Offset) up to a hard internal cap of 200 matches per call to
// keep the rate-limit envelope sane.
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
	limit := p.Limit
	if limit <= 0 {
		limit = 25
	}
	const hardCap = 200
	if limit > hardCap {
		limit = hardCap
	}

	c.mu.RLock()
	selfID := ""
	if c.selfUser != nil {
		selfID = c.selfUser.ID
	}
	c.mu.RUnlock()

	out := SearchResult{}
	offset := p.Offset
	for len(out.Messages) < limit {
		page, total, err := c.searchPage(ctx, p, offset)
		if err != nil {
			return out, err
		}
		out.TotalResults = total
		if len(page) == 0 {
			break
		}
		for _, m := range page {
			if len(out.Messages) >= limit {
				break
			}
			if selfID != "" && m.Author.ID == selfID {
				m.IsFromMe = true
			}
			_ = c.upsertMessage(ctx, m)
			out.Messages = append(out.Messages, m)
		}
		offset += len(page)
		if offset >= total {
			break
		}
	}
	return out, nil
}

// searchPage fetches a single 25-row Discord search page.
func (c *Client) searchPage(ctx context.Context, p SearchParams, offset int) ([]Message, int, error) {
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
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
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
	var path string
	if p.GuildID != "" {
		path = "/api/v9/guilds/" + url.PathEscape(p.GuildID) + "/messages/search"
	} else {
		path = "/api/v9/channels/" + url.PathEscape(p.ChannelID) + "/messages/search"
	}
	var raw struct {
		TotalResults int            `json:"total_results"`
		Messages     [][]rawMessage `json:"messages"`
	}
	err := c.withDoer(func(d *transport.Doer) error {
		return d.JSON(ctx, http.MethodGet, path, nil, &raw, q)
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]Message, 0, len(raw.Messages))
	for _, group := range raw.Messages {
		if len(group) == 0 {
			continue
		}
		// Each hit-group is the matched message + context; surface the hit only.
		out = append(out, convertMessage(group[0]))
	}
	return out, raw.TotalResults, nil
}
