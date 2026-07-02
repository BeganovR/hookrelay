BEGIN;

ALTER TABLE events ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES workspaces(id);
ALTER TABLE events ADD COLUMN IF NOT EXISTS event_type VARCHAR(100);

CREATE UNIQUE INDEX IF NOT EXISTS idx_events_project_idempotency
    ON events (project_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES workspaces(id);
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES workspaces(id);

COMMIT;
