-- +goose Up
-- One row per ingest command run, for the admin data-status page.
CREATE TABLE IF NOT EXISTS upstonk.ingest_runs (
    id          bigserial PRIMARY KEY,
    kind        text NOT NULL,          -- jse-list | prices | factsheets
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    stored      int NOT NULL DEFAULT 0,
    failed      jsonb NOT NULL DEFAULT '{}'::jsonb   -- code -> reason
);

-- +goose Down
DROP TABLE IF EXISTS upstonk.ingest_runs;
