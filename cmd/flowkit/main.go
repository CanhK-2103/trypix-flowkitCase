package main

import (
	"context"
	"fmt"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/api"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/artifacts"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/config"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/gate"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/jobs"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/providers"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		telemetry.Logger("process", "main").Error("Process failed", "event", "process.failed", "error", err.Error())
		os.Exit(1)
	}
}
func run() error {
	mode := "api"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if e := config.LoadDotEnv(".env"); e != nil {
		return e
	}
	cfg := config.Load()
	log := telemetry.Logger(mode, uuid.NewString())
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if mode == "gate" {
		handler, err := gate.New(gate.Config{PasswordHash: os.Getenv("DASHBOARD_PASSWORD_HASH"), SessionKey: os.Getenv("DASHBOARD_SESSION_KEY"), Upstream: config.Env("GATE_UPSTREAM_URL", "http://dashboard:80")}, log)
		if err != nil {
			return err
		}
		return serveHTTP(ctx, cfg.Address, handler, log, "gate")
	}
	pool, e := pgxpool.New(ctx, cfg.DatabaseURL)
	if e != nil {
		return e
	}
	defer pool.Close()
	if e := pool.Ping(ctx); e != nil {
		return e
	}
	store := storage.New(pool)
	switch mode {
	case "init":
		e = storage.Migrate(ctx, pool, log)
		if e == nil {
			log.Info("Migrations complete", "event", "migration.completed")
		}
		return e
	case "worker":
		media := artifacts.New(cfg.ArtifactDir)
		workerEngine := &engine.Engine{Store: store, Provider: providers.New(cfg), Media: media, Config: cfg, Log: log}
		client, e := jobs.New(pool, workerEngine, log, cfg.Concurrency)
		if e != nil {
			return e
		}
		if e := client.Start(ctx); e != nil {
			return e
		}
		go jobs.Reconcile(ctx, store, workerEngine, log)
		log.Info("Worker started", "event", "worker.started", "concurrency", cfg.Concurrency)
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		return client.Stop(shutdown)
	case "api":
		queue, e := jobs.New(pool, nil, log, 0)
		if e != nil {
			return e
		}
		handler := (&api.Server{Store: store, Queue: queue, Config: cfg, Log: log}).Handler()
		return serveHTTP(ctx, cfg.Address, handler, log, "api")
	default:
		return fmt.Errorf("unknown mode %q; use init, api, worker or gate", mode)
	}
}

func serveHTTP(ctx context.Context, address string, handler http.Handler, log *slog.Logger, service string) error {
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second, ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelError)}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Info("HTTP service started", "event", service+".started", "address", address)
	select {
	case err := <-done:
		if err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}
