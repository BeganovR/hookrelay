BEGIN;

CREATE INDEX idx_events_received_at ON events(received_at DESC);
CREATE INDEX idx_deliveries_event_id ON deliveries(event_id);
CREATE INDEX idx_deliveries_subscription_id ON deliveries(subscription_id);
CREATE INDEX idx_delivery_attempts_delivery_id ON delivery_attempts(delivery_id);

COMMIT;
