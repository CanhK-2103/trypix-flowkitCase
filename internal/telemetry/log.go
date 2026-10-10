package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"
)

func Logger(service, instance string) *slog.Logger {
	level := slog.LevelDebug
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		_ = level.UnmarshalText([]byte(v))
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level, ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			a.Value = slog.TimeValue(a.Value.Time().UTC())
		}
		return a
	}})
	return slog.New(&contextHandler{Handler: handler}).With("service", service, "instance_id", instance)
}

type contextHandler struct {
	slog.Handler
	keys map[string]bool
}

func (h *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	found := map[string]bool{}
	for key := range h.keys {
		found[key] = true
	}
	record.Attrs(func(a slog.Attr) bool { found[a.Key] = true; return true })
	for _, key := range []string{"run_id", "node_id", "attempt"} {
		if !found[key] {
			record.AddAttrs(slog.Any(key, nil))
		}
	}
	return h.Handler.Handle(ctx, record)
}
func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	keys := map[string]bool{}
	for key := range h.keys {
		keys[key] = true
	}
	for _, attr := range attrs {
		keys[attr.Key] = true
	}
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs), keys: keys}
}
func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name), keys: h.keys}
}
func Duration(start time.Time) int64 { return time.Since(start).Milliseconds() }
