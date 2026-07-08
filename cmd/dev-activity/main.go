// Copyright (c) 2026 Asher Buk
// SPDX-License-Identifier: MIT

// dev-activity is a one-shot collector: it fetches recent GitHub activity
// for a user and stores it in PostgreSQL, then exits. It is designed to run
// as a Kubernetes CronJob (no daemon, no HTTP server).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/AshBuk/dev-activity/internal/github"
	"github.com/AshBuk/dev-activity/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("run failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	user := os.Getenv("GITHUB_USER")
	dsn := os.Getenv("DATABASE_URL")
	if user == "" || dsn == "" {
		return fmt.Errorf("GITHUB_USER and DATABASE_URL must be set")
	}
	// Optional: raises the rate limit and includes private events.
	token := os.Getenv("GITHUB_TOKEN")

	// Hard deadline for the whole run; the CronJob also enforces
	// activeDeadlineSeconds, this is the in-process safety net.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	events, err := github.NewClient(token).Events(ctx, user)
	if err != nil {
		return fmt.Errorf("fetch events: %w", err)
	}

	st, err := store.Open(ctx, dsn)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close(ctx)

	inserted, err := st.InsertEvents(ctx, events)
	if err != nil {
		return fmt.Errorf("insert events: %w", err)
	}

	slog.Info("done", "fetched", len(events), "new", inserted)
	return nil
}
