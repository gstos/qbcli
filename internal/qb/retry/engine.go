package retry

import (
	"context"
	"fmt"
	"github.com/gstos/qbcli/internal/qb/multierror"
	"io"
	"log/slog"
	"os"
	"time"
)

type Option func(*Engine)

type Engine struct {
	ID          string
	Context     context.Context
	ctxCancel   context.CancelFunc
	curAttempt  int
	errors      []error
	retry       bool
	retryDelay  time.Duration
	maxRetries  int
	timeOut     time.Duration
	Log         *slog.Logger
	LogLevel    *slog.LevelVar
	SetUp       func(*Engine) error
	Prepare     func(*Engine) error
	Do          func(*Engine) error
	WrapError   func(error, string, ...any) error
	IsTransient func(error) bool
}

func defaultOptions() []Option {
	return []Option{
		WithLogger(slog.New(slog.DiscardHandler)),
	}
}

func New(
	id string,
	doFunc func(*Engine) error,
	opts ...Option,
) *Engine {
	eng := &Engine{
		ID:          id,
		curAttempt:  1,
		retry:       false,
		maxRetries:  1,
		timeOut:     0,
		LogLevel:    new(slog.LevelVar),
		SetUp:       func(*Engine) error { return nil },
		Prepare:     func(*Engine) error { return nil },
		Do:          doFunc,
		WrapError:   func(e error, f string, args ...any) error { return fmt.Errorf(f+": %w", append(args, e)...) },
		IsTransient: func(error) bool { return false },
	}
	opts = append(defaultOptions(), opts...)
	for _, opt := range opts {
		opt(eng)
	}

	return eng
}

func FromEngine(fromEng *Engine) Option {
	return func(toEng *Engine) {
		toEng.curAttempt = fromEng.curAttempt
		toEng.retry = fromEng.retry
		toEng.maxRetries = fromEng.maxRetries
		toEng.timeOut = fromEng.timeOut
		toEng.Log = fromEng.Log
		toEng.LogLevel = fromEng.LogLevel
	}
}

func WithTransientErrorCheck(isFatal func(error) bool) Option {
	return func(eng *Engine) {
		eng.IsTransient = isFatal
	}
}

func WithErrorWrap(wrapError func(error, string, ...any) error) Option {
	return func(eng *Engine) {
		eng.WrapError = wrapError
	}
}

func WithRetry(maxRetries int, retryDelay time.Duration) Option {
	return func(eng *Engine) {
		eng.retry = true
		eng.maxRetries = maxRetries
		eng.retryDelay = retryDelay
	}
}

func WithTimeOut(timeout time.Duration) Option {
	return func(eng *Engine) {
		eng.timeOut = timeout
	}
}

func WithSetUp(setUp func(*Engine) error) Option {
	return func(eng *Engine) {
		eng.SetUp = setUp
	}
}

func WithPrepare(prepare func(*Engine) error) Option {
	return func(eng *Engine) {
		eng.Prepare = prepare
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(eng *Engine) {
		eng.Log = logger
	}
}

func WithCustomLogger(w io.Writer, opts *slog.HandlerOptions) Option {
	return func(eng *Engine) {
		if opts == nil {
			opts = &slog.HandlerOptions{}
		}

		if opts.Level == nil {
			opts.Level = eng.LogLevel
		}

		eng.Log = slog.New(slog.NewTextHandler(w, opts))
	}
}

func WithLogLevel(logLevel slog.Level) Option {
	return func(eng *Engine) {
		eng.Log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: eng.LogLevel}))
		eng.LogLevel.Set(logLevel)
	}
}

func (eng *Engine) CurAttempt() int {
	return eng.curAttempt
}

func (eng *Engine) RetryDelay() time.Duration {
	return eng.retryDelay
}

func (eng *Engine) Errors() []error {
	return eng.errors
}

func (eng *Engine) Unwrap() error {
	return multierror.WrapIfError(eng.errors)
}

func (eng *Engine) runFunc(msg string, f func(*Engine) error) (error, bool) {
	err := f(eng)
	if err == nil {
		eng.Log.Debug(msg, "result", "success")
		return nil, false
	}
	err = eng.WrapError(err, "%s: %s", msg, eng.ID)

	if isFatal := eng.IsTransient(err); isFatal {
		eng.Log.Error(msg, "result", "fatal", "error", err)
		return err, true
	}

	eng.Log.Warn(msg, "result", "transient", "error", err)
	return err, false
}

func (eng *Engine) Wait(delay time.Duration) error {
	select {
	case <-eng.Context.Done():
		msg := "execution loop aborted by context"
		err := eng.WrapError(eng.Context.Err(), "%s: %s", msg, eng.ID)
		eng.errors = append(eng.errors, err)
		eng.Log.Error(msg, "result", "canceled", "error", err)
		return err
	case <-time.After(delay):
		return nil
	}
}

func (eng *Engine) Run(ctx context.Context) error {
	// SetUp
	prevLog := eng.Log
	eng.Log = eng.Log.With("ID", eng.ID)
	defer func() { eng.Log = prevLog }()

	if eng.timeOut > 0 {
		eng.Context, eng.ctxCancel = context.WithTimeout(ctx, eng.timeOut)
	} else {
		eng.Context, eng.ctxCancel = context.WithCancel(ctx)
	}
	defer func() {
		if eng.ctxCancel != nil {
			eng.ctxCancel()
		}
		eng.Context = nil
	}()

	if err, isFatal := eng.runFunc("setup", eng.SetUp); isFatal {
		return err
	} else {
		eng.errors = append(eng.errors, err)
	}

	// Retry loop
	for attempt := eng.curAttempt; attempt <= eng.maxRetries || eng.maxRetries == 0; attempt++ {
		eng.Log = prevLog.With("ID", eng.ID, "retry", attempt, "maxRetries", eng.maxRetries, "delay", eng.retryDelay)
		eng.curAttempt = attempt

		if err := eng.Wait(0); err != nil {
			eng.errors = append(eng.errors, err)
			return err
		}

		if err, isFatal := eng.runFunc("preparing", eng.Prepare); err != nil {
			eng.errors = append(eng.errors, err)
			if isFatal {
				return err
			}
		} else if err, isFatal = eng.runFunc("doing", eng.Do); err == nil {
			return nil
		} else {
			eng.errors = append(eng.errors, err)
			if isFatal {
				return err
			}
		}

		if err := eng.Wait(eng.retryDelay); err != nil {
			eng.errors = append(eng.errors, err)
			return err
		}
	}

	err := eng.WrapError(fmt.Errorf("reached %d max attempts", eng.maxRetries), "execution loop terminated")
	eng.errors = append(eng.errors, err)
	return err
}
