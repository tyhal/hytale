package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

const authUrl = "https://oauth.accounts.hytale.com/oauth2"

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
type RefreshToken = string
type TokenPollSuccess struct {
	AccessToken  AccessToken  `json:"access_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	RefreshToken RefreshToken `json:"refresh_token"`
	Scope        string       `json:"scope"`
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
