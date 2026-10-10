package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/db/migrations"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
	"github.com/CanhK-2103/trypix-flowkitCase/internal/storage/dbgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

type Store struct {
	Pool *pgxpool.Pool
	Q    *dbgen.Queries
}

func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool, Q: dbgen.New(pool)} }
func UUID(s string) pgtype.UUID {
	v, e := uuid.Parse(s)
	if e != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: v, Valid: true}
}
func Timestamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t.UTC(), Valid: true} }
func Numeric(s *string) (pgtype.Numeric, error) {
	n := pgtype.Numeric{}
	if s != nil {
		if e := n.Scan(*s); e != nil {
			return n, e
		}
	}
	return n, nil
}
func Unique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
func (s *Store) LoadRun(ctx context.Context, id string) (*domain.Run, error) {
	row, e := s.Q.GetRun(ctx, UUID(id))
	if e != nil {
		return nil, e
	}
	var r domain.Run
	if e := domain.Decode(row.Data, &r); e != nil {
		return nil, e
	}
	r.Version = row.Version
	return &r, nil
}
func (s *Store) SaveRun(ctx context.Context, r *domain.Run) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	q := s.Q.WithTx(tx)
	version := r.Version
	r.Version++
	rows, e := q.SaveRun(ctx, dbgen.SaveRunParams{ID: UUID(r.ID), Status: r.Status, Data: domain.JSON(r), ExpectedVersion: version})
	if e != nil || rows != 1 {
		r.Version = version
		if e != nil {
			return e
		}
		return fmt.Errorf("run checkpoint conflict")
	}
	if e := s.SyncNodes(ctx, q, r); e != nil {
		r.Version = version
		return e
	}
	if e := tx.Commit(ctx); e != nil {
		r.Version = version
		return e
	}
	return nil
}
func (s *Store) SyncNodes(ctx context.Context, q *dbgen.Queries, r *domain.Run) error {
	for i, n := range r.Nodes {
		if e := q.UpsertNode(ctx, dbgen.UpsertNodeParams{RunID: UUID(r.ID), NodeID: n.Node.ID, Position: int32(i), Status: n.Status, Data: domain.JSON(n)}); e != nil {
			return e
		}
		for _, a := range n.Attempts {
			if e := q.UpsertAttempt(ctx, dbgen.UpsertAttemptParams{RunID: UUID(r.ID), NodeID: n.Node.ID, Attempt: int32(a.Number), Data: domain.JSON(a)}); e != nil {
				return e
			}
		}
		for _, p := range n.Requests {
			estimated, e := Numeric(p.Cost.Estimated)
			if e != nil {
				return e
			}
			reported, e := Numeric(p.Cost.Reported)
			if e != nil {
				return e
			}
			if e := q.UpsertProviderRequest(ctx, dbgen.UpsertProviderRequestParams{ID: UUID(p.ID), RunID: UUID(r.ID), NodeID: n.Node.ID, ProviderRequestID: pgtype.Text{String: p.ProviderRequestID, Valid: p.ProviderRequestID != ""}, Phase: p.Phase, EstimatedCostUsd: estimated, ReportedCostUsd: reported, Data: domain.JSON(p)}); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Store) SaveArtifact(ctx context.Context, a domain.Artifact) error {
	return s.Q.SaveArtifact(ctx, dbgen.SaveArtifactParams{ID: UUID(a.ID), RunID: UUID(a.RunID), Filename: a.FileName, Data: domain.JSON(a)})
}
func (s *Store) Artifact(ctx context.Context, id string) (domain.Artifact, error) {
	row, e := s.Q.GetArtifact(ctx, UUID(id))
	if e != nil {
		return domain.Artifact{}, e
	}
	var a domain.Artifact
	e = domain.Decode(row.Data, &a)
	a.FileName = row.Filename
	return a, e
}
func (s *Store) ModelMap(ctx context.Context, q *dbgen.Queries, definition domain.Definition, lock bool) (map[string]domain.Model, error) {
	models := map[string]domain.Model{}
	for _, n := range definition.Nodes {
		if n.ModelRef == "" {
			continue
		}
		if _, ok := models[n.ModelRef]; ok {
			continue
		}
		var data json.RawMessage
		var e error
		if lock {
			data, e = q.LockModel(ctx, n.ModelRef)
		} else {
			data, e = q.GetModel(ctx, n.ModelRef)
		}
		if errors.Is(e, pgx.ErrNoRows) {
			continue
		}
		if e != nil {
			return nil, e
		}
		var m domain.Model
		if e := domain.Decode(data, &m); e != nil {
			return nil, e
		}
		models[m.Alias] = m
	}
	return models, nil
}
func Migrate(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	conn, e := pool.Acquire(ctx)
	if e != nil {
		return e
	}
	defer conn.Release()
	if _, e := conn.Exec(ctx, "SELECT pg_advisory_lock(734026889)"); e != nil {
		return e
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock(734026889)")
	rm, e := rivermigrate.New(riverpgxv5.New(pool), &rivermigrate.Config{Logger: log})
	if e != nil {
		return e
	}
	if _, e := rm.Migrate(ctx, rivermigrate.DirectionUp, nil); e != nil {
		return e
	}
	goose.SetBaseFS(migrations.Files)
	goose.SetLogger(&migrationLogger{log: log})
	if e := goose.SetDialect("postgres"); e != nil {
		return e
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return goose.UpContext(ctx, db, ".")
}

type migrationLogger struct{ log *slog.Logger }

func (l *migrationLogger) Printf(format string, args ...any) {
	l.log.Info(fmt.Sprintf(format, args...), "event", "migration.progress")
}
func (l *migrationLogger) Fatalf(format string, args ...any) {
	l.log.Error(fmt.Sprintf(format, args...), "event", "migration.failed")
}
