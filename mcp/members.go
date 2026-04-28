package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	discord "github.com/teslashibe/discord-go"
)

type ListMembersInput struct {
	GuildID string `json:"guild_id"`
	Limit   int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=1000"`
	AfterID string `json:"after_id,omitempty" jsonschema:"description=pagination: returns members with id > after_id"`
}

func listMembers(ctx context.Context, c *discord.Client, in ListMembersInput) (any, error) {
	return c.ListMembers(ctx, discord.MemberListParams{
		GuildID: in.GuildID, Limit: in.Limit, AfterID: in.AfterID,
	})
}

type ResolveUserInput struct {
	Ref string `json:"ref" jsonschema:"description=snowflake id\\, mention <@id>\\, or username[#discriminator]"`
}

func resolveUser(ctx context.Context, c *discord.Client, in ResolveUserInput) (any, error) {
	return c.ResolveUser(ctx, in.Ref)
}

var memberTools = []mcptool.Tool{
	mcptool.Define[*discord.Client, ListMembersInput](
		"discord_list_members",
		"List members of a guild; needs the GuildMembers privileged intent",
		"ListMembers",
		listMembers,
	),
	mcptool.Define[*discord.Client, ResolveUserInput](
		"discord_resolve_user",
		"Resolve a snowflake / mention / username#disc to a User",
		"ResolveUser",
		resolveUser,
	),
}
