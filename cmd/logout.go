package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var logoutUsername string

var logoutCmd = &cobra.Command{
	Use:   "logout <host>",
	Short: "Log out and delete cached cookie",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := rootEnv.Context()
		defer cancel()

		cli, err := rootEnv.Client()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		if err := cli.Logout(ctx); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}

		if err := cli.CleanAuthCookie(); err != nil {
			return fmt.Errorf("could not delete cached cookie: %w", err)
		}

		fmt.Println("Logged out and cookie deleted.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
