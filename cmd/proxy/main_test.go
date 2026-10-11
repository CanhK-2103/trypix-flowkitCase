package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNativeAndFallbackLogsKeepContextAndPayload(t *testing.T) {
	input := `{"level":"info","ts":1791747000.123456,"msg":"shutdown finished"}
{"level":"error","logger":"http.log.access","request":{"headers":{"Cookie":["REDACTED"]}},"run_id":"untrusted","status":401}
plain process error with "quotes" and \ slash

null
`
	var output bytes.Buffer
	if err := forwardLogs(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d log lines", len(lines))
	}
	for _, line := range lines {
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatal(err)
		}
		if fields["service"] != "edge" {
			t.Fatal("missing edge service")
		}
		for _, key := range []string{"run_id", "node_id", "attempt"} {
			if value, ok := fields[key]; !ok || value != nil {
				t.Fatalf("missing null %s", key)
			}
		}
	}
	if !strings.Contains(lines[0], "1791747000.123456") || !strings.Contains(lines[1], `"Cookie":["REDACTED"]`) || !strings.Contains(lines[1], `"status":401`) {
		t.Fatal("native log values lost")
	}
	var plain map[string]any
	_ = json.Unmarshal([]byte(lines[2]), &plain)
	if plain["msg"] != `plain process error with "quotes" and \ slash` {
		t.Fatal("fallback message not preserved")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestLogForwardingFailsOnBrokenStreams(t *testing.T) {
	if forwardLogs(strings.NewReader(`{"msg":"test"}`), failedWriter{}) == nil {
		t.Fatal("write error ignored")
	}
	var output bytes.Buffer
	if forwardLogs(strings.NewReader(strings.Repeat("x", 4*1024*1024+1)), &output) == nil {
		t.Fatal("oversized log line accepted")
	}
}
