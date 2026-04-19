ALTER TABLE jobs ADD COLUMN timeout_seconds INT NOT NULL DEFAULT 30;

ALTER TABLE executions DROP CONSTRAINT executions_status_check;
ALTER TABLE executions ADD CONSTRAINT executions_status_check
    CHECK (status IN ('running', 'success', 'failure', 'error'));
ALTER TABLE executions ALTER COLUMN finished_at DROP NOT NULL;

CREATE INDEX idx_executions_running ON executions (started_at) WHERE status = 'running';