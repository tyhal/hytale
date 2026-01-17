package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

var authUrl = "https://oauth.accounts.hytale.com/oauth2"
var accountUrl = "https://account-data.hytale.com"
var sessionUrl = "https://sessions.hytale.com"

type DeviceCode struct {
	DeviceCode              string    `json:"device_code"`
	UserCode                string    `json:"user_code"`
	VerificationUri         string    `json:"verification_uri"`
	VerificationUriComplete string    `json:"verification_uri_complete"`
	ExpiresIn               int       `json:"expires_in"`
	ExpiresAt               time.Time `json:"-"`
	Interval                int       `json:"interval"`
}

func RequestDeviceCode() (DeviceCode, error) {
	data := url.Values{}
	data.Set("client_id", "hytale-server")
	data.Set("scope", "openid offline auth:server")
	resp, err := http.PostForm(authUrl+"/device/auth", data)
	if err != nil {
		return DeviceCode{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return DeviceCode{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var device DeviceCode
	err = json.NewDecoder(resp.Body).Decode(&device)
	if err != nil {
		return DeviceCode{}, err
	}
	device.ExpiresAt = time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)
	return device, nil
}

type TokenPollError struct {
	Error string `json:"error"`
}

type AccessToken = string

type TokenPollSuccess struct {
	AccessToken  AccessToken `json:"access_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"`
	RefreshToken string      `json:"refresh_token"`
	Scope        string      `json:"scope"`
}

func pollToken(data url.Values) (*TokenPollSuccess, error) {
	resp, err := http.PostForm(authUrl+"/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 400 {
			var pollErr TokenPollError
			json.NewDecoder(resp.Body).Decode(&pollErr)
			log.Println(pollErr.Error)
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var token TokenPollSuccess
	err = json.NewDecoder(resp.Body).Decode(&token)
	return &token, nil
}

func WaitForToken(dc DeviceCode) (TokenPollSuccess, error) {
	ticker := time.NewTicker(time.Duration(dc.Interval) * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(time.Until(dc.ExpiresAt))
	defer timeout.Stop()

	data := url.Values{}
	data.Set("client_id", "hytale-server")
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", dc.DeviceCode)
	for {
		select {
		case <-ticker.C:
			succ, err := pollToken(data)
			if err != nil {
				return TokenPollSuccess{}, err
			}
			if succ != nil {
				return *succ, nil
			}
		case <-timeout.C:
			// Handle timeout
			return TokenPollSuccess{}, fmt.Errorf("device code expired")
		}
	}
}

type UUID = string

type Profile struct {
	Uuid     UUID   `json:"uuid"`
	Username string `json:"username"`
}

type OwnerUUID = UUID

type Profiles struct {
	Owner    OwnerUUID `json:"owner"`
	Profiles []Profile `json:"profiles"`
}

func GetProfiles(accessToken AccessToken) (Profiles, error) {
	req, err := http.NewRequest("GET", accountUrl+"/my-account/get-profiles", nil)
	if err != nil {
		return Profiles{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Profiles{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Profiles{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var profiles Profiles
	err = json.NewDecoder(resp.Body).Decode(&profiles)
	if err != nil {
		return Profiles{}, err
	}
	return profiles, nil
}

type SessionToken = string
type IdentityToken = string

type Session struct {
	SessionToken  string    `json:"sessionToken"`
	IdentityToken string    `json:"identityToken"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

func CreateGameSession(accessToken AccessToken, profile Profile) (Session, error) {
	payload := map[string]string{"uuid": profile.Uuid}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return Session{}, err
	}
	req, err := http.NewRequest("POST", sessionUrl+"/game-session/new", bytes.NewBuffer(jsonData))
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Session{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Session{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var session Session
	err = json.NewDecoder(resp.Body).Decode(&session)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}
