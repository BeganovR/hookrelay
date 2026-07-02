BEGIN;

DROP INDEX IF EXISTS idx_events_source_idempotency;
DROP TABLE IF EXISTS events;

COMMIT;