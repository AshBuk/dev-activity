-- Append-only log of GitHub events. Aggregates (commits per day, streak)
-- are computed at read time with SQL, e.g. from Grafana.
CREATE TABLE IF NOT EXISTS github_events (
    id         bigint      PRIMARY KEY, -- GitHub event ID, globally unique
    type       text        NOT NULL,    -- PushEvent, PullRequestEvent, ...
    repo       text        NOT NULL,    -- owner/name
    payload    jsonb       NOT NULL,    -- raw event payload as returned by the API
    created_at timestamptz NOT NULL     -- when the event happened on GitHub
);

CREATE INDEX IF NOT EXISTS github_events_created_at_idx
    ON github_events (created_at);
