package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gstos/qbcli/internal/qb/cookiejar"
)

func (cli *Client) getAuthCookie() (*http.Cookie, bool) {
	if cli.cachedCookie != nil && !cookiejar.IsExpired(cli.cachedCookie) {
		return cli.cachedCookie, true
	}

	cli.cachedCookie = nil

	if cli.cookieJar == nil {
		return nil, false
	}

	cookie, err := cli.cookieJar.Retrieve(cli.credentials)
	if err != nil {
		return nil, false
	}

	cli.cachedCookie = cookie
	return cookie, true
}

func (cli *Client) storeAuthCookie(cookie *http.Cookie) error {
	if cookiejar.IsExpired(cookie) {
		return fmt.Errorf("cookie expired")
	}
	cli.cachedCookie = cookie

	if cli.cookieJar == nil {
		return nil
	}

	return cli.cookieJar.Store(cli.credentials, cookie)
}

func (cli *Client) CleanAuthCookie() error {
	if cli.cookieJar == nil {
		cli.Log.Debug("cookie jar is not configured; skipping cleanup")
		return nil
	}

	cli.cachedCookie = nil
	if err := cli.cookieJar.Delete(cli.credentials); err != nil {
		cli.Log.Warn("deleting stored cookie", "error", err)
		return fmt.Errorf("deleting stored cookie: %w", err)
	}
	return nil
}

func (cli *Client) extractAuthCookie(resp *http.Response) (*http.Cookie, error) {
	// Extract SID authCookie
	var authCookie *http.Cookie
	for _, thisCookie := range resp.Cookies() {
		if thisCookie.Name == "SID" {
			authCookie = thisCookie
			break
		}
	}

	if authCookie == nil {
		cli.Log.Error("missing authCookie")
		return nil, NewFatalError("authCookie not found for %s", cli.credentials)
	}

	return authCookie, nil
}

func (cli *Client) NoAuth(ctx context.Context) (*http.Request, error) {
	return nil, nil
}

func (cli *Client) SessionAuth(ctx context.Context) (*http.Cookie, bool, error) {
	if !cli.forceAuth {
		if authCookie, found := cli.getAuthCookie(); found {
			return authCookie, true, nil
		}
	}

	form := url.Values{
		"username": {cli.credentials.Username},
		"password": {cli.credentials.Password},
	}

	req, err := cli.PrepareForm(ctx, "POST", "auth/login", nil, form)
	if err != nil {
		return nil, false, FatalErrorFrom(err, "preparing authentication request for %s", cli.credentials)
	}

	body, resp, err := cli.fetchRequest(ctx, req)
	if err != nil {
		return nil, false, WrapFatalUnlessExplicit(err, "authenticating %s", cli.credentials)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, NewFatalError("authentication request for %s failed with status %s", cli.credentials, resp.Status)
	}

	msg := string(body)
	if !strings.HasPrefix(msg, "Ok") {
		return nil, false, NewFatalError("authentication denied for %s: %s", cli.credentials, msg)
	}

	authCookie, err := cli.extractAuthCookie(resp)

	if err != nil {
		return nil, false, FatalErrorFrom(err, "authentication denied for %s: no session cookie found", cli.credentials)
	}

	cli.Log.Debug("authenticated successfully", "creds", cli.credentials, "expiresAt", authCookie.Expires.Format("2006-01-02 15:04:05 MST"))
	err = cli.storeAuthCookie(authCookie)
	if err != nil {
		cli.Log.Warn("storing authCookie", "error", err)
	}

	return authCookie, false, nil
}
