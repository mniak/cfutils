package main

import (
	"fmt"
	"os"

	"github.com/mniak/cfutils/codefresh"
	"github.com/spf13/cobra"
)

func newTokensCommand() *cobra.Command {
	var browser string

	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Show Codefresh tokens from the browser's local storage",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			accessToken, refreshToken, err := codefresh.TokensFromBrowser(browser)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Access token:  %s\n", accessToken)
			if refreshToken != "" {
				fmt.Printf("Refresh token: %s\n", refreshToken)
			}
		},
	}

	cmd.Flags().StringVarP(&browser, "browser", "b", "chrome", "Browser to read tokens from (chrome, brave, edge, etc.)")

	return cmd
}