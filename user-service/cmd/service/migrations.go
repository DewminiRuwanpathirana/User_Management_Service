package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func runMigrations(ctx context.Context, dbPool *pgxpool.Pool) error {
	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil {
		return err
	}

	sort.Strings(files)
	for _, path := range files {
		sqlBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		if _, execErr := dbPool.Exec(ctx, string(sqlBytes)); execErr != nil {
			return execErr
		}
	}

	return nil
}
