package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authURL    = "https://oauth.accounts.hytale.com/oauth2"
	accountURL = "https://account-data.hytale.com"
	sessionURL = "https://sessions.hytale.com"

	ServerClientID     = "hytale-server"
	ServerScope        = "openid offline auth:server"
	DownloaderClientID = "hytale-downloader"
	DownloaderScope    = "openid offline auth:downloader"
)

// Domain types & aliases
type (
	UUID          = string
	OwnerUUID     = UUID
	SessionToken  = string
	IdentityToken = string
	AccessToken   string
	RefreshToken  string
)

type Profile struct {
	Uuid     UUID   `json:"uuid"`
	Username string `json:"username"`
}

type OwnedProfiles struct {
	Owner    OwnerUUID `json:"owner"`
	Profiles []Profile `json:"profiles"`
}

type DeviceCode struct {
	ClientID                string    `json:"-"`
	DeviceCode              string    `json:"device_code"`
	UserCode                string    `json:"user_code"`
	VerificationUri         string    `json:"verification_uri"`
	VerificationUriComplete string    `json:"verification_uri_complete"`
	ExpiresIn               int       `json:"expires_in"`
	ExpiresAt               time.Time `json:"-"`
	Interval                int       `json:"interval"`
}

func (dc *DeviceCode) Expired() bool {
	return !dc.ExpiresAt.IsZero() && time.Now().After(dc.ExpiresAt)
}

type TokenPollSuccess struct {
	AccessToken  AccessToken  `json:"access_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	RefreshToken RefreshToken `json:"refresh_token"`
	Scope        string       `json:"scope"`
}

type Session struct {
	SessionToken  SessionToken  `json:"sessionToken"`
	IdentityToken IdentityToken `json:"identityToken"`
	ExpiresAt     time.Time     `json:"expiresAt"`
}

func (s *Session) Valid() bool {
	if s == nil || s.SessionToken == "" {
		return false
	}
	if s.ExpiresAt.IsZero() {
		if exp, err := parseSessionExpiry(s); err == nil {
			s.ExpiresAt = exp
		}
	}
	if s.ExpiresAt.IsZero() {
		return true
	}
	return time.Now().Before(s.ExpiresAt)
}

func parseSessionExpiry(s *Session) (time.Time, error) {
	if s == nil {
		return time.Time{}, fmt.Errorf("session is nil")
	}
	if s.IdentityToken != "" {
		if exp, err := parseJWTExpiry(s.IdentityToken); err == nil {
			return exp, nil
		}
	}
	if s.SessionToken != "" {
		if exp, err := parseJWTExpiry(s.SessionToken); err == nil {
			return exp, nil
		}
	}
	return time.Time{}, fmt.Errorf("no jwt expiry found")
}

func (s *Session) Expired() bool {
	return !s.Valid()
}

// HTTP API Helpers

func apiReq(method, rawURL, bearerToken string, reqBody, target any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}

func postForm(endpoint string, data url.Values, target any) (int, error) {
	resp, err := http.PostForm(authURL+endpoint, data)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK && target != nil {
		return resp.StatusCode, json.NewDecoder(resp.Body).Decode(target)
	}
	return resp.StatusCode, nil
}

func parseJWTExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid jwt format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		if payload, err = base64.URLEncoding.DecodeString(parts[1]); err != nil {
			return time.Time{}, err
		}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("no valid exp claim in jwt")
	}
	return time.Unix(claims.Exp, 0), nil
}

// OAuth & Session Operations

func RequestDeviceCode(clientID, scope string) (DeviceCode, error) {
	if clientID == "" {
		clientID = ServerClientID
	}
	if scope == "" {
		scope = ServerScope
	}
	var device DeviceCode
	status, err := postForm("/device/auth", url.Values{
		"client_id": {clientID},
		"scope":     {scope},
	}, &device)
	if err != nil {
		return device, err
	}
	if status != http.StatusOK {
		return device, fmt.Errorf("unexpected status code: %d", status)
	}
	device.ClientID = clientID
	device.ExpiresAt = time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)
	return device, nil
}

func (dc DeviceCode) WaitForToken() (TokenPollSuccess, error) {
	clientID := dc.ClientID
	if clientID == "" {
		clientID = ServerClientID
	}
	interval := dc.Interval
	if interval <= 0 {
		interval = 5
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(time.Until(dc.ExpiresAt))
	defer timeout.Stop()

	data := url.Values{
		"client_id":   {clientID},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {dc.DeviceCode},
	}
	for {
		select {
		case <-ticker.C:
			var token TokenPollSuccess
			status, err := postForm("/token", data, &token)
			if err != nil {
				return TokenPollSuccess{}, err
			}
			if status == http.StatusOK {
				return token, nil
			}
		case <-timeout.C:
			return TokenPollSuccess{}, fmt.Errorf("device code expired")
		}
	}
}

func (rt RefreshToken) Refresh(clientID ...string) (*TokenPollSuccess, error) {
	cid := ServerClientID
	if len(clientID) > 0 && clientID[0] != "" {
		cid = clientID[0]
	}
	var token TokenPollSuccess
	status, err := postForm("/token", url.Values{
		"client_id":     {cid},
		"grant_type":    {"refresh_token"},
		"refresh_token": {string(rt)},
	}, &token)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", status)
	}
	return &token, nil
}

func (at AccessToken) GetProfiles() (OwnedProfiles, error) {
	var p OwnedProfiles
	err := apiReq(http.MethodGet, accountURL+"/my-account/get-profiles", string(at), nil, &p)
	return p, err
}

func (at AccessToken) CreateGameSession(p Profile) (Session, error) {
	var s Session
	err := apiReq(http.MethodPost, sessionURL+"/game-session/new", string(at), map[string]string{"uuid": p.Uuid}, &s)
	if err == nil {
		if exp, err := parseSessionExpiry(&s); err == nil {
			s.ExpiresAt = exp
		}
	}
	return s, err
}

func (s *Session) Refresh() error {
	if s == nil || s.SessionToken == "" {
		return fmt.Errorf("session token is empty")
	}
	var refreshed Session
	if err := apiReq(http.MethodPost, sessionURL+"/game-session/refresh", s.SessionToken, nil, &refreshed); err != nil {
		return err
	}
	if exp, err := parseSessionExpiry(&refreshed); err == nil {
		refreshed.ExpiresAt = exp
	}
	*s = refreshed
	return nil
}

func (s *Session) Terminate() error {
	if s == nil || s.SessionToken == "" {
		return fmt.Errorf("session token is empty")
	}
	return apiReq(http.MethodDelete, sessionURL+"/game-session", s.SessionToken, nil, nil)
}
