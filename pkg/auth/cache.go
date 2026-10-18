package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AuthData holds persisted OAuth token credentials, account identity, and active game session.
type AuthData struct {
	TokenPollSuccess
	Owner   OwnerUUID `json:"owner,omitempty"`
	Profile Profile   `json:"profile,omitempty"`
	Session Session   `json:"session,omitempty"`
}

// Valid returns true if the AuthData has an active game session and access token.
func (a *AuthData) Valid() bool {
	return a != nil && a.Session.Valid() && a.AccessToken != ""
}

// DefaultAuthFilePath returns the default path for cached auth tokens:
// ~/.config/tyhal/hytale/auth.json
func DefaultAuthFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "tyhal", "hytale", "auth.json"), nil
}

// Save writes authentication data to the default auth cache file.
func (a *AuthData) Save() error {
	path, err := DefaultAuthFilePath()
	if err != nil {
		return err
	}
	return a.SaveTo(path)
}

// saveJSONAtomic atomically writes any JSON-serializable value to path with 0600 permissions
// and ensures directory permissions are restricted to owner only (0700).
func saveJSONAtomic(path string, data any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode json data: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".auth-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := tmpFile.Chmod(0600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to set permissions on %s: %w", tmpPath, err)
	}

	if _, err := tmpFile.Write(jsonData); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write json data: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to sync json data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace file %s: %w", path, err)
	}

	return nil
}

// SaveTo atomically writes authentication data to a specific file path with 0600 permissions
// and ensures directory permissions are restricted to owner only (0700).
func (a *AuthData) SaveTo(path string) error {
	return saveJSONAtomic(path, a)
}

// LoadAuth loads authentication data from the default cache file.
func LoadAuth() (*AuthData, error) {
	path, err := DefaultAuthFilePath()
	if err != nil {
		return nil, err
	}
	return LoadAuthFromPath(path)
}

// LoadAuthFromPath loads authentication data from a specific file path.
func LoadAuthFromPath(path string) (*AuthData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0600)

	var authData AuthData
	if err := json.Unmarshal(data, &authData); err != nil {
		return nil, fmt.Errorf("failed to decode auth file %s: %w", path, err)
	}
	return &authData, nil
}

// initSession retrieves account profiles, provisions a game session, and saves the cache.
func (a *AuthData) initSession() error {
	if a.Profile.Uuid == "" {
		profiles, err := a.AccessToken.GetProfiles()
		if err != nil || len(profiles.Profiles) == 0 {
			return fmt.Errorf("no profiles found for the authenticated account")
		}
		a.Owner = profiles.Owner
		a.Profile = profiles.Profiles[0]
	}
	session, err := a.AccessToken.CreateGameSession(a.Profile)
	if err != nil || !session.Valid() {
		return fmt.Errorf("failed to create game session: %w", err)
	}
	a.Session = session
	return a.Save()
}

// Authenticate performs the interactive OAuth device code flow, retrieves account profiles,
// creates a game session, persists the auth cache to disk with 0600 permissions, and returns AuthData.
func Authenticate() (*AuthData, error) {
	deviceCode, err := RequestDeviceCode(ServerClientID, ServerScope)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}

	fmt.Printf("Auth for Game Session\nAuthenticate your Hytale account by opening:\n  %s\n\nWaiting for confirmation...\n", deviceCode.VerificationUriComplete)

	token, err := deviceCode.WaitForToken()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for token: %w", err)
	}

	authData := &AuthData{TokenPollSuccess: token}
	if err := authData.initSession(); err != nil {
		return nil, err
	}
	return authData, nil
}

// DownloaderCredentials holds persisted OAuth token credentials for the official downloader binary.
type DownloaderCredentials struct {
	AccessToken  AccessToken  `json:"access_token"`
	TokenType    string       `json:"token_type,omitempty"`
	RefreshToken RefreshToken `json:"refresh_token,omitempty"`
	Expiry       time.Time    `json:"expiry,omitempty"`
	ExpiresIn    int          `json:"expires_in,omitempty"`
}

// Valid returns true if the downloader credentials have a valid, unexpired token.
func (d *DownloaderCredentials) Valid() bool {
	if d == nil || d.AccessToken == "" {
		return false
	}
	if d.Expiry.IsZero() {
		return true
	}
	return time.Now().Before(d.Expiry)
}

// DefaultDownloaderAuthFilePath returns the default path for downloader credentials:
// ~/.config/tyhal/hytale/downloader.json
func DefaultDownloaderAuthFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "tyhal", "hytale", "downloader.json"), nil
}

