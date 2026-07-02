INSERT INTO events (source_id, idempotency_key, headers, payload, payload_size, sender_ip, received_at)
SELECT
    s.id,
    md5(s.id::text || n.i::text),
    '{"Content-Type": "application/json"}'::jsonb,
    ('{"action": "push", "index": ' || n.i || '}')::jsonb,
    200 + (n.i % 800),
    ('93.184.' || n.i % 256 || '.' || (n.i + 1) % 256)::inet,
    now() - (n.i || ' hours')::interval
FROM (SELECT id FROM sources LIMIT :seed_count) s
CROSS JOIN generate_series(1, 20) AS n(i)
ON CONFLICT (source_id, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING;
