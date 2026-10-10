package telemetry

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestJSONContextContract(t *testing.T) {
	var output bytes.Buffer
	handler := &contextHandler{Handler: slog.NewJSONHandler(&output, nil)}
	log := slog.New(handler).With("service", "test")
	log.Info("startup", "event", "test.startup")
	log.With("run_id", "run-1").Error("provider failure", "event", "provider.error", "node_id", "render", "attempt", 2, "provider_error_code", "runner_server_error")
	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
	if len(lines) != 2 {
		t.Fatal(output.String())
	}
	for i, line := range lines {
		var value map[string]any
		if e := json.Unmarshal(line, &value); e != nil {
			t.Fatal(e)
		}
		for _, key := range []string{"run_id", "node_id", "attempt"} {
			if _, ok := value[key]; !ok {
				t.Fatal("missing context", key)
			}
		}
		if i == 1 {
			if value["run_id"] != "run-1" || value["node_id"] != "render" || value["attempt"] != float64(2) || value["level"] != "ERROR" {
				t.Fatal(value)
			}
			if bytes.Count(line, []byte(`"run_id":`)) != 1 {
				t.Fatal("duplicate context key", string(line))
			}
		}
	}
}