// Save writes downloader credentials to the default cache file.
func (d *DownloaderCredentials) Save() error {
	path, err := DefaultDownloaderAuthFilePath()
	if err != nil {
		return err
	}
	return d.SaveTo(path)
}

// SaveTo atomically writes downloader credentials to a specific file path with 0600 permissions.
func (d *DownloaderCredentials) SaveTo(path string) error {
	return saveJSONAtomic(path, d)
}

// LoadDownloaderAuth loads downloader credentials from a specific file path (or default path if empty).
func LoadDownloaderAuth(path string) (*DownloaderCredentials, error) {
	if path == "" {
		var err error
		path, err = DefaultDownloaderAuthFilePath()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0600)

	var creds DownloaderCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to decode downloader credentials %s: %w", path, err)
	}
	return &creds, nil
}

// AuthenticateDownloader performs the interactive OAuth device code flow for the downloader client,
// saves credentials to disk with 0600 permissions, and returns DownloaderCredentials.
func AuthenticateDownloader(path string) (*DownloaderCredentials, error) {
	if path == "" {
		var err error
		path, err = DefaultDownloaderAuthFilePath()
		if err != nil {
			return nil, err
		}
	}

	deviceCode, err := RequestDeviceCode(DownloaderClientID, DownloaderScope)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}

	fmt.Printf("Auth for Game Downloader\nAuthenticate your Hytale account by opening:\n  %s\n\nWaiting for confirmation...\n", deviceCode.VerificationUriComplete)

	token, err := deviceCode.WaitForToken()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for token: %w", err)
	}

	creds := &DownloaderCredentials{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.ExpiresIn,
	}
	if token.ExpiresIn > 0 {
		creds.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	if err := creds.SaveTo(path); err != nil {
		return nil, err
	}
	return creds, nil
}

// EnsureDownloaderAuth returns valid DownloaderCredentials from cache if available,
// automatically refreshes them if expired, or falls back to interactive device code authentication.
func EnsureDownloaderAuth(path string) (*DownloaderCredentials, error) {
	if path == "" {
		var err error
		path, err = DefaultDownloaderAuthFilePath()
		if err != nil {
			return nil, err
		}
	}

	if cached, err := LoadDownloaderAuth(path); err == nil && cached != nil {
		if cached.Valid() {
			return cached, nil
		}
		if cached.RefreshToken != "" {
			if refreshed, err := cached.RefreshToken.Refresh(DownloaderClientID); err == nil {
				cached.AccessToken = refreshed.AccessToken
				cached.RefreshToken = refreshed.RefreshToken
				cached.TokenType = refreshed.TokenType
				cached.ExpiresIn = refreshed.ExpiresIn
				if refreshed.ExpiresIn > 0 {
					cached.Expiry = time.Now().Add(time.Duration(refreshed.ExpiresIn) * time.Second)
				}
				_ = cached.SaveTo(path)
				return cached, nil
			}
		}
	}

	return AuthenticateDownloader(path)
}

// EnsureAuth returns valid AuthData from the local cache if available and active,
// refreshes the session or OAuth tokens if expired, or falls back to interactive device code authentication.
func EnsureAuth() (*AuthData, error) {
	if cached, err := LoadAuth(); err == nil && cached != nil {
		if cached.Session.ExpiresAt.IsZero() {
			if exp, err := parseSessionExpiry(&cached.Session); err == nil {
				cached.Session.ExpiresAt = exp
			}
		}

		// 1. Check if cached session is still active (with 2-minute safety buffer if expiry is known)
		if cached.Session.Valid() {
			if cached.Session.ExpiresAt.IsZero() || time.Now().Add(2*time.Minute).Before(cached.Session.ExpiresAt) {
				return cached, nil
			}
		}

		// 2. Try direct session refresh
		if cached.Session.SessionToken != "" && cached.Session.Refresh() == nil && cached.Session.Valid() {
			_ = cached.Save()
			return cached, nil
		}

		// 3. Try creating a new game session using current access token
		if cached.AccessToken != "" && cached.initSession() == nil {
			return cached, nil
		}

		// 4. Try refreshing OAuth access token
		if cached.RefreshToken != "" {
			if refreshed, err := cached.RefreshToken.Refresh(); err == nil {
				cached.TokenPollSuccess = *refreshed
				if cached.initSession() == nil {
					return cached, nil
				}
			}
		}
	}

	return Authenticate()
}
