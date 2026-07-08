// Copyright (c) 2026 Asher Buk
// SPDX-License-Identifier: MIT

// Package store persists GitHub events in PostgreSQL.
package store

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/AshBuk/dev-activity/internal/github"
)

// The schema is idempotent (CREATE ... IF NOT EXISTS) and applied on every
// run, so the job has no separate migration step to depend on.
//
//go:embed schema.sql
var schema string

type Store struct {
	conn *pgx.Conn
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(ctx, schema); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{conn: conn}, nil
}

func (s *Store) Close(ctx context.Context) {
	_ = s.conn.Close(ctx)
}

// InsertEvents stores events, silently skipping IDs that already exist:
// consecutive runs re-fetch overlapping pages by design. Returns the
// number of newly inserted rows.
func (s *Store) InsertEvents(ctx context.Context, events []github.Event) (int64, error) {
	batch := &pgx.Batch{}
	for _, e := range events {
		id, err := strconv.ParseInt(e.ID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("event id %q: %w", e.ID, err)
		}
		batch.Queue(
			`INSERT INTO github_events (id, type, repo, payload, created_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (id) DO NOTHING`,
			id, e.Type, e.Repo.Name, e.Payload, e.CreatedAt,
		)
	}

	results := s.conn.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()

	var inserted int64
	for range events {
		tag, err := results.Exec()
		if err != nil {
			return inserted, err
		}
		inserted += tag.RowsAffected()
	}
	return inserted, nil
}
