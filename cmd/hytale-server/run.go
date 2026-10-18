package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tyhal/hytale/pkg/auth"
	"github.com/tyhal/hytale/pkg/downloader"
	"github.com/tyhal/hytale/pkg/server"
	"github.com/tyhal/x/flop"
)

func cmdRun() *cobra.Command {
	var (
		worldPath  string
		noAuth     bool
		reauth     bool
		serverOpts flop.With[server.Options]
	)

	cmd := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			var dynamicServerOpts []server.Options
			if !noAuth {
				authData, err := auth.EnsureAuth()
				if reauth {
					authData, err = auth.Authenticate()
				}
				if err != nil {
					return fmt.Errorf("authentication failed: %w", err)
				}
				dynamicServerOpts = append(dynamicServerOpts,
					server.WithSessionToken(authData.Session.SessionToken),
					server.WithIdentityToken(authData.Session.IdentityToken),
				)
			}

			dl, err := downloader.New(downloader.WithAllowAOTFail())
			if err != nil {
				return err
			}
			if _, err := os.Stat(string(dl.GameJarPath())); err != nil {
				if err := dl.Update(); err != nil {
					return fmt.Errorf("failed to download game server: %w", err)
				}
			}

			return server.RunServer(
				dl.GameJarPath(),
				dl.GameAssetsPath(),
				dl.GameAotPath(),
				worldPath,
				append(dynamicServerOpts, serverOpts.Get(cmd)...)...,
			)
		},
	}

	cmd.Flags().StringVar(&worldPath, "world", "", "Path to world")
	_ = cmd.MarkFlagRequired("world")

	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Skip automatic authentication")
	cmd.Flags().BoolVar(&reauth, "reauth", false, "Force re-authentication via interactive device code flow")

	cmd.Flags().Bool("ipv6", false, "Enable IPv6 support")
	serverOpts = append(serverOpts, flop.Toggle("ipv6", server.WithIPv6))

	cmd.Flags().Bool("exec", false, "Run server directly once-off in foreground")
	serverOpts = append(serverOpts, flop.Toggle("exec", server.WithExec))

	cmd.Flags().Bool("enable-service", false, "Configure systemd service, enable and start it, and verify health")
	serverOpts = append(serverOpts, flop.Toggle("enable-service", server.WithEnableService))

	cmd.Flags().String("backup-path", "", "Path to backup directory")
	serverOpts = append(serverOpts, flop.String("backup-path", server.WithBackups))

	return cmd
}
