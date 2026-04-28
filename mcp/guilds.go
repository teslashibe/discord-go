package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"

	discord "github.com/teslashibe/discord-go"
)

type ListGuildsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=default 100\\, max 200,minimum=1,maximum=200"`
}

func listGuilds(ctx context.Context, c *discord.Client, in ListGuildsInput) (any, error) {
	return c.ListGuilds(ctx, discord.GuildListParams{Limit: in.Limit})
}

type GetGuildInput struct {
	GuildID string `json:"guild_id" jsonschema:"description=guild snowflake id"`
}

func getGuild(ctx context.Context, c *discord.Client, in GetGuildInput) (any, error) {
	return c.GetGuild(ctx, in.GuildID)
}

var guildTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, ListGuildsInput](
		"discord_list_guilds",
		"List the Discord guilds (servers) the authed user is in",
		"ListGuilds",
		listGuilds,
	),
	mcptool.Define[*discord.Client, GetGuildInput](
		"discord_get_guild",
		"Fetch a single guild by snowflake id",
		"GetGuild",
		getGuild,
	),
}
