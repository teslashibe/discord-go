// Package transport is the low-level HTTP layer for discord-go.
//
// It mirrors the request shape of discord.com's web client so requests
// don't stand out: cookie jar (Cloudflare's __cf_bm + __dcfduid live
// here), browser-faithful headers (UA, sec-ch-ua, sec-fetch-*,
// X-Super-Properties, X-Discord-Locale, X-Discord-Timezone), and an
// adaptive rate limiter that respects Discord's X-RateLimit-* and
// Retry-After headers.
//
// Failure modes:
//   - HTTP 401 → token revoked or wrong → ErrUnauthorized
//   - HTTP 403 with Cloudflare banner → ErrCloudflareBlocked
//   - HTTP 429 with Retry-After → automatic backoff + retry (up to 3)
//   - HTTP 5xx → exponential backoff + retry (up to 3)
//
// The transport never logs the Authorization header.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// BaseURL is the Discord API root.
const BaseURL = "https://discord.com"

// MaxBody bounds the body we read from non-2xx responses for the error
// message (responses are otherwise drained for connection reuse).
const MaxBody = 16 << 10 // 16 KiB

const (
	defaultUserAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	defaultBuildNumber    = 308352
	defaultClientVersion  = "1.0.9032"
	defaultLocale         = "en-US"
	defaultMaxRetries     = 3
	defaultMinGap         = 500 * time.Millisecond
	defaultRequestTimeout = 30 * time.Second
)

// Doer is what callers see; package-private fields keep the type honest.
type Doer struct {
	httpClient *http.Client
	jar        *cookiejar.Jar
	logger     Logger

	token       string
	userAgent   string
	locale      string
	timezone    string
	buildNumber int
	clientVer   string

	gateMu  sync.Mutex
	lastCall time.Time
	minGap   time.Duration
	maxRetries int

	// Rate-limit state surfaced via RateLimit().
	rlMu    sync.RWMutex
	rlState RateLimitState
}

// Logger is the minimal logger interface Doer needs.
type Logger interface {
	Warn(msg string, kv ...any)
}

type nopLogger struct{}

func (nopLogger) Warn(string, ...any) {}

// Options configures a Doer.
type Options struct {
	Token       string
	UserAgent   string
	Locale      string
	Timezone    string
	BuildNumber int
	ClientVer   string
	MinGap      time.Duration
	MaxRetries  int
	Logger      Logger
	HTTPClient  *http.Client
}

// New constructs a Doer.
func New(opts Options) (*Doer, error) {
	if opts.Token == "" {
		return nil, errors.New("transport: Token required")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("transport: cookie jar: %w", err)
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: defaultRequestTimeout, Jar: jar}
	} else {
		hc.Jar = jar
	}
	d := &Doer{
		httpClient:  hc,
		jar:         jar,
		logger:      opts.Logger,
		token:       opts.Token,
		userAgent:   firstNonEmpty(opts.UserAgent, defaultUserAgent),
		locale:      firstNonEmpty(opts.Locale, defaultLocale),
		timezone:    firstNonEmpty(opts.Timezone, "America/Los_Angeles"),
		buildNumber: orInt(opts.BuildNumber, defaultBuildNumber),
		clientVer:   firstNonEmpty(opts.ClientVer, defaultClientVersion),
		minGap:      orDuration(opts.MinGap, defaultMinGap),
		maxRetries:  orInt(opts.MaxRetries, defaultMaxRetries),
	}
	if d.logger == nil {
		d.logger = nopLogger{}
	}
	return d, nil
}

// CookieJar exposes the jar (for tests).
func (d *Doer) CookieJar() http.CookieJar { return d.jar }

// JSON performs a request and decodes a JSON response body into out.
// method = GET/POST/PATCH/DELETE/PUT, path = "/api/v9/...", body =
// optional JSON-marshalable input, out = optional pointer to decode
// into. q is the query string (already encoded) appended after path.
func (d *Doer) JSON(ctx context.Context, method, path string, body any, out any, q url.Values) error {
	var attempt int
	for {
		err := d.do(ctx, method, path, body, out, q)
		if err == nil {
			return nil
		}
		var he *HTTPError
		if errors.As(err, &he) && d.shouldRetry(he, attempt) {
			d.sleepForRetry(ctx, he, attempt)
			attempt++
			continue
		}
		return err
	}
}

