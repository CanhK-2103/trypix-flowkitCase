package main

import (
	"bytes"
	"fmt"
	"github.com/CanhK-2103/trypix-flowkitCase/examples"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	log := telemetry.Logger("bootstrap", "cli")
	if config.Env("LOAD_EXAMPLES", "true") != "true" {
		log.Info("Example loading disabled", "event", "bootstrap.disabled")
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for _, group := range []string{"models", "workflows"} {
		entries, e := examples.Files.ReadDir(group)
		if e != nil {
			log.Error("Cannot read examples", "event", "bootstrap.failed", "error", e.Error())
			os.Exit(1)
		}
		for _, entry := range entries {
			data, e := examples.Files.ReadFile(group + "/" + entry.Name())
			if e != nil {
				log.Error("Cannot read example", "event", "bootstrap.failed", "error", e.Error())
				os.Exit(1)
			}
			req, _ := http.NewRequest("POST", config.Env("API_URL", "http://localhost:8080")+"/"+group, bytes.NewReader(data))
			req.Header.Set("Content-Type", "application/json")
			res, e := client.Do(req)
			if e != nil {
				log.Error("Example API unavailable", "event", "bootstrap.failed", "error", e.Error())
				os.Exit(1)
			}
			body, _ := io.ReadAll(io.LimitReader(res.Body, 16384))
			res.Body.Close()
			if res.StatusCode != 201 && res.StatusCode != 409 {
				log.Error("Example rejected", "event", "bootstrap.failed", "file", entry.Name(), "status", res.StatusCode, "error", fmt.Sprint(string(body)))
				os.Exit(1)
			}
			log.Info("Example API loading completed", "event", "bootstrap.item", "file", entry.Name(), "status", res.StatusCode)
		}
	}
}
