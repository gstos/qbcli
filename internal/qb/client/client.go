package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gstos/qbcli/internal/qb/cookiejar"
	"github.com/gstos/qbcli/internal/qb/credentials"
	"github.com/gstos/qbcli/internal/qb/retry"
)

type Client struct {
	credentials  *credentials.Credentials
	cookieJar    *cookiejar.CookieJar
	cachedCookie *http.Cookie
	baseEndpoint string
	apiVersion   string
	forceAuth    bool
	retry        bool
	retryCount   int
	retryDelay   time.Duration
	maxRetries   int
	timeOut      time.Duration
	Log          *slog.Logger
	LogLevel     *slog.LevelVar
}

type Option func(*Client)

func defaultOptions() []Option {
	return []Option{
		WithLogger(slog.New(slog.DiscardHandler)),
	}
}

func New(creds *credentials.Credentials, opts ...Option) *Client {
	cli := &Client{
		credentials: creds,
		apiVersion:  "v2",
		LogLevel:    new(slog.LevelVar),
	}
	opts = append(defaultOptions(), opts...)
	for _, opt := range opts {
		opt(cli)
	}
	return cli
}

func WithCookieJar(jar *cookiejar.CookieJar) Option {
	return func(cli *Client) {
		cli.cookieJar = jar
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(cli *Client) {
		cli.Log = logger
	}
}

func WithCustomLogger(w io.Writer, opts *slog.HandlerOptions) Option {
	return func(cli *Client) {
		if opts == nil {
			opts = &slog.HandlerOptions{}
		}

		if opts.Level == nil {
			opts.Level = cli.LogLevel
		}

		cli.Log = slog.New(slog.NewTextHandler(w, opts))
	}
}

func WithLogLevel(logLevel slog.Level) Option {
	return func(cli *Client) {
		cli.Log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cli.LogLevel}))
		cli.LogLevel.Set(logLevel)
	}
}

func WithForceAuth() Option {
	return func(cli *Client) {
		cli.forceAuth = true
	}
}

func WithTimeOut(timeOut time.Duration) Option {
	return func(cli *Client) {
		cli.timeOut = timeOut
	}
}

func WithRetry(maxRetries int, delay time.Duration) Option {
	return func(cli *Client) {
		cli.retry = true
		cli.retryDelay = delay
		cli.maxRetries = maxRetries
	}
}

func (cli *Client) SessionCookie() (*http.Cookie, bool) {
	if cli.cachedCookie != nil {
		return cli.cachedCookie, true
	}
	return nil, false
}

func (cli *Client) BaseEndpoint() string {
	if cli.baseEndpoint == "" {
		baseURL := cli.credentials.DeriveBaseURL()
		cli.baseEndpoint = fmt.Sprintf("%s/api/%s/", baseURL, cli.apiVersion)
	}
	return cli.baseEndpoint
}

func (cli *Client) BuildURL(path string) string {
	return fmt.Sprintf("%s%s", cli.BaseEndpoint(), path)
}

func (cli *Client) Fetch(
	ctx context.Context,
	req *http.Request,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) ([]byte, *http.Response, error) {
	var httpResponse *http.Response
	var httpResponseBody []byte
	var authCached bool

	requestID := fmt.Sprintf("%s %s", req.Method, req.URL.String())

	prepare := func(eng *retry.Engine) error {
		var authCookie *http.Cookie
		if auth != nil {
			if cookie, cached, err := auth(eng.Context); err != nil {
				return err
			} else {
				authCookie, authCached = cookie, cached
			}
		}
		req.AddCookie(authCookie)
		return nil
	}

	do := func(eng *retry.Engine) error {
		if body, resp, err := cli.fetchRequest(eng.Context, req); err != nil {
			return err
		} else {
			httpResponse, httpResponseBody = resp, body
		}

		isFatal, err := cli.handleResponseStatus(httpResponse, authCached)
		if err == nil || isFatal {
			return err
		}

		retryAfter, ok := parseRetryAfter(httpResponse)
		if !ok {
			return err
		}

		delay := retryAfter - eng.RetryDelay()
		if delay < 0 {
			return err
		}

		if waitErr := eng.Wait(delay); waitErr != nil {
			err = FatalErrorFrom(waitErr, "while waiting for retry delay")
		}
		return err
	}

	eng := retry.New(
		requestID,
		do,
		retry.WithPrepare(prepare),
		retry.WithTransientErrorCheck(IsTransientError),
		retry.WithErrorWrap(WrapFatalUnlessExplicit),
		retry.WithLogger(cli.Log),
		retry.WithRetry(cli.maxRetries, cli.retryDelay),
	)

	if err := eng.Run(ctx); err != nil {
		return nil, nil, err
	}

	if httpResponse == nil {
		panic("unexpected nil httpResponse")
	}

	return httpResponseBody, httpResponse, nil
}

func (cli *Client) Do(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
	payload []byte,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) ([]byte, *http.Response, error) {
	req, err := cli.Prepare(ctx, method, path, params, nil, payload)
	if err != nil {
		return nil, nil, err
	}

	return cli.Fetch(ctx, req, auth)
}

func (cli *Client) Get(
	ctx context.Context,
	path string,
	params url.Values,
	payload []byte,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) ([]byte, *http.Response, error) {
	return cli.Do(ctx, "GET", path, params, payload, auth)
}

func (cli *Client) Post(
	ctx context.Context,
	path string,
	params url.Values,
	payload []byte,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) ([]byte, *http.Response, error) {
	return cli.Do(ctx, "POST", path, params, payload, auth)
}

func (cli *Client) PostForm(
	ctx context.Context,
	path string,
	params url.Values,
	form url.Values,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) ([]byte, *http.Response, error) {
	req, err := cli.PrepareForm(ctx, "POST", path, params, form)
	if err != nil {
		return nil, nil, err
	}
	return cli.Fetch(ctx, req, auth)
}

func (cli *Client) DoResource(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
	data any,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) (any, error) {
	var payload []byte
	if data != nil {
		if jsonPayload, err := json.Marshal(data); err != nil {
			return nil, FatalErrorFrom(err, "marshalling payload")
		} else {
			payload = jsonPayload
		}
	}

	req, err := cli.Prepare(ctx, method, path, params, nil, payload)
	if err != nil {
		return nil, err
	}

	body, _, err := cli.Fetch(ctx, req, auth)
	if err != nil {
		return nil, err
	}

	var outData any
	if err = json.Unmarshal(body, &outData); err != nil {
		return nil, FatalErrorFrom(err, "unmarshalling response")
	}
	return outData, nil
}

func (cli *Client) GetResource(
	ctx context.Context,
	path string,
	params url.Values,
	data any,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) (any, error) {
	return cli.DoResource(ctx, "GET", path, params, data, auth)
}

func (cli *Client) PostResource(
	ctx context.Context,
	path string,
	params url.Values,
	data any,
	auth func(ctx context.Context) (*http.Cookie, bool, error),
) (any, error) {
	return cli.DoResource(ctx, "POST", path, params, data, auth)
}
