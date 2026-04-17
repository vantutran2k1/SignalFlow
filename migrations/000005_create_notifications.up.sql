CREATE TABLE notifications
(
    id           UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    execution_id UUID        NOT NULL REFERENCES executions (id) ON DELETE CASCADE,
    channel_id   UUID        NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    status       TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    payload      TEXT        NOT NULL DEFAULT '',
    error        TEXT        NOT NULL DEFAULT '',
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_execution_id ON notifications (execution_id);
CREATE INDEX idx_notifications_status ON notifications (status) WHERE status = 'pending';
