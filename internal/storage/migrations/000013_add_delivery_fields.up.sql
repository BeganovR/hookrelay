BEGIN;

ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS endpoint_id UUID REFERENCES endpoints(id);
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS max_retries INT NOT NULL DEFAULT 5;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS retry_interval INT NOT NULL DEFAULT 5;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS retry_type VARCHAR(20) NOT NULL DEFAULT 'exponential';
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Go supports subscription-less (direct source->endpoint) deliveries
ALTER TABLE deliveries ALTER COLUMN subscription_id DROP NOT NULL;

COMMIT;
