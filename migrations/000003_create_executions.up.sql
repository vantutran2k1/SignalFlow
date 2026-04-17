CREATE TABLE executions
(
    id          UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    job_id      UUID        NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    status      TEXT        NOT NULL CHECK (status IN ('success', 'failure', 'error')),
    output      TEXT        NOT NULL DEFAULT '',
    error       TEXT        NOT NULL DEFAULT '',
    duration_ms BIGINT      NOT NULL DEFAULT 0,
    started_at  TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_executions_job_id ON executions (job_id);
CREATE INDEX idx_executions_started_at ON executions (started_at DESC);
