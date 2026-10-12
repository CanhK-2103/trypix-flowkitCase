// logwrap preserves official PostgreSQL/Nginx entrypoints and normalizes every
// child output line. Child stop signals are forwarded without echoing arguments.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
)

func main() {
	service := "infrastructure"
	if len(os.Args) > 1 {
		service = os.Args[1]
	}
	if err := run(os.Args[1:]); err != nil {
		telemetry.Logger(service, "process").Error("Infrastructure process failed", "event", "process.failed", "error", err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 || (args[0] != "database" && args[0] != "dashboard") {
		return fmt.Errorf("a supported service and entrypoint are required")
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("process log pipe unavailable")
	}
	defer reader.Close()
	defer writer.Close()
	child := exec.Command(args[1], args[2:]...)
	child.Stdout, child.Stderr = writer, writer
	if err := child.Start(); err != nil {
		return fmt.Errorf("cannot start infrastructure process")
	}
	_ = writer.Close()
	signals := make(chan os.Signal, 4)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	defer signal.Stop(signals)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case stop := <-signals:
				_ = child.Process.Signal(stop)
			case <-done:
				return
			}
		}
	}()
	if err := forwardLogs(reader, os.Stdout, args[0], credentialValues(os.Environ())); err != nil {
		_ = child.Process.Kill()
		_ = child.Wait()
		return fmt.Errorf("process log forwarding failed")
	}
	if err := child.Wait(); err != nil {
		return fmt.Errorf("infrastructure process exited unsuccessfully")
	}
	return nil
}

func credentialValues(environment []string) []string {
	var values []string
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if ok && value != "" && (strings.HasSuffix(key, "_PASSWORD") || strings.HasSuffix(key, "_KEY") || strings.HasSuffix(key, "_SECRET") || strings.HasSuffix(key, "_TOKEN") || key == "DATABASE_URL") {
			values = append(values, value)
			quoted, _ := json.Marshal(value)
			values = append(values, string(quoted[1:len(quoted)-1]))
		}
	}
	return values
}

func forwardLogs(reader io.Reader, writer io.Writer, service string, secrets []string) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	encoder := json.NewEncoder(writer)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		for _, secret := range secrets {
			if secret != "" {
				line = strings.ReplaceAll(line, secret, "[REDACTED]")
			}
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &fields); err != nil || fields == nil {
			message, _ := json.Marshal(line)
			fields = map[string]json.RawMessage{"msg": message}
		}
		fields["service"], _ = json.Marshal(service)
		fields["run_id"], fields["node_id"], fields["attempt"] = json.RawMessage("null"), json.RawMessage("null"), json.RawMessage("null")
		if _, ok := fields["event"]; !ok {
			fields["event"] = json.RawMessage(`"process.output"`)
		}
		if _, ok := fields["level"]; !ok {
			level := "INFO"
			upper := strings.ToUpper(line)
			if strings.Contains(upper, "ERROR:") || strings.Contains(upper, "FATAL:") || strings.Contains(upper, "PANIC:") || strings.Contains(upper, "[ERROR]") || strings.Contains(upper, "[EMERG]") {
				level = "ERROR"
			} else if strings.Contains(upper, "WARNING:") || strings.Contains(upper, "[WARN]") {
				level = "WARN"
			}
			fields["level"], _ = json.Marshal(level)
		}
		if _, ok := fields["time"]; !ok {
			if _, ok := fields["ts"]; !ok {
				fields["time"], _ = json.Marshal(time.Now().UTC())
			}
		}
		if err := encoder.Encode(fields); err != nil {
			return err
		}
	}
	return scanner.Err()
}
