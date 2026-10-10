package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flowkit/internal/direct"
)

func TestEvidenceUpdatePreservesQueueHandleAndRedactsSecrets(t *testing.T) {
	t.Setenv("FAL_KEY", "test-secret-key")
	path := filepath.Join(t.TempDir(), "experiment", "evidence.json")
	record := evidence{ID: "experiment-id"}
	if err := save(path, record); err != nil {
		t.Fatal(err)
	}
	record.Queue = &direct.QueueHandle{RequestID: "provider-id", StatusURL: "https://queue.fal.run/status", ResponseURL: "https://queue.fal.run/result"}
	record.Error = "provider echoed test-secret-key"
	if err := save(path, record); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "test-secret-key") {
		t.Fatal("secret persisted in evidence")
	}
	var reloaded evidence
	if err := json.Unmarshal(payload, &reloaded); err != nil {
		t.Fatal(err)
	}
	if reloaded.Queue == nil || reloaded.Queue.RequestID != "provider-id" {
		t.Fatal("queue handle was not persisted")
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatal("temporary files left behind")
	}
}
