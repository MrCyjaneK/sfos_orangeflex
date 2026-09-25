package client

import (
	"encoding/json"
	"os"
	"time"
)

// Session is persisted OAuth + identity state.
type Session struct {
	DeviceID         string    `json:"deviceId"`
	DeviceIDToken    string    `json:"deviceIdToken,omitempty"`
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	TokenExpiry      time.Time `json:"tokenExpiry"`
	RefreshExpiry    time.Time `json:"refreshExpiry"`
	UserID           string    `json:"userId"`
	PendingTID       string    `json:"pendingTid,omitempty"`
	PendingMSISDN    string    `json:"pendingMsisdn,omitempty"`
	PendingEmailMask string    `json:"pendingEmailMask,omitempty"`
}

func loadSession(path string) (Session, error) {
	var s Session
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	return s, nil
}

func (c *Client) saveSession() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, err := json.MarshalIndent(c.session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.sessionPath(), b, 0600)
}

func (c *Client) setPending(tid, msisdn, mask string) {
	c.mu.Lock()
	c.session.PendingTID = tid
	c.session.PendingMSISDN = msisdn
	c.session.PendingEmailMask = mask
	c.mu.Unlock()
	_ = c.saveSession()
}

func (c *Client) applyTokens(tr TokenResponse) {
	now := time.Now()
	c.mu.Lock()
	c.session.AccessToken = tr.AccessToken
	if tr.RefreshToken != "" {
		c.session.RefreshToken = tr.RefreshToken
	}
	if tr.ExpiresIn > 0 {
		c.session.TokenExpiry = now.Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	if tr.RefreshExpiresIn > 0 {
		c.session.RefreshExpiry = now.Add(time.Duration(tr.RefreshExpiresIn) * time.Second)
	}
	c.session.PendingTID = ""
	c.session.PendingEmailMask = ""
	c.session.PendingMSISDN = ""
	c.mu.Unlock()
	_ = c.saveSession()
}

// PendingTID is the current OTP ticket.
func (c *Client) PendingTID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session.PendingTID
}

// PendingEmailMask is the masked email shown for 2FA, if any.
func (c *Client) PendingEmailMask() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session.PendingEmailMask
}

// Logout clears tokens but keeps device id and language.
func (c *Client) Logout() error {
	c.mu.Lock()
	dev := c.session.DeviceID
	devTok := c.session.DeviceIDToken
	c.session = Session{DeviceID: dev, DeviceIDToken: devTok}
	c.lastDash = nil
	c.lists = map[string][]map[string]any{}
	c.unread = nil
	c.mu.Unlock()
	return c.saveSession()
}
