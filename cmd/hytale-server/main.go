// Package main for hytale-server is a working example of using the pkgs supplied by the repo, it is not supposed to be the main feature
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var version = ""

func main() {
	cmd := rootCmd()
	if err := fang.Execute(context.Background(), cmd, fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "hytale-server"}
	cmd.AddCommand(cmdRun())
	return cmd
}
