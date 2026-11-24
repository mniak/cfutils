package main

import (
	"fmt"
	"os"

	cfutils "github.com/mniak/cfutils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newTokensCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Show Codefresh tokens from the browser's local storage",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			accessToken, refreshToken, err := cfutils.TokensFromBrowser(flags.Browser)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			enc := yaml.NewEncoder(os.Stdout)
			result := map[string]any{
				"access_token": accessToken,
			}
			if refreshToken != "" {
				result["refresh_token"] = refreshToken
			}
			enc.Encode(result)
		},
	}

	return cmd
}
