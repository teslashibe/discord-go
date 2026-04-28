package discord

import "time"

// Guild summarises a Discord guild (server) the bot is in.
type Guild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	OwnerID     string `json:"ownerId,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`
	MemberCount int    `json:"memberCount,omitempty"`
}

// Channel summarises a guild channel (text/voice/category/thread/etc.).
type Channel struct {
	ID       string `json:"id"`
	GuildID  string `json:"guildId,omitempty"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Topic    string `json:"topic,omitempty"`
	NSFW     bool   `json:"nsfw,omitempty"`
	ParentID string `json:"parentId,omitempty"`
	Position int    `json:"position,omitempty"`
}

// User is a Discord user (or bot) reference.
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"globalName,omitempty"`
	Discriminator string `json:"discriminator,omitempty"` // legacy "#1234" tag, often "0" on new accounts
	Bot           bool   `json:"bot,omitempty"`
	System        bool   `json:"system,omitempty"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
}

// Member is a guild-scoped user (with guild nick + roles).
type Member struct {
	User     User      `json:"user"`
	GuildID  string    `json:"guildId"`
	Nick     string    `json:"nick,omitempty"`
	JoinedAt time.Time `json:"joinedAt,omitempty"`
	Roles    []string  `json:"roles,omitempty"`
}

// Reaction is one emoji + its count on a message.
type Reaction struct {
	Emoji string `json:"emoji"` // unicode emoji or "name:id" for custom
	Count int    `json:"count"`
	Me    bool   `json:"me"`
}

// Message is a single Discord message.
type Message struct {
	ID         string     `json:"id"`
	RowID      int64      `json:"rowId,omitempty"`     // local-log monotonic id (for Watch)
	ChannelID  string     `json:"channelId"`
	GuildID    string     `json:"guildId,omitempty"`
	Author     User       `json:"author"`
	IsFromMe   bool       `json:"isFromMe"`
	Body       string     `json:"body,omitempty"`
	HasMedia   bool       `json:"hasMedia,omitempty"`
	Attachments []string  `json:"attachments,omitempty"` // download URLs
	ReplyToID  string     `json:"replyToId,omitempty"`
	Pinned     bool       `json:"pinned,omitempty"`
	Mentions   []string   `json:"mentions,omitempty"` // mentioned user IDs
	Reactions  []Reaction `json:"reactions,omitempty"`
	Timestamp  time.Time  `json:"timestamp"`
	EditedAt   time.Time  `json:"editedAt,omitempty"`
}

// StatusReport summarises the current client state.
type StatusReport struct {
	StoreDir       string `json:"storeDir"`
	Connected      bool   `json:"connected"`
	Authorized     bool   `json:"authorized"`
	BotID          string `json:"botId,omitempty"`
	BotUsername    string `json:"botUsername,omitempty"`
	GuildCount     int    `json:"guildCount"`
	Intents        int    `json:"intents"`
	StoredMessages int64  `json:"storedMessages"`
	HelpAuth       string `json:"helpAuth,omitempty"`
}

// --- param structs ---

type GuildListParams struct {
	Limit int // default 100, max 200
}

type ChannelListParams struct {
	GuildID string
}

type MessageListParams struct {
	ChannelID string
	Limit     int    // default 50, max 100 (Discord cap)
	BeforeID  string // pagination: messages older than this id
	AfterID   string // pagination: messages newer than this id
	AroundID  string // pagination: messages around this id
	FromLocal bool   // true = read from local log only; false = call REST API
}

type MemberListParams struct {
	GuildID string
	Limit   int    // default 100, max 1000
	AfterID string // pagination
}

type SendParams struct {
	ChannelID string
	Body      string
	ReplyToID string // optional message id to reply to
	Silent    bool   // suppress @mentions notifications
	TTS       bool   // text-to-speech
	Confirm   bool
}

type EditParams struct {
	ChannelID string
	MessageID string
	Body      string
	Confirm   bool
}

type DeleteParams struct {
	ChannelID string
	MessageID string
	Confirm   bool
}

type BulkDeleteParams struct {
	ChannelID  string
	MessageIDs []string // 2–100 ids; must be <14 days old
	Confirm    bool
}

type ReactParams struct {
	ChannelID string
	MessageID string
	Emoji     string // unicode emoji or "name:id" for a custom emoji
	Confirm   bool
}

type UnreactParams struct {
	ChannelID string
	MessageID string
	Emoji     string
	UserID    string // empty = remove the bot's own reaction
	Confirm   bool
}

type SendDMParams struct {
	UserID  string
	Body    string
	Confirm bool
}

type WatchParams struct {
	SinceRowID int64
	ChannelID  string // optional filter
	GuildID    string // optional filter
	Limit      int
}

type WatchResult struct {
	Messages []Message `json:"messages"`
	Cursor   int64     `json:"cursor"`
}
