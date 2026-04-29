package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"

	discord "github.com/teslashibe/discord-go"
)

type ListMembersInput struct {
	GuildID string `json:"guild_id" jsonschema:"description=guild snowflake id"`
	Limit   int    `json:"limit,omitempty" jsonschema:"description=default 100\\, max 1000,minimum=1,maximum=1000"`
	AfterID string `json:"after_id,omitempty" jsonschema:"description=user snowflake to paginate after"`
}

func listMembers(ctx context.Context, c *discord.Client, in ListMembersInput) (any, error) {
	return c.ListMembers(ctx, discord.MemberListParams{
		GuildID: in.GuildID, Limit: in.Limit, AfterID: in.AfterID,
	})
}

type ResolveUserInput struct {
	Ref string `json:"ref" jsonschema:"description=user snowflake\\, <@id> mention\\, or username/global-name fragment (latter is O(N guilds))"`
}

func resolveUser(ctx context.Context, c *discord.Client, in ResolveUserInput) (any, error) {
	return c.ResolveUser(ctx, in.Ref)
}

var memberTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, ListMembersInput](
		"discord_list_members",
		"List members of a guild (paginate via after_id)",
		"ListMembers",
		listMembers,
	),
	mcptool.Define[*discord.Client, ResolveUserInput](
		"discord_resolve_user",
		"Resolve a user by snowflake/mention/username; falls back to per-guild member search",
		"ResolveUser",
		resolveUser,
	),
}
