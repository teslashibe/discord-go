package discord

import "errors"

var (
	ErrClosed            = errors.New("discord: client is closed")
	ErrNotConnected      = errors.New("discord: client is not connected; call Connect first")
	ErrAlreadyConnected  = errors.New("discord: client is already connected")
	ErrInvalidParams     = errors.New("discord: invalid parameters")
	ErrConfirmRequired   = errors.New("discord: confirm=true required for write operations")
	ErrChannelNotAllowed = errors.New("discord: channel is not in the configured allowlist")
	ErrGuildNotAllowed   = errors.New("discord: guild is not in the configured allowlist")
	ErrNotFound          = errors.New("discord: not found")
	ErrSendFailed        = errors.New("discord: send failed")
	ErrMessageEmpty      = errors.New("discord: message body is empty")
	ErrDMNotAllowed      = errors.New("discord: recipient does not allow DMs from this bot (no shared guild or DMs disabled)")
	ErrStoreInit         = errors.New("discord: failed to initialise local store")
	ErrAuthFailed        = errors.New("discord: authentication failed (bad token or revoked)")
)
