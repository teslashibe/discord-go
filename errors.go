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
	ErrStoreInit         = errors.New("discord: failed to initialise local store")
	ErrAuthFailed        = errors.New("discord: authentication failed")
	ErrUnauthorized      = errors.New("discord: token rejected by Discord (revoked, wrong, or invalidated)")
	ErrCloudflareBlocked = errors.New("discord: blocked by Cloudflare bot challenge — try a residential IP / non-VPN")
	ErrParseFailed       = errors.New("discord: failed to parse server response")
)
