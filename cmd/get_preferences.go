package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var getPreferencesCmd = &cobra.Command{
	Use:   "getPreferences",
	Short: "Get qBittorrent preferences",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := rootEnv.Context()
		defer cancel()

		cli, err := rootEnv.Client()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		prefs, err := cli.GetPreferences(ctx)
		if err != nil {
			return fmt.Errorf("failed to get preferences: %w", err)
		}

		output, err := json.MarshalIndent(prefs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal preferences: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getPreferencesCmd)
}
