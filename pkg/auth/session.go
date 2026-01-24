package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const sessionUrl = "https://sessions.hytale.com"

type SessionToken = string
type IdentityToken = string

type Session struct {
	SessionToken  SessionToken  `json:"sessionToken"`
	IdentityToken IdentityToken `json:"identityToken"`
	ExpiresAt     time.Time     `json:"expiresAt"`
}

func CreateGameSession(at AccessToken, p Profile) (Session, error) {
	payload := map[string]string{"uuid": p.Uuid}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return Session{}, err
	}
	req, err := http.NewRequest("POST", sessionUrl+"/game-session/new", bytes.NewBuffer(jsonData))
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+at)
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

func (s *Session) Refresh() error {
	req, err := http.NewRequest("POST", sessionUrl+"/game-session/refresh", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.SessionToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Session) Terminate() error {
	req, err := http.NewRequest("DELETE", sessionUrl+"/game-session", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.SessionToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
