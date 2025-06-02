// From: https://github.com/atomicgo/splitslog

/*
MIT License

Copyright (c) 2024 Marvin Wendt (aka. MarvinJWendt)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package splitslog_test

import (
	"log/slog"
	"os"

	"github.com/gstos/qbcli/internal/splistlog"
)

func Example_demo() {
	splitter := splitslog.Splitter{
		// Debug and info messages are printed to stdout.
		slog.LevelDebug: slog.NewJSONHandler(os.Stdout, nil),
		slog.LevelInfo:  slog.NewJSONHandler(os.Stdout, nil),

		// Warn and error messages are printed to stderr.
		slog.LevelWarn:  slog.NewJSONHandler(os.Stderr, nil),
		slog.LevelError: slog.NewJSONHandler(os.Stderr, nil),
	}

	handler := splitslog.NewSplitHandler(splitter)
	logger := slog.New(handler)

	logger.Info("info message prints to stdout")
	logger.Error("error message prints to stderr")

	// stdout: {"time":"2023-09-07T16:56:22.563817+02:00","level":"INFO","msg":"info message prints to stdout"}
	// stderr: {"time":"2023-09-07T16:56:22.564103+02:00","level":"ERROR","msg":"error message prints to stderr"}
}

func ExampleNewSplitHandler() {
	splitter := splitslog.Splitter{
		// Debug and info messages are printed to stdout.
		slog.LevelDebug: slog.NewJSONHandler(os.Stdout, nil),
		slog.LevelInfo:  slog.NewJSONHandler(os.Stdout, nil),

		// Warn and error messages are printed to stderr.
		slog.LevelWarn:  slog.NewJSONHandler(os.Stderr, nil),
		slog.LevelError: slog.NewJSONHandler(os.Stderr, nil),
	}

	handler := splitslog.NewSplitHandler(splitter)
	logger := slog.New(handler)

	logger.Info("info message prints to stdout")
	logger.Error("error message prints to stderr")

	// stdout: {"time":"2023-09-07T16:56:22.563817+02:00","level":"INFO","msg":"info message prints to stdout"}
	// stderr: {"time":"2023-09-07T16:56:22.564103+02:00","level":"ERROR","msg":"error message prints to stderr"}
}
