BEGIN;

ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Go's ingest layer does not currently capture the caller's IP; keep the
-- NOT NULL contract but let it default rather than requiring every INSERT to set it.
ALTER TABLE events ALTER COLUMN sender_ip SET DEFAULT '0.0.0.0'::inet;

COMMIT;