func (d *Doer) do(ctx context.Context, method, path string, body any, out any, q url.Values) error {
	if err := d.gate(ctx); err != nil {
		return err
	}

	u := BaseURL + path
	if len(q) > 0 {
		if hasQ := containsByte(path, '?'); hasQ {
			u = u + "&" + q.Encode()
		} else {
			u = u + "?" + q.Encode()
		}
	}

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("transport: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("transport: build request: %w", err)
	}
	d.applyHeaders(req, body != nil)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	d.updateRateLimit(resp)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil || resp.StatusCode == http.StatusNoContent {
			io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain for keep-alive
			return nil
		}
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(out); err != nil && err != io.EOF {
			return fmt.Errorf("transport: decode %s %s: %w", method, path, err)
		}
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		return nil
	}

	// Non-2xx: capture a bounded prefix for the error, drain the rest.
	limited := io.LimitReader(resp.Body, MaxBody)
	bodyBytes, _ := io.ReadAll(limited)
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	he := &HTTPError{
		Status:    resp.StatusCode,
		Method:    method,
		Path:      path,
		Body:      string(bodyBytes),
		Retry:     parseRetryAfter(resp.Header),
		CFBlocked: looksCloudflareBlocked(resp, bodyBytes),
	}
	return he
}

// gate is the throttle. Honours minGap + adaptive backoff hints.
//
// Implementation note: we reserve the earliest legal slot inside the
// mutex, then wait outside of it, so concurrent callers serialise
// across the gap rather than all racing the same wakeup.
func (d *Doer) gate(ctx context.Context) error {
	d.gateMu.Lock()
	now := time.Now()
	earliest := d.lastCall.Add(d.adaptiveGapLocked())
	if earliest.Before(now) {
		earliest = now
	}
	d.lastCall = earliest
	d.gateMu.Unlock()

	wait := time.Until(earliest)
	if wait <= 0 {
		return nil
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (d *Doer) adaptiveGapLocked() time.Duration {
	d.rlMu.RLock()
	defer d.rlMu.RUnlock()
	gap := d.minGap
	if d.rlState.MinGap > gap {
		gap = d.rlState.MinGap
	}
	return gap
}

// applyHeaders adds the browser-faithful header set.
func (d *Doer) applyHeaders(req *http.Request, hasBody bool) {
	h := req.Header
	h.Set("Authorization", d.token) // user-token: NO "Bot " prefix
	h.Set("User-Agent", d.userAgent)
	h.Set("Accept", "*/*")
	h.Set("Accept-Language", d.locale+",en;q=0.9")
	h.Set("Origin", "https://discord.com")
	h.Set("Referer", "https://discord.com/channels/@me")
	h.Set("X-Discord-Locale", d.locale)
	h.Set("X-Discord-Timezone", d.timezone)
	h.Set("X-Super-Properties", d.superProperties())
	h.Set("X-Debug-Options", "bugReporterEnabled")
	h.Set("Sec-Fetch-Dest", "empty")
	h.Set("Sec-Fetch-Mode", "cors")
	h.Set("Sec-Fetch-Site", "same-origin")
	h.Set("Sec-Ch-Ua", `"Google Chrome";v="124", "Chromium";v="124", "Not-A.Brand";v="99"`)
	h.Set("Sec-Ch-Ua-Mobile", "?0")
	h.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	if hasBody {
		h.Set("Content-Type", "application/json")
	}
}

// superProperties is the X-Super-Properties header that Discord's web
// client sends. base64(JSON of os/browser/build info). See README.
func (d *Doer) superProperties() string {
	props := map[string]any{
		"os":                       "Mac OS X",
		"browser":                  "Chrome",
		"device":                   "",
		"system_locale":            d.locale,
		"browser_user_agent":       d.userAgent,
		"browser_version":          "124.0.0.0",
		"os_version":               "10.15.7",
		"referrer":                 "",
		"referring_domain":         "",
		"referrer_current":         "",
		"referring_domain_current": "",
		"release_channel":          "stable",
		"client_build_number":      d.buildNumber,
		"client_event_source":      nil,
	}
	buf, _ := json.Marshal(props)
	return base64Std(buf)
}

// updateRateLimit reads Discord's X-RateLimit-* headers and adjusts the
// minGap upward if the bucket is close to empty.
func (d *Doer) updateRateLimit(resp *http.Response) {
	rem := resp.Header.Get("X-RateLimit-Remaining")
	reset := resp.Header.Get("X-RateLimit-Reset-After")
	bucket := resp.Header.Get("X-RateLimit-Bucket")
	if rem == "" {
		return
	}
	remN, _ := strconv.Atoi(rem)
	resetF, _ := strconv.ParseFloat(reset, 64)

	d.rlMu.Lock()
	defer d.rlMu.Unlock()
	d.rlState.LastBucket = bucket
	d.rlState.LastRemaining = remN
	d.rlState.LastResetAfter = time.Duration(resetF * float64(time.Second))
	// If we're close to empty, slow down so the next call doesn't 429.
	if remN <= 1 && resetF > 0 {
		gap := time.Duration(resetF * float64(time.Second))
		if gap > d.rlState.MinGap {
			d.rlState.MinGap = gap
		}
	} else if remN > 5 {
		// Plenty of budget; let the floor relax back to default.
		d.rlState.MinGap = 0
	}
}

// shouldRetry decides whether a non-2xx is worth retrying.
func (d *Doer) shouldRetry(he *HTTPError, attempt int) bool {
	if attempt >= d.maxRetries {
		return false
	}
	if he.Status == http.StatusTooManyRequests {
		return true
	}
	if he.Status >= 500 && he.Status < 600 {
		return true
	}
	return false
}

func (d *Doer) sleepForRetry(ctx context.Context, he *HTTPError, attempt int) {
	wait := he.Retry
	if wait <= 0 {
		// Exponential backoff with jitter: 500ms, 1s, 2s
		wait = time.Duration(500*(1<<attempt)) * time.Millisecond
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return
	case <-t.C:
	}
}

// RateLimitState is exported for diagnostics.
type RateLimitState struct {
	LastBucket     string        `json:"lastBucket,omitempty"`
	LastRemaining  int           `json:"lastRemaining"`
	LastResetAfter time.Duration `json:"lastResetAfter"`
	MinGap         time.Duration `json:"minGap"`
}

// RateLimit returns a snapshot of the adaptive rate-limit state.
func (d *Doer) RateLimit() RateLimitState {
	d.rlMu.RLock()
	defer d.rlMu.RUnlock()
	return d.rlState
}

// HTTPError is returned for any non-2xx response.
type HTTPError struct {
	Status    int
	Method    string
	Path      string
	Body      string
	Retry     time.Duration
	CFBlocked bool
}

func (e *HTTPError) Error() string {
	if e.CFBlocked {
		return fmt.Sprintf("discord transport: %s %s: %d (Cloudflare bot challenge); body=%q",
			e.Method, e.Path, e.Status, truncate(e.Body, 200))
	}
	return fmt.Sprintf("discord transport: %s %s: %d; body=%q",
		e.Method, e.Path, e.Status, truncate(e.Body, 200))
}

// IsUnauthorized returns true for HTTP 401 — the token is wrong or
// has been invalidated (Discord rotates tokens on every password
// change and on certain abuse-detection events).
func IsUnauthorized(err error) bool {
	var he *HTTPError
	if !errors.As(err, &he) {
		return false
	}
	return he.Status == http.StatusUnauthorized
}

// IsCloudflareBlocked returns true when the request was rejected by
// Discord's Cloudflare layer (usually IP-level — VPN/datacenter).
func IsCloudflareBlocked(err error) bool {
	var he *HTTPError
	if !errors.As(err, &he) {
		return false
	}
	return he.CFBlocked
}

// IsNotFound returns true for HTTP 404.
func IsNotFound(err error) bool {
	var he *HTTPError
	if !errors.As(err, &he) {
		return false
	}
	return he.Status == http.StatusNotFound
}

// --- helpers ---

func parseRetryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil {
		return time.Duration(n * float64(time.Second))
	}
	return 0
}

func looksCloudflareBlocked(resp *http.Response, body []byte) bool {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
		return false
	}
	if h := resp.Header.Get("Server"); h == "cloudflare" {
		if bytes.Contains(body, []byte("Just a moment")) ||
			bytes.Contains(body, []byte("Attention Required")) ||
			bytes.Contains(body, []byte("cf-error-details")) {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

func orInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func orDuration(v, def time.Duration) time.Duration {
	if v <= 0 {
		return def
	}
	return v
}

// base64Std avoids an extra import in the public surface.
func base64Std(b []byte) string {
	return stdBase64Encode(b)
}
