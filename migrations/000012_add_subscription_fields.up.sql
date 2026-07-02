BEGIN;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS name VARCHAR(100);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS filter_cfg JSONB;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS max_retries INT NOT NULL DEFAULT 5;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS retry_interval INT NOT NULL DEFAULT 5;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS retry_type VARCHAR(20) NOT NULL DEFAULT 'exponential';

-- Go allows subscriptions with no fixed source (applies to all sources in the project)
ALTER TABLE subscriptions ALTER COLUMN source_id DROP NOT NULL;

COMMIT;
