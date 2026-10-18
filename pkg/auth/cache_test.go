package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadAuth(t *testing.T) {
	tempDir := t.TempDir()
	authFilePath := filepath.Join(tempDir, "subdir", "auth.json")

	now := time.Now().Truncate(time.Second)
	testAuth := &AuthData{
		TokenPollSuccess: TokenPollSuccess{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    3600,
		},
		Owner: "owner-uuid-1234",
		Profile: Profile{
			Uuid:     "profile-uuid-5678",
			Username: "Tester",
		},
		Session: Session{
			SessionToken:  "session-token-abc",
			IdentityToken: "identity-token-def",
			ExpiresAt:     now.Add(24 * time.Hour),
		},
	}

	err := testAuth.SaveTo(authFilePath)
	if err != nil {
		t.Fatalf("unexpected error saving auth: %v", err)
	}

	// Verify directory permissions
	dirInfo, err := os.Stat(filepath.Dir(authFilePath))
	if err != nil {
		t.Fatalf("failed to stat auth directory: %v", err)
	}
	if dirPerm := dirInfo.Mode().Perm(); dirPerm != 0700 {
		t.Errorf("expected directory permissions 0700, got %o", dirPerm)
	}

	// Verify file permissions
	fileInfo, err := os.Stat(authFilePath)
	if err != nil {
		t.Fatalf("failed to stat auth file: %v", err)
	}
	if filePerm := fileInfo.Mode().Perm(); filePerm != 0600 {
		t.Errorf("expected file permissions 0600, got %o", filePerm)
	}

	// Verify loading
	loaded, err := LoadAuthFromPath(authFilePath)
	if err != nil {
		t.Fatalf("unexpected error loading auth: %v", err)
	}

	if loaded.AccessToken != testAuth.AccessToken {
		t.Errorf("expected AccessToken %q, got %q", testAuth.AccessToken, loaded.AccessToken)
	}
	if loaded.RefreshToken != testAuth.RefreshToken {
		t.Errorf("expected RefreshToken %q, got %q", testAuth.RefreshToken, loaded.RefreshToken)
	}
	if loaded.Owner != testAuth.Owner {
		t.Errorf("expected Owner %q, got %q", testAuth.Owner, loaded.Owner)
	}
	if loaded.Profile.Uuid != testAuth.Profile.Uuid || loaded.Profile.Username != testAuth.Profile.Username {
		t.Errorf("expected Profile %+v, got %+v", testAuth.Profile, loaded.Profile)
	}
	if loaded.Session.SessionToken != testAuth.Session.SessionToken || loaded.Session.IdentityToken != testAuth.Session.IdentityToken {
		t.Errorf("expected Session %+v, got %+v", testAuth.Session, loaded.Session)
	}
	if !loaded.Valid() {
		t.Errorf("expected loaded auth data to be valid")
	}
}

func TestEnforcePermissionsOnLoad(t *testing.T) {
	tempDir := t.TempDir()
	authFilePath := filepath.Join(tempDir, "auth.json")

	// Create a file with overly permissive permissions (0644) matching hytale-downloader format
	content := []byte(`{"access_token":"token","refresh_token":"rtoken","owner":"owner","profile":{"uuid":"u","username":"un"},"session":{"sessionToken":"s","identityToken":"i"}}`)
	if err := os.WriteFile(authFilePath, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Load should enforce 0600 on the file
	loaded, err := LoadAuthFromPath(authFilePath)
	if err != nil {
		t.Fatalf("unexpected error loading auth: %v", err)
	}

	if loaded.AccessToken != "token" {
		t.Errorf("expected AccessToken 'token', got %q", loaded.AccessToken)
	}

	fileInfo, err := os.Stat(authFilePath)
	if err != nil {
		t.Fatalf("failed to stat auth file: %v", err)
	}
	if filePerm := fileInfo.Mode().Perm(); filePerm != 0600 {
		t.Errorf("expected permissions to be tightened to 0600, got %o", filePerm)
	}
}

func TestModelMethods(t *testing.T) {
	dc := &DeviceCode{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	if !dc.Expired() {
		t.Errorf("expected device code to be expired")
	}

	sess := &Session{
		SessionToken: "token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	if !sess.Valid() {
		t.Errorf("expected session to be valid")
	}
	if sess.Expired() {
		t.Errorf("expected session not to be expired")
	}
}

func TestParseJWTExpiry(t *testing.T) {
	// Sample JWT payload: {"exp": 1800000000, "sub": "test"} -> eyJleHAiOjE4MDAwMDAwMDAsInN1YiI6InRlc3QifQ
	fakeJWT := "eyJhbGciOiJIUzI1NiJ9.eyJleHAiOjE4MDAwMDAwMDAsInN1YiI6InRlc3QifQ.fake-signature"
	exp, err := parseJWTExpiry(fakeJWT)
	if err != nil {
		t.Fatalf("unexpected error parsing JWT expiry: %v", err)
	}
	expected := time.Unix(1800000000, 0)
	if !exp.Equal(expected) {
		t.Errorf("expected expiry %v, got %v", expected, exp)
	}

	sess := &Session{
		SessionToken:  "opaque-session-token",
		IdentityToken: fakeJWT,
	}
	if !sess.Valid() {
		t.Errorf("expected session with valid future exp in IdentityToken JWT to be valid")
	}
	if !sess.ExpiresAt.Equal(expected) {
		t.Errorf("expected session ExpiresAt to be populated from IdentityToken as %v, got %v", expected, sess.ExpiresAt)
	}
}

func TestDownloaderCredentialsSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	credsPath := filepath.Join(tempDir, "downloader", "downloader.json")

	now := time.Now().Truncate(time.Second)
	testCreds := &DownloaderCredentials{
		AccessToken:  "test-dl-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-dl-refresh-token",
		Expiry:       now.Add(1 * time.Hour),
		ExpiresIn:    3600,
	}

	if err := testCreds.SaveTo(credsPath); err != nil {
		t.Fatalf("unexpected error saving downloader credentials: %v", err)
	}

	// Verify permissions
	dirInfo, err := os.Stat(filepath.Dir(credsPath))
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}
	if dirPerm := dirInfo.Mode().Perm(); dirPerm != 0700 {
		t.Errorf("expected directory perm 0700, got %o", dirPerm)
	}

	fileInfo, err := os.Stat(credsPath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if filePerm := fileInfo.Mode().Perm(); filePerm != 0600 {
		t.Errorf("expected file perm 0600, got %o", filePerm)
	}

	loaded, err := LoadDownloaderAuth(credsPath)
	if err != nil {
		t.Fatalf("unexpected error loading downloader credentials: %v", err)
	}

	if loaded.AccessToken != testCreds.AccessToken {
		t.Errorf("expected AccessToken %q, got %q", testCreds.AccessToken, loaded.AccessToken)
	}
	if loaded.RefreshToken != testCreds.RefreshToken {
		t.Errorf("expected RefreshToken %q, got %q", testCreds.RefreshToken, loaded.RefreshToken)
	}
	if loaded.TokenType != testCreds.TokenType {
		t.Errorf("expected TokenType %q, got %q", testCreds.TokenType, loaded.TokenType)
	}
	if !loaded.Valid() {
		t.Errorf("expected loaded downloader credentials to be valid")
	}
}
