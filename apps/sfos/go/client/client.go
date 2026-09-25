package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	UserAgent   = "okhttp/4.12.0"
	APIKey      = "8beb7c7a-726c-4fae-aa82-585e295b09e1"
	BuildNumber = "81804"
	APIVersion  = "3"
	ClientID    = "dop-mobile-app-v2"

	defaultAnonHost   = "https://prd-anonymous-gateway.dopapp.pl"
	defaultPublicHost = "https://prd-public-gateway.dopapp.pl"

	tokenExchangeGrant = "urn:ietf:params:oauth:grant-type:token-exchange"
	tokenJWTType       = "urn:ietf:params:oauth:token-type:jwt"
)

// Client talks to the Orange Flex gateways. Telemetry SDKs are not used.
type Client struct {
	http       *http.Client
	cacheDir   string
	lang       string
	anonHost   string
	publicHost string
	progressFn ProgressFunc

	mu       sync.Mutex
	session  Session
	lastDash *Dashboard
	lists    map[string][]map[string]any
	unread   []string

	pollInterval time.Duration
	pollTries    int
}

var networkAllowed atomic.Bool

// SetNetworkAllowed gates all HTTP. Network is denied until this is true.
func SetNetworkAllowed(allow bool) {
	networkAllowed.Store(allow)
}

// NetworkAllowed reports whether HTTP is permitted.
func NetworkAllowed() bool {
	return networkAllowed.Load()
}

// ProgressEvent is a status update from Client work.
type ProgressEvent struct {
	Step    string
	Message string
}

// ProgressFunc receives ProgressEvent values during Client work.
type ProgressFunc func(ProgressEvent)

// Option configures a Client.
type Option func(*Client)

// WithCacheDir overrides the on-disk cache directory.
func WithCacheDir(dir string) Option {
	return func(c *Client) { c.cacheDir = dir }
}

// WithHTTPClient overrides the HTTP client used for requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// WithProgress reports init and login steps.
func WithProgress(fn ProgressFunc) Option {
	return func(c *Client) { c.progressFn = fn }
}

// WithLanguage sets Accept-Language (en, pl, uk).
func WithLanguage(lang string) Option {
	return func(c *Client) { c.lang = lang }
}

// WithGateways overrides anonymous and public API base URLs (tests).
func WithGateways(anon, public string) Option {
	return func(c *Client) {
		c.anonHost = strings.TrimRight(anon, "/")
		c.publicHost = strings.TrimRight(public, "/")
	}
}

// New returns a Client. The default cache directory is os.UserCacheDir()/orangeflex.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		http:       &http.Client{Timeout: 45 * time.Second},
		lang:       "en",
		anonHost:   defaultAnonHost,
		publicHost: defaultPublicHost,
		lists:      map[string][]map[string]any{},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.cacheDir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("client: user cache dir: %w", err)
		}
		c.cacheDir = filepath.Join(base, "orangeflex")
	}
	if err := os.MkdirAll(c.cacheDir, 0750); err != nil {
		return nil, err
	}
	c.loadPersisted()
	return c, nil
}

// CacheDir is the on-disk cache root.
func (c *Client) CacheDir() string {
	return c.cacheDir
}

func (c *Client) report(step, message string) {
	if c == nil || c.progressFn == nil {
		return
	}
	c.progressFn(ProgressEvent{Step: step, Message: message})
}

func (c *Client) language() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lang == "" {
		return "en"
	}
	return c.lang
}

// SetLanguage persists UI/API language.
func (c *Client) SetLanguage(lang string) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang != "pl" && lang != "uk" {
		lang = "en"
	}
	c.mu.Lock()
	c.lang = lang
	c.mu.Unlock()
	_ = os.WriteFile(c.languagePath(), []byte(lang+"\n"), 0640)
}

// Language is the current Accept-Language.
func (c *Client) Language() string {
	return c.language()
}

func (c *Client) loadPersisted() {
	if b, err := os.ReadFile(c.languagePath()); err == nil {
		lang := strings.TrimSpace(string(b))
		if lang != "" {
			c.lang = lang
		}
	}
	sess, err := loadSession(c.sessionPath())
	if err == nil {
		c.session = sess
	}
	if c.session.DeviceID == "" {
		c.session.DeviceID = newUUID()
		_ = c.saveSession()
	}
}

func (c *Client) languagePath() string {
	return filepath.Join(c.cacheDir, "language")
}

func (c *Client) sessionPath() string {
	return filepath.Join(c.cacheDir, "session.json")
}

func (c *Client) networkFlagPath() string {
	return filepath.Join(c.cacheDir, "network.allowed")
}

