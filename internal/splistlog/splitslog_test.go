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
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"testing/slogtest"

	"github.com/gstos/qbcli/internal/splistlog"
)

func TestSplitHandler(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	splitter := splitslog.Splitter{
		slog.LevelDebug: slog.NewJSONHandler(&buf, nil),
		slog.LevelInfo:  slog.NewJSONHandler(&buf, nil),
		slog.LevelWarn:  slog.NewJSONHandler(&buf, nil),
		slog.LevelError: slog.NewJSONHandler(&buf, nil),
	}

	handler := splitslog.NewSplitHandler(splitter)

	results := func() []map[string]any {
		var resultMap []map[string]any

		for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}

			var m map[string]any
			if err := json.Unmarshal(line, &m); err != nil {
				t.Fatal(err)
			}

			resultMap = append(resultMap, m)
		}

		return resultMap
	}

	err := slogtest.TestHandler(handler, results)
	if err != nil {
		t.Fatal(err)
	}
}
