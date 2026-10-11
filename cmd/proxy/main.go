// proxy runs Caddy and adds the FlowKit log contract to its native process logs.
// It has no application database, session secrets or provider credentials.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
)

func main() {
	log := telemetry.Logger("edge", "proxy-process")
	if err := run(); err != nil {
		// Avoid echoing process arguments or environment on failure.
		log.Error("Proxy process failed", "event", "proxy.failed", "error", err.Error())
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 || os.Args[1] != "caddy" {
		return fmt.Errorf("proxy requires a caddy command")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	reader, writer, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("proxy log pipe unavailable")
	}
	defer reader.Close()
	defer writer.Close()
	child := exec.Command("caddy", os.Args[2:]...)
	child.Stdout, child.Stderr = writer, writer
	if err := child.Start(); err != nil {
		return fmt.Errorf("cannot start caddy")
	}
	_ = writer.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = child.Process.Signal(syscall.SIGTERM)
		case <-done:
		}
	}()
	readError := forwardLogs(reader, os.Stdout)
	if readError != nil {
		_ = child.Process.Kill()
	}
	waitError := child.Wait()
	if readError != nil {
		return fmt.Errorf("proxy log forwarding failed")
	}
	if waitError != nil {
		return fmt.Errorf("caddy exited unsuccessfully")
	}
	return nil
}

func forwardLogs(reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	encoder := json.NewEncoder(writer)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &fields); err != nil || fields == nil {
			message, _ := json.Marshal(scanner.Text())
			fields = map[string]json.RawMessage{"msg": message, "level": json.RawMessage(`"error"`), "event": json.RawMessage(`"proxy.process_output"`)}
		}
		fields["service"] = json.RawMessage(`"edge"`)
		for _, key := range []string{"run_id", "node_id", "attempt"} {
			fields[key] = json.RawMessage("null")
		}
		if _, hasTS := fields["ts"]; !hasTS {
			if _, hasTime := fields["time"]; !hasTime {
				fields["time"], _ = json.Marshal(time.Now().UTC())
			}
		}
		if err := encoder.Encode(fields); err != nil {
			return err
		}
	}
	return scanner.Err()
}
