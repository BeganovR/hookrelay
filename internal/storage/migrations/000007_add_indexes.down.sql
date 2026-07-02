BEGIN;

DROP INDEX IF EXISTS idx_events_received_at;
DROP INDEX IF EXISTS idx_deliveries_event_id;
DROP INDEX IF EXISTS idx_deliveries_subscription_id;
DROP INDEX IF EXISTS idx_delivery_attempts_delivery_id;

COMMIT;
