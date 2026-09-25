package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// StartResult is returned by LoginStart.
type StartResult struct {
	TID       string
	Timeout   int
	ExpiresAt string
}

// OTPResult is returned by LoginSubmitOTP.
type OTPResult struct {
	LoggedIn    bool
	NeedNextOTP bool
	RestartSMS  bool
	MaskedEmail string
	Timeout     int
}

// TokenResponse is an OAuth token payload from POST /proxy/token.
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	IssuedTokenType  string `json:"issued_token_type"`
}

type actionTokenResponse struct {
	TID           string `json:"tid"`
	MaskedEmail   string `json:"masked_email"`
	ActionToken   string `json:"action_token"`
	DeviceIDToken string `json:"device_id_token"`
	Timeout       int    `json:"timeout"`
}

type startResponse struct {
	TID       string `json:"tid"`
	Timeout   int    `json:"timeout"`
	ExpiresAt string `json:"expiresAt"`
}

func looksJWT(s string) bool {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	return strings.HasPrefix(s, "eyJ") && len(parts) >= 3
}

// LoginStart sends an SMS OTP to msisdn.
// Captured: POST {to, deviceId} → {tid, timeout, expiresAt}.
func (c *Client) LoginStart(ctx context.Context, to string) (*StartResult, error) {
	to = strings.TrimSpace(to)
	if to == "" {
		return nil, fmt.Errorf("client: empty login identity")
	}
	c.report("login", "Sending one-time code")
	var out startResponse
	err := c.postJSON(ctx, c.anonHost+"/verification/start", false, map[string]string{
		"to":       to,
		"deviceId": c.DeviceID(),
	}, &out)
	if err != nil {
		return nil, err
	}
	if out.TID == "" {
		return nil, fmt.Errorf("client: verification start returned no ticket")
	}
	c.setPending(out.TID, to, "")
	return &StartResult{TID: out.TID, Timeout: out.Timeout, ExpiresAt: out.ExpiresAt}, nil
}

// LoginSubmitOTP consumes the current ticket + OTP.
//
// Capture:
//   - SMS success: {tid, masked_email} then POST /verification/start {to: tid, channel: EMAIL, deviceId}
//   - Login success: {action_token, device_id_token?, tid} then POST /proxy/token with action_token
//   - proxy/token 400 "User already exists" (registration action_token): keep device_id_token, new SMS start
func (c *Client) LoginSubmitOTP(ctx context.Context, otp string) (*OTPResult, error) {
	otp = strings.TrimSpace(otp)
	if otp == "" {
		return nil, fmt.Errorf("client: empty OTP")
	}
	tid := c.PendingTID()
	if tid == "" {
		return nil, fmt.Errorf("client: no pending OTP ticket")
	}
	c.report("login", "Checking one-time code")
	payload := map[string]string{
		"tid": tid,
		"otp": otp,
	}
	if tok := c.deviceIDToken(); tok != "" {
		payload["device_id_token"] = tok
	}
	var out actionTokenResponse
	err := c.postJSON(ctx, c.anonHost+"/verification/action-token", false, payload, &out)
	if err != nil {
		return nil, err
	}

	if out.DeviceIDToken != "" {
		c.setDeviceIDToken(out.DeviceIDToken)
	}

	if looksJWT(out.ActionToken) {
		// action-token consumed this OTP. Confirm must not resubmit it
		// (capture: same tid+otp after a 200 is HTTP 403 Invalid OTP).
		msisdn := c.sessionMSISDN()
		c.setPending("", msisdn, "")
		if err := c.exchangeToken(ctx, out.ActionToken); err != nil {
			return c.afterFailedExchange(ctx, msisdn, err)
		}
		return &OTPResult{LoggedIn: true}, nil
	}

	mask := strings.TrimSpace(out.MaskedEmail)
	if out.TID != "" && mask != "" {
		msisdn := c.sessionMSISDN()
		c.setPending(out.TID, msisdn, mask)
		emailStart, err := c.startEmailOTP(ctx, out.TID)
		if err != nil {
			return nil, err
		}
		c.setPending(emailStart.TID, msisdn, mask)
		return &OTPResult{NeedNextOTP: true, MaskedEmail: mask, Timeout: emailStart.Timeout}, nil
	}

	return nil, fmt.Errorf("client: OTP accepted but no action_token was returned")
}

