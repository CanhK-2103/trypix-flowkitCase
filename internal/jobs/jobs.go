package jobs

import (
	"context"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/engine"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"log/slog"
	"time"
)

type RunArgs struct {
	RunID string `json:"run_id"`
}

func (RunArgs) Kind() string { return "flowkit_run" }

type Worker struct {
	river.WorkerDefaults[RunArgs]
	Engine *engine.Engine
}

func (w *Worker) Work(ctx context.Context, job *river.Job[RunArgs]) error {
	delay, e := w.Engine.Step(ctx, job.Args.RunID)
	if e != nil {
		return e
	}
	if delay > 0 {
		return river.JobSnooze(delay)
	}
	return nil
}
func New(pool *pgxpool.Pool, e *engine.Engine, log *slog.Logger, concurrency int) (*river.Client[pgx.Tx], error) {
	cfg := &river.Config{Logger: log, MaxAttempts: 5, JobTimeout: 45 * time.Second, RescueStuckJobsAfter: 90 * time.Second, SoftStopTimeout: 35 * time.Second, CompletedJobRetentionPeriod: -1, DiscardedJobRetentionPeriod: -1, CancelledJobRetentionPeriod: -1}
	if e != nil {
		workers := river.NewWorkers()
		river.AddWorker(workers, &Worker{Engine: e})
		cfg.Workers = workers
		cfg.Queues = map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: concurrency}}
	}
	return river.NewClient(riverpgxv5.New(pool), cfg)
}
func Reconcile(ctx context.Context, s *storage.Store, e *engine.Engine, log *slog.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ReconcileBilling(ctx, s, e); err != nil {
				log.Error("Billing reconciliation failed", "event", "billing.reconcile_error", "error", err.Error())
			}
			// River owns its schema. This read-only integration query detects jobs that its maintenance discarded.
			rows, err := s.Pool.Query(ctx, "SELECT r.data FROM runs r JOIN river_job j ON j.id=(r.data->>'job_id')::bigint WHERE r.status IN ('queued','running') AND j.state IN ('discarded','cancelled') LIMIT 100")
			if err != nil {
				log.Error("Run reconciliation query failed", "event", "reconcile.error", "error", err.Error())
				continue
			}
			var runs []*domain.Run
			for rows.Next() {
				var data []byte
				if rows.Scan(&data) == nil {
					var r domain.Run
					if domain.Decode(data, &r) == nil {
						runs = append(runs, &r)
					}
				}
			}
			rows.Close()
			for _, r := range runs {
				if err := e.FailInfrastructure(ctx, r); err != nil {
					log.Error("Run reconciliation failed", "event", "reconcile.error", "run_id", r.ID, "error", err.Error())
				}
			}
		}
	}
}

// Due checks are persisted in each request. Multiple workers may read the same
// accounting record, but optimistic checkpoints prevent overwriting newer data.
func ReconcileBilling(ctx context.Context, s *storage.Store, e *engine.Engine) error {
	if e.Config.BillingDisabled {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rows, err := s.Q.ListBillingDue(ctx, 25)
	if err != nil {
		return err
	}
	for _, id := range rows {
		if err := e.ReconcileCosts(ctx, id.String()); err != nil {
			return err
		}
	}
	return nil
}
