BEGIN;

ALTER TABLE subscriptions ALTER COLUMN source_id SET NOT NULL;

ALTER TABLE subscriptions DROP COLUMN IF EXISTS retry_type;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS retry_interval;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS max_retries;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS filter_cfg;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS updated_at;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS is_active;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS name;

COMMIT;
