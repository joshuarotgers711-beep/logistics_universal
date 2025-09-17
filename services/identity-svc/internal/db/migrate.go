package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Migrate(ctx context.Context, d *DB) error {
	migDir := os.Getenv("MIGRATIONS_DIR")
	if migDir == "" { migDir = "migrations" }
	if _, err := os.Stat(migDir); errors.Is(err, os.ErrNotExist) { return nil }
	_, err := d.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now())`)
	if err != nil { return err }
	applied := map[string]bool{}
	rows, err := d.Pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err == nil {
		for rows.Next() { var v string; rows.Scan(&v); applied[v] = true }
		rows.Close()
	}
	files := []string{}
	filepath.WalkDir(migDir, func(path string, de fs.DirEntry, err error) error {
		if err != nil { return err }
		if de.IsDir() { return nil }
		if strings.HasSuffix(de.Name(), ".sql") { files = append(files, path) }
		return nil
	})
	sort.Strings(files)
	for _, f := range files {
		ver := filepath.Base(f)
		if applied[ver] { continue }
		b, err := os.ReadFile(f)
		if err != nil { return err }
		if _, err := d.Pool.Exec(ctx, string(b)); err != nil { return fmt.Errorf("migration %s failed: %w", ver, err) }
		_, _ = d.Pool.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, ver)
	}
	return nil
}

