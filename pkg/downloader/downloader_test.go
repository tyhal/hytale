package downloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDownloaderDefaultsAndOptions(t *testing.T) {
	d, err := New()
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	dl, ok := d.(*downloader)
	if !ok {
		t.Fatalf("expected *downloader instance")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("unexpected error getting user config dir: %v", err)
	}
	expectedCredsPath := filepath.Join(configDir, "tyhal", "hytale", "downloader.json")

	if dl.credentialsPath != expectedCredsPath {
		t.Errorf("expected credentialsPath %q, got %q", expectedCredsPath, dl.credentialsPath)
	}

	// Test custom credentials path
	dCustom, err := New(WithCredentialsPath("/custom/path.json"), WithAllowAOTFail())
	if err != nil {
		t.Fatalf("unexpected error creating custom downloader: %v", err)
	}
	dlCustom := dCustom.(*downloader)
	if dlCustom.credentialsPath != "/custom/path.json" {
		t.Errorf("expected credentialsPath %q, got %q", "/custom/path.json", dlCustom.credentialsPath)
	}
	if !dlCustom.allowAOTFail {
		t.Errorf("expected allowAOTFail to be true")
	}
}
