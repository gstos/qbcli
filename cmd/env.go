package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gstos/qbcli/internal/qb/client"
	"github.com/gstos/qbcli/internal/qb/cookiejar"
	"github.com/gstos/qbcli/internal/qb/credentials"
	"github.com/gstos/qbcli/internal/splistlog"
)

type Environment struct {
	cacheDir      string
	noCache       bool
	hostRawURL    string
	username      string
	password      string
	logLevelStr   string
	timeOut       time.Duration
	listeningPort int
	forceAuth     bool
	retry         bool
	maxRetries    int
	delay         time.Duration

	hostURL   *url.URL
	client    *client.Client
	ctx       context.Context
	ctxCancel context.CancelFunc
	Log       *slog.Logger
	LogLevel  *slog.LevelVar
}

const defaultLocalHostURL = "http://127.0.0.1:8080"
const defaultLogLevelStr = "warn"
const defaultTimeOut = 30 * time.Second
const defaultMaxRetries = 5
const defaultDelay = 30 * time.Second

func defaultHostURL() string {
	if envHostURL := os.Getenv("QBCLI_HOST_URL"); envHostURL != "" {
		return envHostURL
	}
	return defaultLocalHostURL
}

func defaultCacheDir() string {
	if envCacheDir := os.Getenv("QBCLI_CACHE_DIR"); envCacheDir != "" {
		return envCacheDir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".qbcli"
	}
	return filepath.Join(home, ".cache", "qbcli")
}

func defaultUsername() string {
	if envUsername := os.Getenv("QBCLI_USERNAME"); envUsername != "" {
		return envUsername
	}
	return ""
}

func (env *Environment) HostURL() (*url.URL, error) {
	if env.hostURL != nil {
		return env.hostURL, nil
	}

	if env.hostRawURL == "" {
		return nil, fmt.Errorf("host URL is empty")
	}

	if u, err := url.Parse(env.hostRawURL); err != nil {
		return nil, fmt.Errorf("invalid host URL: %w", err)
	} else {
		env.hostURL = u
		return u, nil
	}
}

func (env *Environment) Client() (*client.Client, error) {
	if env.client != nil {
		return env.client, nil
	}

	var opts []client.Option

	if jar := env.CookieJar(); jar != nil {
		opts = append(opts, client.WithCookieJar(jar))
	}

	if env.forceAuth {
		opts = append(opts, client.WithForceAuth())
	}

	if env.retry {
		opts = append(opts, client.WithRetry(env.maxRetries, env.delay))
	}

	if logger, err := env.Logger(); err != nil {
		return nil, fmt.Errorf("invalid logger: %w", err)
	} else {
		opts = append(opts, client.WithLogger(logger))
	}

	if env.timeOut > 0 {
		opts = append(opts, client.WithTimeOut(env.timeOut))
	}

	creds, err := env.Credentials()
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	env.client = client.New(creds, opts...)
	return env.client, nil
}

func (env *Environment) Credentials() (*credentials.Credentials, error) {
	var opts []credentials.Option

	hostURL, err := env.HostURL()
	if err != nil {
		return nil, err
	}

	if env.username != "" {
		opts = append(opts, credentials.WithUsername(env.username))
	}

	if env.password != "" {
		opts = append(opts, credentials.WithPassword(env.password))
	}

	return credentials.FromURL(hostURL, opts...)
}

func (env *Environment) CookieJar() *cookiejar.CookieJar {
	if env.noCache || env.cacheDir == "" {
		return nil
	}
	logger, err := env.Logger()
	if err != nil {
		logger.Error("failed to create cookie jar", "error", err)
		return nil
	}
	return cookiejar.New(env.cacheDir, cookiejar.WithLogger(logger))
}

func (env *Environment) Context() (context.Context, context.CancelFunc) {
	if env.ctx != nil {
		return env.ctx, env.ctxCancel
	}

	ctx := context.Background()
	env.ctx, env.ctxCancel = context.WithCancel(ctx)

	return env.ctx, env.ctxCancel
}

func (env *Environment) Logger() (*slog.Logger, error) {
	if env.Log != nil {
		return env.Log, nil
	}

	logLevel, err := parseLogLevel(env.logLevelStr)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{Level: env.LogLevel}
	if logLevel <= slog.LevelInfo {
		splitter := splitslog.Splitter{
			slog.LevelDebug: slog.NewTextHandler(os.Stdout, handlerOpts),
			slog.LevelInfo:  slog.NewTextHandler(os.Stdout, handlerOpts),
			slog.LevelWarn:  slog.NewTextHandler(os.Stderr, handlerOpts),
			slog.LevelError: slog.NewTextHandler(os.Stderr, handlerOpts),
		}
		handler = splitslog.NewSplitHandler(splitter)
	} else {
		handler = slog.NewTextHandler(os.Stderr, handlerOpts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	env.Log = logger
	env.LogLevel.Set(logLevel)
	return env.Log, nil
}

func parseLogLevel(levelStr string) (slog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelError, fmt.Errorf("invalid log level: %s", levelStr)
	}
}

func (env *Environment) ListeningPort() (int, error) {
	if env.listeningPort < 0 || env.listeningPort > 65535 {
		return 0, fmt.Errorf("invalid listening port: %d", env.listeningPort)
	}
	return env.listeningPort, nil
}
