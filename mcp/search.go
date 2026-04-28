package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"

	discord "github.com/teslashibe/discord-go"
)

type SearchMessagesInput struct {
	GuildID   string `json:"guild_id,omitempty" jsonschema:"description=guild snowflake to scope the search (set this OR channel_id)"`
	ChannelID string `json:"channel_id,omitempty" jsonschema:"description=DM channel snowflake to scope the search (set this OR guild_id)"`
	AuthorID  string `json:"author_id,omitempty" jsonschema:"description=optional: only matches by this user snowflake"`
	Query     string `json:"query,omitempty" jsonschema:"description=full-text query (at least one of query/author_id/has_* required)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=25"`
	Offset    int    `json:"offset,omitempty" jsonschema:"description=paginate the result set in 25-message pages"`
	HasLink   bool   `json:"has_link,omitempty"`
	HasFile   bool   `json:"has_file,omitempty"`
	HasImage  bool   `json:"has_image,omitempty"`
	HasVideo  bool   `json:"has_video,omitempty"`
}

func searchMessages(ctx context.Context, c *discord.Client, in SearchMessagesInput) (any, error) {
	return c.SearchMessages(ctx, discord.SearchParams{
		GuildID:   in.GuildID,
		ChannelID: in.ChannelID,
		AuthorID:  in.AuthorID,
		Query:     in.Query,
		Limit:     in.Limit,
		Offset:    in.Offset,
		HasLink:   in.HasLink,
		HasFile:   in.HasFile,
		HasImage:  in.HasImage,
		HasVideo:  in.HasVideo,
	})
}

var searchTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, SearchMessagesInput](
		"discord_search_messages",
		"Server-side message search (user-only Discord feature) scoped to a guild or DM channel",
		"SearchMessages",
		searchMessages,
	),
}
