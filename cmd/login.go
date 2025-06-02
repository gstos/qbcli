package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with qBittorrent and cache the session cookie",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := rootEnv.Context()
		defer cancel()

		cli, err := rootEnv.Client()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		body, _, err := cli.Login(ctx)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		authCookie, ok := cli.SessionCookie()
		if !ok {
			return fmt.Errorf("no valid authCookie found")
		}

		expires := "no"
		if !authCookie.Expires.IsZero() {
			expires = authCookie.Expires.Format("2006-01-02 15:04:05 MST")
		}

		version := string(body)

		cli.Log.Info("Logged in successfully", "version", version, "expires", expires)
		fmt.Println(version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
