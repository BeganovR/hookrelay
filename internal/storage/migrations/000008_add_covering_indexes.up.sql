BEGIN;

CREATE INDEX idx_deliveries_pending_retry
    ON deliveries(next_retry_at)
    WHERE status = 'pending';

COMMIT;
