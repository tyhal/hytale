package main

import (
	"github.com/spf13/cobra"
	"github.com/tyhal/hytale/pkg/downloader"
	"github.com/tyhal/x/flop"
)

func cmdUpdate() *cobra.Command {
	var downloaderOpts flop.With[downloader.Options]

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Download latest Hytale server assets and build AOT cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			dl, err := downloader.New(downloaderOpts.Get(cmd)...)
			if err != nil {
				return err
			}
			return dl.Update()
		},
	}

	cmd.Flags().Bool("allow-aot-fail", false, "Allow AOT cache generation to fail without aborting")
	downloaderOpts = append(downloaderOpts, flop.Toggle("allow-aot-fail", downloader.WithAllowAOTFail))

	cmd.Flags().String("credentials-path", "", "Path to credentials file")
	downloaderOpts = append(downloaderOpts, flop.String("credentials-path", downloader.WithCredentialsPath))

	return cmd
}
