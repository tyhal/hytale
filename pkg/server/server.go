package server

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/tyhal/hytale/pkg/auth"
	"github.com/tyhal/hytale/pkg/downloader"
)

type Options func(*serverOptions)
type serverOptions struct {
	backups      string
	sessionToken auth.SessionToken
	idToken      auth.IdentityToken
	owner        auth.OwnerUUID
	ipv6         bool
	dryRun       bool
	javaFlags    []string
}

func WithBackups(backups string) Options {
	return func(opts *serverOptions) {
		opts.backups = backups
	}
}

func WithSessionToken(token auth.SessionToken) Options {
	return func(opts *serverOptions) {
		opts.sessionToken = token
	}
}

func WithIdentityToken(token auth.IdentityToken) Options {
	return func(opts *serverOptions) {
		opts.idToken = token
	}
}

func WithOwner(owner auth.OwnerUUID) Options {
	return func(opts *serverOptions) {
		opts.owner = owner
	}
}

func WithIPv6() Options {
	return func(opts *serverOptions) {
		opts.ipv6 = true
	}
}

func WithJavaFlags(flags ...string) Options {
	return func(opts *serverOptions) {
		opts.javaFlags = append(opts.javaFlags, flags...)
	}
}

func WithExitOnOOM() Options {
	return func(opts *serverOptions) {
		opts.javaFlags = append(opts.javaFlags, "-XX:+ExitOnOutOfMemoryError")
	}
}

func WithDryRun() Options {
	return func(opts *serverOptions) {
		opts.dryRun = true
	}
}

func RunServer(
	serverPath downloader.GameJarPath,
	assetsPath downloader.GameAssetsPath,
	aotPath downloader.GameAotPath,
	worldPath string,
	opts ...Options,
) error {
	serverOpts := serverOptions{}
	for _, opt := range opts {
		opt(&serverOpts)
	}

	// Java args
	args := []string{
		"-XX:MaxRAMPercentage=90.0",
		"-XX:+UseG1GC",                // Newer GC
		"-XX:+UseStringDeduplication", // Feature of G1GC
		"-Xshare:on",
		fmt.Sprintf("-XX:AOTCache=%s", aotPath),
	}
	if serverOpts.javaFlags != nil && len(serverOpts.javaFlags) > 0 {
		args = append(args, serverOpts.javaFlags...)
	}
	if serverOpts.ipv6 {
		args = append(args, "-Djava.net.preferIPv6Addresses=true")
	}
	args = append(args, "-jar", string(serverPath))

	// Server args
	args = append(args, "--assets", string(assetsPath))
	if serverOpts.backups != "" {
		args = append(args, "--backup", "--backup-dir", serverOpts.backups)
	}
	if serverOpts.sessionToken != "" {
		args = append(args, "--session-token", serverOpts.sessionToken)
	}
	if serverOpts.idToken != "" {
		args = append(args, "--identity-token", serverOpts.idToken)
	}
	if serverOpts.owner != "" {
		args = append(args, "--owner-uuid", serverOpts.owner)
	}

	cmd := exec.Command("java", args...)
	cmd.Dir = worldPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if serverOpts.dryRun {
		fmt.Println(cmd.String())
		return nil
	}
	return cmd.Run()
}
