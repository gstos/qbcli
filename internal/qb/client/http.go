package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"syscall"
	"time"
)

func (cli *Client) Prepare(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
	headers map[string]string,
	payload []byte,
) (*http.Request, error) {
	requestURL := cli.BuildURL(path)

	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, FatalErrorFrom(err, "creating HTTP request")
	}

	// Refer to: https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#authentication
	hostURL := cli.credentials.DeriveBaseURL()
	httpReq.Header.Set("Origin", hostURL)
	httpReq.Header.Set("Referer", hostURL)

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	cli.Log.Debug("request prepared", "method", method, "url", requestURL)
	return httpReq, nil
}

func (cli *Client) PrepareJSON(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
	payload any,
) (*http.Request, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, FatalErrorFrom(err, "marshaling payload")
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	return cli.Prepare(ctx, method, path, params, headers, payloadJSON)
}

func (cli *Client) PrepareForm(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
	form url.Values,
) (*http.Request, error) {
	var payload []byte
	if form != nil {
		payload = []byte(form.Encode())
	}

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	return cli.Prepare(ctx, method, path, params, headers, payload)
}

func (cli *Client) fetchRequest(ctx context.Context, req *http.Request) ([]byte, *http.Response, error) {
	log := cli.Log.With("method", req.Method, "url", req.URL.String())
	log.Debug("executing HTTP request")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		switch {
		case isTimeOut(err) || errors.Is(ctx.Err(), context.DeadlineExceeded):
			return nil, nil, TransientErrorFrom(err, "request timed out")
		case isConnRefused(err):
			return nil, nil, TransientErrorFrom(err, "connection refused")
		default:
			return nil, nil, FatalErrorFrom(err, "failed before receiving response")
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("closing response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
			return nil, nil, TransientErrorFrom(err, "request timed out")
		case isTimeOut(err):
			return nil, nil, TransientErrorFrom(err, "network error")
		default:
			return nil, nil, FatalErrorFrom(err, "failed reading response body")
		}
	}

	log.Debug("response received", "status", resp.StatusCode, "body", string(body))
	return body, resp, nil
}

func (cli *Client) handleResponseStatus(resp *http.Response, authCached bool) (bool, error) {
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return false, nil
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return true, NewFatalError("unexpected redirection failure: %s [%d]", resp.Status, resp.StatusCode)
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden && authCached:
		if err := cli.CleanAuthCookie(); err != nil {
			cli.forceAuth = true
			return false, NewTransientError("cached authentication failed; failed to clean up cache; forcing re-authentication")
		}
		return true, NewFatalError("cached authentication failed; cached credentials removed")
	case cli.isTransientStatus(resp.StatusCode):
		return false, NewTransientError("transient error: %s [%d]", resp.Status, resp.StatusCode)
	default:
		return true, NewFatalError("unexpected response: %s [%d]", resp.Status, resp.StatusCode)
	}
}

func (cli *Client) isTransientStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func parseRetryAfter(resp *http.Response) (delay time.Duration, ok bool) {
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return 0, false
	}

	// Try parsing as integer seconds
	if seconds, err := strconv.Atoi(ra); err == nil {
		return time.Duration(seconds) * time.Second, true
	}

	// Try parsing as RFC1123 date format
	if t, err := http.ParseTime(ra); err == nil {
		return time.Until(t), true
	}

	// Invalid format
	return 0, false
}

func isTimeOut(err error) bool {
	var netErr interface{ Timeout() bool }
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isConnRefused(err error) bool {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}

	// Case 1: wrapped syscall inside *os.SyscallError
	var sysErr *os.SyscallError
	if errors.As(opErr.Err, &sysErr) {
		return errors.Is(sysErr.Err, syscall.ECONNREFUSED)
	}

	// Case 2: syscall.Errno directly
	return errors.Is(opErr.Err, syscall.ECONNREFUSED)
}
