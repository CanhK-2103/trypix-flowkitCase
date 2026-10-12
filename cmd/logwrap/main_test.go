package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestOfficialEntrypointNativeAndAccessLogs(t *testing.T) {
	input := `/docker-entrypoint.sh: Configuration complete
2026-10-12 10:00:00 UTC [1] LOG: database system is ready
2026-10-12 10:00:01 UTC [1] ERROR: syntax error at or near "test"
2026/10/12 10:00:01 [warn] native nginx warning
{"time":"2026-10-12T10:00:00Z","status":200,"duration":0.123456789012,"run_id":"untrusted"}

`
	for _, service := range []string{"database", "dashboard"} {
		var output bytes.Buffer
		if err := forwardLogs(strings.NewReader(input), &output, service, nil); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) != 5 {
			t.Fatal(len(lines))
		}
		for i, line := range lines {
			var value map[string]any
			if err := json.Unmarshal([]byte(line), &value); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"run_id", "node_id", "attempt"} {
				if v, ok := value[key]; !ok || v != nil {
					t.Fatal("context missing", line)
				}
			}
			if value["service"] != service || value["event"] == nil || value["time"] == nil {
				t.Fatal(value)
			}
			if i == 2 && value["level"] != "ERROR" {
				t.Fatal("native provider error level lost")
			}
		}
		if !strings.Contains(lines[4], "0.123456789012") {
			t.Fatal("JSON numeric precision lost")
		}
	}
}

func TestCredentialsRedactedInNativeAndEscapedJSON(t *testing.T) {
	secret := "password-with-\"quote"
	encoded, _ := json.Marshal(map[string]any{"msg": "connection failed: " + secret})
	input := "FATAL: invalid password " + secret + "\n" + string(encoded)
	var output bytes.Buffer
	if err := forwardLogs(strings.NewReader(input), &output, "database", credentialValues([]string{"POSTGRES_PASSWORD=" + secret, "LOG_LEVEL=debug"})); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "password-with-") || !strings.Contains(output.String(), "[REDACTED]") {
		t.Fatal("credential echoed")
	}
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if !json.Valid([]byte(line)) {
			t.Fatal("redaction broke JSON")
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken") }
func TestBrokenStreamsAndUnsupportedEntrypoints(t *testing.T) {
	if forwardLogs(strings.NewReader("log"), brokenWriter{}, "database", nil) == nil {
		t.Fatal("write failure ignored")
	}
	var output bytes.Buffer
	if forwardLogs(strings.NewReader(strings.Repeat("x", 4*1024*1024+1)), &output, "dashboard", nil) == nil {
		t.Fatal("oversized line ignored")
	}
	if run([]string{"unknown", "command"}) == nil {
		t.Fatal("unsupported entrypoint accepted")
	}
}
