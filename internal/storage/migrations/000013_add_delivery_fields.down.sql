BEGIN;

ALTER TABLE deliveries ALTER COLUMN subscription_id SET NOT NULL;

ALTER TABLE deliveries DROP COLUMN IF EXISTS created_at;
ALTER TABLE deliveries DROP COLUMN IF EXISTS retry_type;
ALTER TABLE deliveries DROP COLUMN IF EXISTS retry_interval;
ALTER TABLE deliveries DROP COLUMN IF EXISTS max_retries;
ALTER TABLE deliveries DROP COLUMN IF EXISTS endpoint_id;

COMMIT;
