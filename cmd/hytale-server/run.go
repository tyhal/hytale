package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tyhal/hytale/pkg/downloader"
	"github.com/tyhal/hytale/pkg/server"
)

// command returns the root command for the klar CLI
func command() *cobra.Command {
	cmd := cobra.Command{
		Use:   "run",
		Short: "",
		Run:   basicRun,
	}

	return &cmd
}

func basicRun(cmd *cobra.Command, args []string) {
	hytaleDownloader, err := downloader.New()
	if err != nil {
		fmt.Println(err)
		return
	}
	//err = hytaleDownloader.Update()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	err = server.RunServer(
		hytaleDownloader.GameJarPath(),
		hytaleDownloader.GameAssetsPath(),
		"",
		server.WithBackups(""),
		server.WithIPv6(),
		server.WithDryRun())
	if err != nil {
		fmt.Println(err)
	}
}
