package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var getListeningPortCmd = &cobra.Command{
	Use:   "getListeningPort <port>",
	Short: "Get qBittorrent listening port",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := rootEnv.Context()
		defer cancel()

		cli, err := rootEnv.Client()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		listeningPort, err := cli.GetListeningPort(ctx)
		if err != nil {
			return fmt.Errorf("failed to set listening port: %w", err)
		}
		cli.Log.Info("Listening port get successfully", "port", listeningPort)

		fmt.Println(listeningPort)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getListeningPortCmd)
}
