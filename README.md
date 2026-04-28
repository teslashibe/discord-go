# discord-go

A Go client + MCP tool surface for Discord that authenticates as
**you** using your user-account token (the same one your browser
session uses), so an AI agent can read your DMs, your guilds, and
your message history — and reply on your behalf when you ask it to.

```go
import "github.com/teslashibe/discord-go"
```

> **READ THIS FIRST.** This is a user-token client (a "selfbot").
> Discord's Terms of Service forbid automating user accounts.
> Detection has gotten substantially better since 2024 and the
> punishment is **permanent account ban** — no warning, no appeal,
> you lose every DM, server, friend, and Nitro in one shot.
>
> This package is intended for **personal CRM-style relationship
> tracking on your own account**. It is shipped with safety knobs
> turned all the way up: REST polling only (no gateway socket),
> browser-faithful headers, adaptive rate limiting that respects
> Discord's `X-RateLimit-*` headers, and confirm-gated + allowlisted
> writes. Nothing automates around the ToS — it just looks more
> like a human using the web client than a naive bot.
>
> If you're building anything multi-tenant, customer-facing, or
> commercial, **use the official Bot API instead.**

## Architecture

```
┌──────────────────────────────────┐
│ MCP tools (discord_*)            │  thin wrappers over Client
├──────────────────────────────────┤
│ Client                           │  typed methods (Send, List, Search…)
│   ├─ messages.db (SQLite)        │  local log, populated by Watch polls
│   └─ token + cookie jar          │  Authorization header + Cloudflare cookies
├──────────────────────────────────┤
│ internal/transport               │  cookie jar, browser-faithful headers,
│                                   │  X-Super-Properties, adaptive throttle,
│                                   │  retry-with-backoff on 429/5xx
└──────────────────────────────────┘
```

## Authentication

You need your Discord user token. To extract it:

1. Open <https://discord.com/app> in Chrome/Firefox.
2. Open DevTools (`Cmd+Opt+I`) → **Console** tab.
3. Paste this and press Enter:
   ```js
   (webpackChunkdiscord_app.push([[''],{},e=>{m=[];for(let c in e.c)m.push(e.c[c])}]),m).find(m=>m?.exports?.default?.getToken!==void 0).exports.default.getToken()
   ```
4. Copy the string it prints (without the surrounding quotes).
5. Treat it like a password — anyone with it can read/send everything
   on your account. **Never commit it.** This package's `.gitignore`
   excludes `.env*`, `*.token`, `secrets/`, `discord_token.txt`.

```go
client := discord.New(discord.Config{
    Token:    os.Getenv("DISCORD_USER_TOKEN"),
    Locale:   "en-US",   // "X-Discord-Locale" header
    Timezone: "America/Los_Angeles",
    StoreDir: "/Users/me/.config/teslashibe/discord-go",
},
    discord.WithRequireConfirm(true),
    discord.WithAllowedChannels([]string{"123456789012345678"}),
    discord.WithMinGap(750*time.Millisecond), // human-pace floor
)
defer client.Close()

if err := client.Connect(ctx); err != nil { ... }

// Read your DMs
dms, _ := client.ListDMChannels(ctx, discord.DMListParams{Limit: 50})

// Search across a guild for a person you've been talking to
hits, _ := client.SearchMessages(ctx, discord.SearchParams{
    GuildID: "123…", Query: "alice", Limit: 25,
})

// Pull a conversation
msgs, _ := client.GetMessages(ctx, discord.MessageListParams{
    ChannelID: dms[0].ID, Limit: 100,
})

// Reply (confirm-gated)
_ = client.SendMessage(ctx, discord.SendParams{
    ChannelID: dms[0].ID, Body: "got it, thanks", Confirm: true,
})
```

## Capability surface

### Read
| Method | Tool |
|---|---|
| `Status` | `discord_status` |
| `ListGuilds` / `GetGuild` | `discord_list_guilds`, `discord_get_guild` |
| `ListChannels` / `GetChannel` | `discord_list_channels`, `discord_get_channel` |
| `ListDMChannels` | `discord_list_dm_channels` |
| `GetMessages` / `GetMessage` | `discord_get_messages`, `discord_get_message` |
| `SearchMessages` | `discord_search_messages` (user-only API — the killer feature) |
| `ListMembers` / `ResolveUser` | `discord_list_members`, `discord_resolve_user` |
| `Watch` | `discord_watch` (REST polling, never blocks) |

### Write (confirm-gated + channel-allowlisted)
| Method | Tool |
|---|---|
| `SendMessage` | `discord_send_message` |
| `EditMessage` | `discord_edit_message` |
| `DeleteMessage` | `discord_delete_message` |
| `React` / `Unreact` | `discord_react`, `discord_unreact` |

## Safety knobs

- `WithRequireConfirm(true)` (default) — every write tool requires
  `confirm=true` from the agent.
- `WithAllowedChannels([]string)` — sends/edits/deletes are denied
  for off-list channels. Strongly recommended.
- `WithAllowedGuilds([]string)` — same, but at the guild scope.
- `WithDryRun(true)` — write tools log + return without hitting
  Discord.
- `WithMinGap(d)` — minimum delay between any two HTTP calls
  (default 500ms — slower than naïve scripts, faster than humans).
- `WithBuildNumber(n)` / `WithUserAgent(s)` — override the values
  baked into `X-Super-Properties` if Discord's web build moves and
  you want to track it.

## Drift prevention

`mcp/mcp_test.go` runs `mcptool.Coverage` against `*discord.Client`.
Any new exported method that isn't wrapped by an MCP tool or listed
in `mcp.Excluded` (with a reason) fails the build.

## What's intentionally NOT in here

- **Gateway websocket.** A long-lived authenticated socket with the
  user token is the single biggest selfbot tell. `Watch` polls REST
  on a cadence you control instead.
- **Friend / block / DMS settings mutations.** You can read, you
  can't change account-level state from this package — the API
  exists, but it's high-signal automation behaviour.
- **Voice / video / presence updates.** Same reason.

## License

MIT (this package). Underlying libraries: see `go.sum`.
