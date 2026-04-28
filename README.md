# discord-go

A Go client + MCP tool surface for Discord, built on
[`github.com/bwmarrin/discordgo`](https://github.com/bwmarrin/discordgo)
— the official-grade community Discord library.

```go
import "github.com/teslashibe/discord-go"
```

Same shape as `imessage-go`, `whatsapp-go`, `telegram-go`,
`linkedin-go`, `zillow-go`. Reads guilds, channels, and message
history; sends messages, edits, deletes, reactions, and bulk-deletes
on behalf of a Discord **bot**.

> **Bot only — not selfbot.** Discord's ToS bans automating user
> accounts; getting caught wipes the account. This package only
> supports a real bot identity (token from
> <https://discord.com/developers/applications>) which the user has
> explicitly invited to each guild.

## Architecture

```
┌──────────────────────────────────┐
│ MCP tools (discord_*)            │  thin wrappers over Client
├──────────────────────────────────┤
│ Client                           │  typed methods (Send, List, …)
│   └─ messages.db (SQLite)        │  local log populated by gateway
├──────────────────────────────────┤
│ discordgo                        │  REST + Gateway (websocket)
└──────────────────────────────────┘
```

Discord's REST API serves message history on demand, so reads always
fall back to the live API when needed. The local SQLite log exists for
fast `Watch` polling — populated by the gateway's `MessageCreate` /
`MessageUpdate` events delivered while the client is `Connect`ed.

## Authentication

1. Create a bot at <https://discord.com/developers/applications> →
   New Application → Bot → Reset Token (the token is shown once).
2. Enable **Privileged Gateway Intents** as needed:
   - **Message Content** (required to read message bodies)
   - **Server Members** (required for `ListMembers`)
3. Generate an OAuth2 invite URL with the `bot` scope and the
   permissions you need (Read Messages, Send Messages, Manage
   Messages for delete/edit-on-behalf, Add Reactions, …).
4. Pass the token via env or config:

```go
client := discord.New(discord.Config{
    Token: os.Getenv("DISCORD_BOT_TOKEN"),
    Intents: discord.IntentsGuilds |
        discord.IntentsGuildMessages |
        discord.IntentsMessageContent |
        discord.IntentsGuildMessageReactions,
},
    discord.WithRequireConfirm(true),
    discord.WithAllowedChannels([]string{"123456789012345678"}),
)
defer client.Close()

if err := client.Connect(ctx); err != nil { ... }

guilds, _ := client.ListGuilds(ctx, discord.GuildListParams{})
msgs, _ := client.GetMessages(ctx, discord.MessageListParams{
    ChannelID: "123…", Limit: 50,
})
_ = client.SendMessage(ctx, discord.SendParams{
    ChannelID: "123…", Body: "ack", Confirm: true,
})
```

## Capability surface

### V1 (read)
| Method | Tool |
|---|---|
| `Status` | `discord_status` |
| `ListGuilds` / `GetGuild` | `discord_list_guilds`, `discord_get_guild` |
| `ListChannels` / `GetChannel` | `discord_list_channels`, `discord_get_channel` |
| `GetMessages` | `discord_get_messages` |
| `GetMessage` | `discord_get_message` |
| `ListMembers` | `discord_list_members` |
| `ResolveUser` | `discord_resolve_user` |
| `Watch` | `discord_watch` |

### V1 (write — confirm-gated)
| Method | Tool |
|---|---|
| `SendMessage` | `discord_send_message` |
| `EditMessage` | `discord_edit_message` |
| `DeleteMessage` | `discord_delete_message` |
| `BulkDeleteMessages` | `discord_bulk_delete_messages` |
| `React` / `Unreact` | `discord_react`, `discord_unreact` |
| `SendDM` | `discord_send_dm` |

## Safety knobs

- `WithRequireConfirm(true)` (default) — every write tool requires
  `confirm=true` from the agent.
- `WithAllowedChannels([]string)` — sends/edits/deletes are denied
  for off-list channels.
- `WithAllowedGuilds([]string)` — same, but at the guild scope.
- `WithDryRun(true)` — write tools log + return without hitting the
  Discord API.

## Drift prevention

`mcp/mcp_test.go` runs `mcptool.Coverage` against `*discord.Client`.
Any new exported method that isn't wrapped by an MCP tool or listed in
`mcp.Excluded` (with a reason) fails the build.

## Notes on the API surface

- **No message search**: Discord's `/search` REST endpoint is
  user-only; bots cannot query it. Search is intentionally not in
  the surface — agents should fetch a window with `GetMessages`
  and filter client-side.
- **Bots can DM users only after a shared guild**: `SendDM` opens a
  DM channel via `UserChannelCreate` and posts there. If the user
  has DMs disabled this fails with `ErrDMNotAllowed`.
- **Rate limits**: discordgo handles `429` with bucket-aware backoff
  internally; we don't wrap it.

## License

MIT (this package). Underlying `discordgo` is BSD-3-Clause.
