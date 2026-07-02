BEGIN;

CREATE TABLE IF NOT EXISTS providers (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL UNIQUE,
    signature_algorithm VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS sources (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id   UUID         NOT NULL REFERENCES workspaces(id),
    provider_id    UUID         NOT NULL REFERENCES providers(id),
    slug           VARCHAR(100) NOT NULL UNIQUE,
    signing_secret VARCHAR(255),
    is_active      BOOLEAN      NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMIT;