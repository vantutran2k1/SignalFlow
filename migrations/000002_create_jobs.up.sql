CREATE TABLE jobs
(
    id              UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    type            TEXT        NOT NULL CHECK (type IN ('http_check', 'command', 'rss_watch')),
    schedule        TEXT        NOT NULL,
    config          JSONB       NOT NULL DEFAULT '{}',
    status          TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'disabled')),
    notify_channels UUID[]      NOT NULL DEFAULT '{}',
    condition       JSONB       NOT NULL DEFAULT '{
      "on": "failure"
    }',
    last_run_at     TIMESTAMPTZ,
    next_run_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_user_id ON jobs (user_id);
CREATE INDEX idx_jobs_status ON jobs (status);
CREATE INDEX idx_jobs_next_run ON jobs (next_run_at) WHERE status = 'active';
