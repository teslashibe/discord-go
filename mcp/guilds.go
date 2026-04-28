package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	discord "github.com/teslashibe/discord-go"
)

type ListGuildsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=number of guilds to return (default 100\\, max 200),minimum=1,maximum=200"`
}

func listGuilds(ctx context.Context, c *discord.Client, in ListGuildsInput) (any, error) {
	return c.ListGuilds(ctx, discord.GuildListParams{Limit: in.Limit})
}

type GetGuildInput struct {
	GuildID string `json:"guild_id" jsonschema:"description=Discord guild snowflake id"`
}

func getGuild(ctx context.Context, c *discord.Client, in GetGuildInput) (any, error) {
	return c.GetGuild(ctx, in.GuildID)
}

var guildTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, ListGuildsInput](
		"discord_list_guilds",
		"List guilds the bot is in (id, name, icon URL)",
		"ListGuilds",
		listGuilds,
	),
	mcptool.Define[*discord.Client, GetGuildInput](
		"discord_get_guild",
		"Fetch a single guild with member counts (requires bot membership)",
		"GetGuild",
		getGuild,
	),
}
