DROP INDEX IF EXISTS idx_executions_running;

ALTER TABLE executions ALTER COLUMN finished_at SET NOT NULL;
ALTER TABLE executions DROP CONSTRAINT executions_status_check;
ALTER TABLE executions ADD CONSTRAINT executions_status_check
    CHECK (status IN ('success', 'failure', 'error'));

ALTER TABLE jobs DROP COLUMN timeout_seconds;