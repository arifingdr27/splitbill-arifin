package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

//go:embed sql/*.up.sql
var upFS embed.FS

// Runner applies versioned SQL migrations embedded in the binary.
type Runner struct {
	pool *pgxpool.Pool
	log  *logrus.Logger
}

func New(pool *pgxpool.Pool, log *logrus.Logger) *Runner {
	return &Runner{pool: pool, log: log}
}

// Up applies all pending *.up.sql migrations in lexical order.
func (r *Runner) Up(ctx context.Context) error {
	if err := r.ensureTable(ctx); err != nil {
		return err
	}

	files, err := listUpFiles()
	if err != nil {
		return err
	}

	applied := 0
	for _, file := range files {
		version := versionFromUpFile(file)
		ok, err := r.isApplied(ctx, version)
		if err != nil {
			return err
		}
		if ok {
			r.log.WithField("version", version).Debug("migration already applied")
			continue
		}

		body, err := upFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations(version) VALUES ($1)`, version,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}

		applied++
		r.log.WithField("version", version).Info("migration applied")
	}

	if applied == 0 {
		r.log.Info("migrations: nothing pending")
	} else {
		r.log.WithField("count", applied).Info("migrations: finished")
	}
	return nil
}

func (r *Runner) ensureTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func (r *Runner) isApplied(ctx context.Context, version string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
	).Scan(&exists)
	return exists, err
}

func listUpFiles() ([]string, error) {
	entries, err := fs.ReadDir(upFS, "sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, "sql/"+name)
	}
	sort.Strings(files)
	return files, nil
}

func versionFromUpFile(path string) string {
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	return strings.TrimSuffix(base, ".up.sql")
}
