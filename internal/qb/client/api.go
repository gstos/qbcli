package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func (cli *Client) Login(ctx context.Context) ([]byte, *http.Response, error) {
	body, _, err := cli.Get(ctx, "app/version", nil, nil, cli.SessionAuth)
	if err != nil {
		cli.Log.Error("logging in", "creds", cli.credentials)
		return nil, nil, FatalErrorFrom(err, "login for %s", cli.credentials)
	}
	fmt.Println(string(body))
	return body, nil, nil
}

func (cli *Client) Logout(ctx context.Context) error {
	defer func() {
		err := cli.CleanAuthCookie()
		if err != nil {
			cli.Log.Error("cleaning up failed", "error", err)
		}
	}()

	_, found := cli.getAuthCookie()
	if !found {
		cli.Log.Warn("no valid authCookie found; presuming already logged out")
		return nil
	}

	if _, _, err := cli.Post(ctx, "auth/logout", nil, nil, cli.SessionAuth); err != nil {
		cli.Log.Error("logging out", "creds", cli.credentials)
		return FatalErrorFrom(err, "logout for %s", cli.credentials)
	}
	return nil
}

func (cli *Client) GetPreferences(ctx context.Context) (map[string]any, error) {
	data, err := cli.GetResource(ctx, "app/preferences", nil, nil, cli.SessionAuth)
	if err != nil {
		cli.Log.Error("getting preferences", "error", err)
		return nil, fmt.Errorf("getting preferences: %w", err)
	}

	if prefs, ok := data.(map[string]any); !ok {
		cli.Log.Error("invalid preferences")
		return nil, fmt.Errorf("invalid preferences")
	} else {
		return prefs, nil
	}
}

func (cli *Client) GetPreferenceEntry(ctx context.Context, entry string) (any, error) {
	prefs, err := cli.GetPreferences(ctx)
	log := cli.Log.With("entry", entry)

	if err != nil {
		log.Error("getting all preferences", "error", err)
		return "", fmt.Errorf("getting all preferences: %w", err)
	}

	value, ok := prefs[entry]
	if !ok {
		log.Error("missing entry")
		return "", fmt.Errorf("missing 'listen_port' from preferences")
	}

	return value, nil
}

func (cli *Client) SetPreferences(ctx context.Context, prefs map[string]any) error {
	jsonPayload, err := json.Marshal(prefs)
	if err != nil {
		cli.Log.Error("failed to marshal preferences", "error", err)
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	form := url.Values{
		"json": {string(jsonPayload)},
	}

	_, _, err = cli.PostForm(ctx, "app/setPreferences", nil, form, cli.SessionAuth)
	if err != nil {
		cli.Log.Error("setting preferences", "error", err)
		return fmt.Errorf("setting preferences: %w", err)
	}

	cli.Log.Info("preferences set", "payload", jsonPayload)
	return nil
}

func (cli *Client) SetListeningPort(ctx context.Context, port int) error {
	if port < 0 || port > 65535 {
		cli.Log.Error("invalid port", "port", port)
		return fmt.Errorf("invalid port: %d", port)
	}

	err := cli.SetPreferences(ctx, map[string]any{"listen_port": port})
	if err != nil {
		cli.Log.Error("setting listening port", "port", port, "error", err)
		return fmt.Errorf("setting preferences: %w", err)
	}

	cli.Log.Info("listening port set", "port", port)
	return nil
}

func (cli *Client) GetListeningPort(ctx context.Context) (int, error) {
	value, err := cli.GetPreferenceEntry(ctx, "listen_port")
	if err != nil {
		cli.Log.Error("getting listening port", "error", err)
		return -1, fmt.Errorf("getting listening port: %w", err)
	}

	var port int
	switch value := value.(type) {
	case string:
		port, err = strconv.Atoi(value)
		if err != nil {
			cli.Log.Error("invalid listening port", "port", value)
			return -1, fmt.Errorf("invalid listening port: %v", value)
		}
		return port, nil
	case float64:
		port = int(value)
		return port, nil
	default:
		cli.Log.Error("invalid listening port", "port", value)
		return -1, fmt.Errorf("invalid listening port: %v", value)
	}
}
