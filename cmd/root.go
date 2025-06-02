package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/gstos/qbcli/internal/qb/version"
)

var rootEnv = Environment{
	LogLevel: &slog.LevelVar{},
}

var rootCmd = &cobra.Command{
	Use:           "qbcli",
	Short:         "A CLI tool to interact with qBittorrent WebUI",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if isPasswordRequired(cmd) {
			v, _ := cmd.Flags().GetBool("version")
			if v {
				fmt.Printf("Version: %s\nCommit: %s\nDate: %s\n", version.Version, version.Commit, version.Date)
				os.Exit(0)
				return nil
			}

			if !cmd.Flags().Changed("password") {
				if pwd, pwdSet := os.LookupEnv("QBCLI_PASSWORD"); !pwdSet {
					return fmt.Errorf("password is required")
				} else {
					rootEnv.password = pwd
				}

			}

			// This will cache context and client and validate all required arguments
			_, _ = rootEnv.Context()

			_, err := rootEnv.Client()
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func isPasswordRequired(cmd *cobra.Command) bool {
	path := cmd.CommandPath()
	return path != "qbcli help" && !strings.HasPrefix(path, "qbcli completion")
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootEnv.hostRawURL, "host", "H", defaultHostURL(), "Host URL (overrides QBCLI_HOST_URL)")
	rootCmd.PersistentFlags().StringVarP(&rootEnv.username, "username", "u", defaultUsername(), "Username for qBittorrent (overrides QBCLI_USERNAME)")
	rootCmd.PersistentFlags().StringVarP(&rootEnv.password, "password", "p", "", "Password for qBittorrent (overrides QBCLI_PASSWORD)")
	rootCmd.PersistentFlags().StringVar(&rootEnv.logLevelStr, "log-level", defaultLogLevelStr, "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&rootEnv.cacheDir, "cache", defaultCacheDir(), "Path to cookie cache (overrides QBCLI_CACHE_DIR)")
	rootCmd.PersistentFlags().BoolVar(&rootEnv.noCache, "no-cache", false, "Ignore cookie cache")
	rootCmd.PersistentFlags().BoolVar(&rootEnv.forceAuth, "auth", false, "Force re-authentication with qBittorrent")
	rootCmd.PersistentFlags().DurationVar(&rootEnv.timeOut, "timeout", defaultTimeOut, "Timeout for HTTP requests (use 10s for 10 seconds, set 0 for no timeout)")
	rootCmd.PersistentFlags().BoolVar(&rootEnv.retry, "retry", false, "Enable HTTP request retries")
	rootCmd.PersistentFlags().IntVar(&rootEnv.maxRetries, "max-retries", defaultMaxRetries, "Maximum number of retries for HTTP requests")
	rootCmd.PersistentFlags().DurationVar(&rootEnv.delay, "delay", defaultDelay, "Delay between retries for HTTP requests (in seconds)")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print version and exit")
}
