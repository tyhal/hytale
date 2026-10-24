package server

import (
	"strings"
	"testing"

	"github.com/tyhal/hytale/pkg/auth"
	"github.com/tyhal/hytale/pkg/downloader"
)

func TestBuildJavaArgs(t *testing.T) {
	opts := serverOptions{
		backups:      "/mnt/backups",
		sessionToken: auth.SessionToken("session123"),
		idToken:      auth.IdentityToken("id123"),
		owner:        auth.OwnerUUID("owner123"),
		ipv6:         true,
		javaFlags:    []string{"-Xms4G", "-Xmx4G"},
	}

	args := buildJavaArgs(
		downloader.GameJarPath("/path/to/HytaleServer.jar"),
		downloader.GameAssetsPath("/path/to/Assets.zip"),
		downloader.GameAotPath(""),
		opts,
	)

	argsStr := strings.Join(args, " ")

	expectedSubstrings := []string{
		"-XX:MaxRAMPercentage=80.0",
		"-XX:+UseG1GC",
		"-XX:+UseStringDeduplication",
		"-Xms4G",
		"-Xmx4G",
		"-Djava.net.preferIPv6Addresses=true",
		"-jar /path/to/HytaleServer.jar",
		"--assets /path/to/Assets.zip",
		"--backup --backup-dir /mnt/backups",
		"--session-token session123",
		"--identity-token id123",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(argsStr, substr) {
			t.Errorf("expected %q in args, got: %s", substr, argsStr)
		}
	}
}

func TestRenderUnitFile(t *testing.T) {
	workingDir := "/var/hytale/world"
	execStart := "/usr/bin/java -jar /opt/hytale/HytaleServer.jar"

	rendered, err := renderUnitFile(workingDir, execStart)
	if err != nil {
		t.Fatalf("unexpected error rendering unit file: %v", err)
	}

	if !strings.Contains(rendered, "WorkingDirectory=/var/hytale/world") {
		t.Errorf("missing WorkingDirectory in rendered unit: %s", rendered)
	}

	if !strings.Contains(rendered, "ExecStart=/bin/sh -c 'mkdir -p /run/hytale && [ -p /run/hytale/stdin.fifo ] || mkfifo /run/hytale/stdin.fifo; exec /usr/bin/java -jar /opt/hytale/HytaleServer.jar 0<> /run/hytale/stdin.fifo'") {
		t.Errorf("missing ExecStart in rendered unit: %s", rendered)
	}

	if !strings.Contains(rendered, "RuntimeDirectory=hytale") {
		t.Errorf("missing RuntimeDirectory in rendered unit: %s", rendered)
	}

	if !strings.Contains(rendered, "KillSignal=SIGTERM") || !strings.Contains(rendered, "TimeoutStopSec=120") {
		t.Errorf("missing graceful shutdown parameters in rendered unit: %s", rendered)
	}

	if !strings.Contains(rendered, "ExecStop=/usr/local/bin/hytale-stop.sh") {
		t.Errorf("missing ExecStop in rendered unit: %s", rendered)
	}

	if !strings.Contains(rendered, "Restart=on-failure") {
		t.Errorf("missing Restart=on-failure in rendered unit: %s", rendered)
	}

	if !strings.Contains(rendered, "[Unit]") || !strings.Contains(rendered, "[Service]") || !strings.Contains(rendered, "[Install]") {
		t.Errorf("missing standard systemd sections in rendered unit: %s", rendered)
	}
}

func TestStopScriptContent(t *testing.T) {
	expectedSubstrings := []string{
		"say [Server] Server shutting down in 60 seconds...",
		"say [Server] Server shutting down in 30 seconds...",
		"say [Server] Server shutting down in 10 seconds...",
		"say [Server] World backup initiated...",
		"backup",
		"stop",
	}

	lastIdx := -1
	for _, substr := range expectedSubstrings {
		idx := strings.Index(stopScriptContent, substr)
		if idx == -1 {
			t.Errorf("expected %q in stopScriptContent, got: %s", substr, stopScriptContent)
		} else if idx < lastIdx {
			t.Errorf("expected %q to appear after previous substring in stopScriptContent", substr)
		}
		lastIdx = idx
	}
}

func TestOptions(t *testing.T) {
	opts := serverOptions{}
	optionsList := []Options{
		WithBackups("/backup/dir"),
		WithSessionToken(auth.SessionToken("sess")),
		WithIdentityToken(auth.IdentityToken("ident")),
		WithOwner(auth.OwnerUUID("owner-id")),
		WithIPv6(),
		WithJavaFlags("-XX:+PrintGC"),
		WithExitOnOOM(),
		WithExec(),
		WithService(),
	}

	for _, opt := range optionsList {
		opt(&opts)
	}

	if opts.backups != "/backup/dir" {
		t.Errorf("expected backups /backup/dir, got %s", opts.backups)
	}
	if opts.sessionToken != "sess" {
		t.Errorf("expected sessionToken sess, got %s", opts.sessionToken)
	}
	if opts.idToken != "ident" {
		t.Errorf("expected idToken ident, got %s", opts.idToken)
	}
	if opts.owner != "owner-id" {
		t.Errorf("expected owner owner-id, got %s", opts.owner)
	}
	if !opts.ipv6 {
		t.Errorf("expected ipv6 true")
	}
	if !opts.exec {
		t.Errorf("expected exec true")
	}
	if !opts.service {
		t.Errorf("expected service true")
	}
	if len(opts.javaFlags) != 2 || opts.javaFlags[0] != "-XX:+PrintGC" || opts.javaFlags[1] != "-XX:+ExitOnOutOfMemoryError" {
		t.Errorf("unexpected javaFlags: %v", opts.javaFlags)
	}
}
