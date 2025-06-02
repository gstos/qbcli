package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var setListeningPortCmd = &cobra.Command{
	Use:   "setListeningPort <port>",
	Short: "Set qBittorrent listening port",
	Args:  cobra.ExactArgs(1), // Require exactly one argument
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := rootEnv.Context()
		defer cancel()

		cli, err := rootEnv.Client()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		port, err := strconv.Atoi(args[0])
		if err != nil || port < 0 || port > 65535 {
			return fmt.Errorf("invalid port: %s", args[0])
		}

		if err := cli.SetListeningPort(ctx, port); err != nil {
			return fmt.Errorf("failed to set listening port: %w", err)
		}

		cli.Log.Info("Listening port set successfully", "port", port)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setListeningPortCmd)
}
