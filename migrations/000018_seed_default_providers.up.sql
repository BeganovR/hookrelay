BEGIN;

INSERT INTO providers (name, signature_algorithm) VALUES
    ('github', 'hmac_sha256'),
    ('stripe', 'stripe_signature'),
    ('telegram', 'secret_token'),
    ('custom', NULL)
ON CONFLICT (name) DO NOTHING;

COMMIT;
