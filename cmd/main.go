package main

import (
	"os"

	"github.com/spf13/cobra"
)

var flags = struct {
	Browser string
}{}

func main() {
	cmd := cobra.Command{
		Use:   "cfutils",
		Short: "Codefresh utilities - token management and MCP server",
	}

	cmd.PersistentFlags().StringVarP(&flags.Browser, "browser", "b", "", "Browser to read tokens from (chrome, brave, edge, etc.)")
	cmd.MarkPersistentFlagRequired("browser")

	cmd.AddCommand(newTokensCommand())
	cmd.AddCommand(newMCPCommand())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
