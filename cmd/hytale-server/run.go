package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tyhal/hytale/pkg/downloader"
	"github.com/tyhal/hytale/pkg/server"
	"github.com/tyhal/x/flop"
)

func cmdRun() *cobra.Command {
	var worldPath string
	var serverOpts flop.With[server.Options]
	var update bool

	cmd := &cobra.Command{
		Use: "run",
		Run: func(cmd *cobra.Command, args []string) {
			hytaleDownloader, err := downloader.New()
			if err != nil {
				fmt.Println(err)
				return
			}
			if update {
				err = hytaleDownloader.Update()
				if err != nil {
					fmt.Println(err)
					return
				}
			}
			err = server.RunServer(
				hytaleDownloader.GameJarPath(),
				hytaleDownloader.GameAssetsPath(),
				hytaleDownloader.GameAotPath(),
				worldPath,
				serverOpts.Get(cmd)...)
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	cmd.Flags().StringVar(&worldPath, "world", "", "Path to world")
	cmd.MarkFlagRequired("world")

	cmd.Flags().BoolVar(&update, "update", false, "Update Hytale server")

	cmd.Flags().Bool("ipv6", false, "Enable IPv6 support")
	serverOpts = append(serverOpts, flop.Toggle("ipv6", server.WithIPv6))

	cmd.Flags().Bool("dry-run", false, "Print the command that would be run instead of running it")
	serverOpts = append(serverOpts, flop.Toggle("dry-run", server.WithDryRun))

	cmd.Flags().String("backup-path", "", "Path to backup directory")
	serverOpts = append(serverOpts, flop.String("backup-path", server.WithBackups))

	return cmd
}
