package main

import (
	"fmt"
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flags.Browser == "" {
				return fmt.Errorf("required flag \"browser\" not set")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVarP(&flags.Browser, "browser", "b", "", "Browser to read tokens from (chrome, brave, edge, etc.)")

	cmd.AddCommand(newTokensCommand())
	cmd.AddCommand(newMCPCommand())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}