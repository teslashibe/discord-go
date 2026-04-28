package discord

import "time"

// Guild summarises a Discord guild (server) you are in.
type Guild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	OwnerID     string `json:"owner_id,omitempty"`
	IconURL     string `json:"icon_url,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
}

// Channel is a guild channel or DM/GroupDM channel.
type Channel struct {
	ID            string   `json:"id"`
	GuildID       string   `json:"guild_id,omitempty"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Topic         string   `json:"topic,omitempty"`
	NSFW          bool     `json:"nsfw,omitempty"`
	ParentID      string   `json:"parent_id,omitempty"`
	Position      int      `json:"position,omitempty"`
	Recipients    []User   `json:"recipients,omitempty"`     // populated for DM / group DM
	LastMessageID string   `json:"last_message_id,omitempty"`
}

// User is a Discord account.
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name,omitempty"`
	Discriminator string `json:"discriminator,omitempty"`
	Bot           bool   `json:"bot,omitempty"`
	System        bool   `json:"system,omitempty"`
	Avatar        string `json:"avatar,omitempty"` // hash; URL is built lazily
	Email         string `json:"email,omitempty"`  // self-only
	Phone         string `json:"phone,omitempty"`  // self-only
}

// AvatarURL returns the CDN URL for a user's avatar, or "" if none.
func (u User) AvatarURL() string {
	if u.Avatar == "" || u.ID == "" {
		return ""
	}
	ext := "png"
	if len(u.Avatar) > 2 && u.Avatar[:2] == "a_" {
		ext = "gif"
	}
	return "https://cdn.discordapp.com/avatars/" + u.ID + "/" + u.Avatar + "." + ext
}

// Member is a guild-scoped user (with guild nick + roles).
type Member struct {
	User     User      `json:"user"`
	GuildID  string    `json:"guild_id"`
	Nick     string    `json:"nick,omitempty"`
	JoinedAt time.Time `json:"joined_at,omitempty"`
	Roles    []string  `json:"roles,omitempty"`
}

// Reaction is one emoji + count on a message.
type Reaction struct {
	Emoji string `json:"emoji"` // unicode emoji or "name:id" for custom
	Count int    `json:"count"`
	Me    bool   `json:"me"`
}

// Message is a single Discord message.
type Message struct {
	ID          string     `json:"id"`
	RowID       int64      `json:"row_id,omitempty"` // local-log id, for Watch
	ChannelID   string     `json:"channel_id"`
	GuildID     string     `json:"guild_id,omitempty"`
	Author      User       `json:"author"`
	IsFromMe    bool       `json:"is_from_me"`
	Body        string     `json:"body,omitempty"`
	HasMedia    bool       `json:"has_media,omitempty"`
	Attachments []string   `json:"attachments,omitempty"`
	ReplyToID   string     `json:"reply_to_id,omitempty"`
	Pinned      bool       `json:"pinned,omitempty"`
	Mentions    []string   `json:"mentions,omitempty"`
	Reactions   []Reaction `json:"reactions,omitempty"`
	Timestamp   time.Time  `json:"timestamp"`
	EditedAt    time.Time  `json:"edited_at,omitempty"`
}

// StatusReport is what discord_status returns.
type StatusReport struct {
	StoreDir       string `json:"store_dir"`
	Connected      bool   `json:"connected"`
	Authorized     bool   `json:"authorized"`
	SelfID         string `json:"self_id,omitempty"`
	SelfUsername   string `json:"self_username,omitempty"`
	SelfGlobalName string `json:"self_global_name,omitempty"`
	StoredMessages int64  `json:"stored_messages"`
	HelpAuth       string `json:"help_auth,omitempty"`
}

// --- param structs ---

type GuildListParams struct {
	Limit int
}

type DMListParams struct {
	Limit int
}

type ChannelListParams struct {
	GuildID string
}

type MessageListParams struct {
	ChannelID string
	Limit     int
	BeforeID  string
	AfterID   string
	AroundID  string
	FromLocal bool
}

type SearchParams struct {
	GuildID   string // pass "" with ChannelID for DM-scoped search
	ChannelID string // optional
	AuthorID  string // optional: only by this user
	Query     string
	Limit     int
	Offset    int
	HasLink   bool
	HasFile   bool
	HasImage  bool
	HasVideo  bool
}

type SearchResult struct {
	TotalResults int       `json:"total_results"`
	Messages     []Message `json:"messages"`
}

type MemberListParams struct {
	GuildID string
	Limit   int
	AfterID string
}

type SendParams struct {
	ChannelID string
	Body      string
	ReplyToID string
	Silent    bool
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
	UserID    string // empty = remove your own reaction
	Confirm   bool
}

type WatchParams struct {
	SinceRowID int64
	ChannelID  string
	GuildID    string
	Limit      int
	// Poll, when true, refreshes the listed channels via REST before
	// returning the local cursor. PollChannels selects which channels.
	Poll         bool
	PollChannels []string
}

type WatchResult struct {
	Messages []Message `json:"messages"`
	Cursor   int64     `json:"cursor"`
	Polled   int       `json:"polled"`
}
