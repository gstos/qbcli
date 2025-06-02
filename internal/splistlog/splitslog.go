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

package splitslog

import (
	"context"
	"fmt"
	"log/slog"
)

// Splitter is a map of log levels to handlers.
// The default log levels (slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError) must be present,
// otherwise the SplitHandler panics.
type Splitter map[slog.Level]slog.Handler

// SplitHandler is a handler that splits log records to different handlers based on their level.
type SplitHandler struct {
	Splitter Splitter

	goas []groupOrAttrs
}

// NewSplitHandler returns a new SplitHandler.
func NewSplitHandler(splitter Splitter) *SplitHandler {
	switch {
	case splitter == nil:
		panic("splitter of SplitHandler must not be nil")
	case splitter[slog.LevelDebug] == nil:
		panic("splitter of SplitHandler must have a handler for debug level")
	case splitter[slog.LevelInfo] == nil:
		panic("splitter of SplitHandler must have a handler for info level")
	case splitter[slog.LevelWarn] == nil:
		panic("splitter of SplitHandler must have a handler for warn level")
	case splitter[slog.LevelError] == nil:
		panic("splitter of SplitHandler must have a handler for error level")
	}

	return &SplitHandler{Splitter: splitter, goas: make([]groupOrAttrs, 0)}
}

// Enabled implements Handler.Enabled.
func (h *SplitHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.getHandler(level).Enabled(ctx, level)
}

// Handle implements Handler.Handle.
func (h *SplitHandler) Handle(ctx context.Context, record slog.Record) error {
	handler := h.getHandler(record.Level)

	for _, goa := range h.goas {
		if goa.group != "" {
			handler = handler.WithGroup(goa.group)
		}

		if len(goa.attrs) > 0 {
			handler = handler.WithAttrs(goa.attrs)
		}
	}

	return handler.Handle(ctx, record) //nolint:wrapcheck
}

// WithAttrs implements Handler.WithAttrs.
func (h *SplitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

// WithGroup implements Handler.WithGroup.
func (h *SplitHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (h *SplitHandler) getHandler(level slog.Level) slog.Handler {
	handler, ok := h.Splitter[level]
	if !ok {
		panic(fmt.Sprintf("no handler registered for level %s", level))
	}

	return handler
}

func (h *SplitHandler) withGroupOrAttrs(goa groupOrAttrs) *SplitHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa

	return &h2
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}
