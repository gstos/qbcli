package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
)

var setPreferencesCmd = &cobra.Command{
	Use:   "setPreferences [json]",
	Short: "Set qBittorrent preferences",
	Long: `Set preferences using a JSON string or file.
If --file is specified, it will read JSON from the given file path or '-' for stdin.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := rootEnv.Context()
		defer cancel()

		cli, err := rootEnv.Client()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		// Check if a file flag was provided
		filePath, _ := cmd.Flags().GetString("file")

		var jsonData []byte
		switch {
		case filePath != "":
			var reader io.Reader
			if filePath == "-" {
				reader = os.Stdin
			} else {
				file, err := os.Open(filePath)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer func() { _ = file.Close() }()
				reader = file
			}
			jsonData, err = io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

		case len(args) == 1:
			jsonData = []byte(args[0])

		default:
			return fmt.Errorf("must specify either a JSON string or --file")
		}

		var raw any
		if err := json.Unmarshal(jsonData, &raw); err != nil {
			return fmt.Errorf("failed to validate preferences: %w", err)
		}

		prefs, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid preferences format: must be JSON object")
		}

		if err := cli.SetPreferences(ctx, prefs); err != nil {
			return fmt.Errorf("failed to set preferences: %w", err)
		}

		cli.Log.Info("Preferences successfully updated")
		return nil
	},
}

func init() {
	setPreferencesCmd.Flags().String("file", "", "Path to JSON file or '-' for stdin")
	rootCmd.AddCommand(setPreferencesCmd)
}
