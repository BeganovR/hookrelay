BEGIN;

ALTER TABLE deliveries DROP COLUMN IF EXISTS project_id;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS project_id;

DROP INDEX IF EXISTS idx_events_project_idempotency;

ALTER TABLE events DROP COLUMN IF EXISTS event_type;
ALTER TABLE events DROP COLUMN IF EXISTS project_id;

COMMIT;