// PersistNetwork stores the network-permission flag.
func (c *Client) PersistNetwork(allow bool) {
	val := "0\n"
	if allow {
		val = "1\n"
	}
	_ = os.WriteFile(c.networkFlagPath(), []byte(val), 0640)
}

// LoadNetworkFlag is true when the user previously allowed network.
func (c *Client) LoadNetworkFlag() bool {
	b, err := os.ReadFile(c.networkFlagPath())
	return err == nil && strings.TrimSpace(string(b)) == "1"
}

// NetworkPromptNeeded is true until the user has answered the prompt.
func (c *Client) NetworkPromptNeeded() bool {
	_, err := os.Stat(c.networkFlagPath())
	return err != nil
}

func (c *Client) newRequest(ctx context.Context, method, rawURL string, body io.Reader, contentType string, authed bool) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("apikey", APIKey)
	req.Header.Set("accept-language", c.language())
	req.Header.Set("x-build-number", BuildNumber)
	req.Header.Set("x-correlation-id", newUUID())
	c.mu.Lock()
	dev := c.session.DeviceID
	userID := c.session.UserID
	token := c.session.AccessToken
	c.mu.Unlock()
	if dev != "" {
		req.Header.Set("x-device-id", dev)
	}
	if authed {
		req.Header.Set("version", APIVersion)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if userID != "" {
			req.Header.Set("x-dop-user-id", userID)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if !NetworkAllowed() {
		return nil, fmt.Errorf("client: network permission not granted")
	}
	return c.http.Do(req)
}

func (c *Client) readJSON(resp *http.Response, dest any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiErr(resp.Request.URL.String(), resp.StatusCode, body)
	}
	if dest == nil || len(body) == 0 || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("client: decode %s: %w", resp.Request.URL.String(), err)
	}
	return nil
}

func (c *Client) getJSON(ctx context.Context, rawURL string, authed bool, dest any) error {
	req, err := c.newRequest(ctx, http.MethodGet, rawURL, nil, "", authed)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	if authed && resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.refreshTokens(ctx); err != nil {
			return err
		}
		req, err = c.newRequest(ctx, http.MethodGet, rawURL, nil, "", true)
		if err != nil {
			return err
		}
		resp, err = c.do(req)
		if err != nil {
			return err
		}
	}
	return c.readJSON(resp, dest)
}

func (c *Client) postJSON(ctx context.Context, rawURL string, authed bool, payload, dest any) error {
	body, err := c.postJSONRaw(ctx, rawURL, authed, payload)
	if err != nil {
		return err
	}
	if dest == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("client: decode %s: %w", rawURL, err)
	}
	return nil
}

func (c *Client) postJSONRaw(ctx context.Context, rawURL string, authed bool, payload any) ([]byte, error) {
	var rdr io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := c.newRequest(ctx, http.MethodPost, rawURL, rdr, "application/json", authed)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiErr(resp.Request.URL.String(), resp.StatusCode, body)
	}
	return body, nil
}

func (c *Client) postJSONWithHeaders(ctx context.Context, rawURL string, authed bool, payload, dest any, extra map[string]string) error {
	var rdr io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := c.newRequest(ctx, http.MethodPost, rawURL, rdr, "application/json", authed)
	if err != nil {
		return err
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	return c.readJSON(resp, dest)
}

func (c *Client) patchEmpty(ctx context.Context, rawURL string, authed bool) error {
	req, err := c.newRequest(ctx, http.MethodPatch, rawURL, nil, "", authed)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	return c.readJSON(resp, nil)
}

func (c *Client) postForm(ctx context.Context, rawURL string, authed bool, form url.Values, dest any) error {
	req, err := c.newRequest(ctx, http.MethodPost, rawURL, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", authed)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	return c.readJSON(resp, dest)
}

// WipeData deletes the cache directory contents (tokens, flags).
func (c *Client) WipeData() error {
	c.mu.Lock()
	c.session = Session{DeviceID: newUUID()}
	c.lastDash = nil
	c.lists = map[string][]map[string]any{}
	c.unread = nil
	c.mu.Unlock()
	if c.cacheDir == "" {
		return nil
	}
	if err := os.RemoveAll(c.cacheDir); err != nil {
		return err
	}
	if err := os.MkdirAll(c.cacheDir, 0750); err != nil {
		return err
	}
	return c.saveSession()
}

// DeviceID is the persisted client device id.
func (c *Client) DeviceID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session.DeviceID
}

// LoggedIn reports whether an access token is stored.
func (c *Client) LoggedIn() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session.AccessToken != ""
}

// UserID is the current dop user id.
func (c *Client) UserID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session.UserID
}
