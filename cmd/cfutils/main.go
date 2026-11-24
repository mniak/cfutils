package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := cobra.Command{
		Use:   "cfutils",
		Short: "Codefresh utilities - token management and MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newTokensCommand())
	cmd.AddCommand(newMCPCommand())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}