func (c *Client) startEmailOTP(ctx context.Context, smsTID string) (*StartResult, error) {
	c.report("login", "Sending email code")
	var out startResponse
	err := c.postJSON(ctx, c.anonHost+"/verification/start", false, map[string]string{
		"to":       smsTID,
		"channel":  "EMAIL",
		"deviceId": c.DeviceID(),
	}, &out)
	if err != nil {
		return nil, err
	}
	if out.TID == "" {
		return nil, fmt.Errorf("client: email verification start returned no ticket")
	}
	return &StartResult{TID: out.TID, Timeout: out.Timeout, ExpiresAt: out.ExpiresAt}, nil
}

func (c *Client) sessionMSISDN() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session.PendingMSISDN
}

// PendingMSISDN is the number the current OTP was sent to.
func (c *Client) PendingMSISDN() string {
	return c.sessionMSISDN()
}

func (c *Client) deviceIDToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session.DeviceIDToken
}

func (c *Client) setDeviceIDToken(tok string) {
	c.mu.Lock()
	c.session.DeviceIDToken = tok
	c.mu.Unlock()
	_ = c.saveSession()
}

func isUserAlreadyExists(err error) bool {
	ae, ok := err.(*APIError)
	return ok && ae.Status == http.StatusBadRequest && ae.Message == "User already exists"
}

// afterFailedExchange follows the capture after proxy/token 400
// "User already exists": keep device_id_token, start a new SMS ticket.
// Android then submitted the new SMS OTP with device_id_token and exchanged
// an action_token whose iss was auto-migrations-otp (that call returned 200).
func (c *Client) afterFailedExchange(ctx context.Context, msisdn string, exchangeErr error) (*OTPResult, error) {
	if !isUserAlreadyExists(exchangeErr) || c.deviceIDToken() == "" || msisdn == "" {
		return nil, exchangeErr
	}
	st, err := c.LoginStart(ctx, msisdn)
	if err != nil {
		return nil, exchangeErr
	}
	return &OTPResult{RestartSMS: true, Timeout: st.Timeout}, nil
}

func (c *Client) exchangeToken(ctx context.Context, actionToken string) error {
	c.report("login", "Opening session")
	form := url.Values{}
	form.Set("client_id", ClientID)
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", actionToken)
	form.Set("subject_token_type", tokenJWTType)
	tr, err := c.doTokenExchange(ctx, form)
	if err != nil {
		return err
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("client: token exchange returned no access_token")
	}
	c.applyTokens(*tr)
	return nil
}

func (c *Client) doTokenExchange(ctx context.Context, form url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.anonHost+"/proxy/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("apikey", APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var tr TokenResponse
	if err := c.readJSON(resp, &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func (c *Client) refreshTokens(ctx context.Context) error {
	c.mu.Lock()
	rt := c.session.RefreshToken
	c.mu.Unlock()
	if rt == "" {
		return fmt.Errorf("client: not authenticated")
	}
	c.report("session", "Refreshing session")
	form := url.Values{}
	form.Set("client_id", ClientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", rt)
	var tr TokenResponse
	if err := c.postForm(ctx, c.anonHost+"/proxy/token", false, form, &tr); err != nil {
		return err
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("client: refresh returned no access_token")
	}
	c.applyTokens(tr)
	return nil
}

// DecodeActionToken is used in tests.
func DecodeActionToken(b []byte) (actionTokenResponse, error) {
	var out actionTokenResponse
	err := json.Unmarshal(b, &out)
	return out, err
}
