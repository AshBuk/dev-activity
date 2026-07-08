# dev-activity

One-shot collector of GitHub activity. Fetches recent events for a user from
the GitHub REST API (`/users/{user}/events`) and stores them in PostgreSQL.
Runs as a Kubernetes CronJob in my [homelab](https://github.com/AshBuk/homelab);
Grafana reads the data straight from PostgreSQL.

## Design

- **One-shot process, no daemon.** Fetch → upsert → exit. Scheduling belongs
  to Kubernetes (CronJob).
- **Raw events, aggregate at read time.** Events are stored append-only with
  the original payload as `jsonb`. Commits per day, PR counts, streaks are
  SQL queries, so new charts never require code changes.
- **Idempotent by construction.** The schema is `CREATE ... IF NOT EXISTS`
  and applied on every run; inserts are `ON CONFLICT DO NOTHING`. Running
  the job twice is safe.

The events API keeps at most 300 events / 90 days of history — regular
polling is what builds the long-term record.

## Configuration

| Env var        | Required | Description                                          |
| -------------- | -------- | ---------------------------------------------------- |
| `GITHUB_USER`  | yes      | GitHub login to collect activity for                 |
| `DATABASE_URL` | yes      | PostgreSQL connection string                         |
| `GITHUB_TOKEN` | no       | Raises rate limits and includes private events       |

## Run locally

```bash
GITHUB_USER=AshBuk \
DATABASE_URL=postgres://app:secret@localhost:5432/app \
go run ./cmd/dev-activity
```

## Deploy

Container image: `ghcr.io/ashbuk/dev-activity` (built by GitHub Actions on
push to `main`). Kubernetes manifests live in
[homelab/apps/dev-activity](https://github.com/AshBuk/homelab/tree/main/apps/dev-activity).
