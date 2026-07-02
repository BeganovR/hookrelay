INSERT INTO subscriptions (source_id, target_url, secret)
SELECT
    s.id,
    'https://hooks.example.com/' || s.id || '/' || n.i,
    md5('sub-secret-' || s.id::text || n.i::text)
FROM (SELECT id FROM sources LIMIT :seed_count) s
CROSS JOIN generate_series(1, 3) AS n(i)
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions sub
    WHERE sub.source_id = s.id
      AND sub.target_url = 'https://hooks.example.com/' || s.id || '/' || n.i
);

INSERT INTO deliveries (event_id, subscription_id, status, attempts_count)
SELECT
    e.id,
    sub.id,
    (ARRAY['pending', 'delivered', 'failed'])[(ROW_NUMBER() OVER () % 3 + 1)::int],
    (ROW_NUMBER() OVER () % 5)::int
FROM events e
JOIN subscriptions sub ON sub.source_id = e.source_id
WHERE NOT EXISTS (
    SELECT 1 FROM deliveries d WHERE d.event_id = e.id AND d.subscription_id = sub.id
)
LIMIT 20 * :seed_count;

INSERT INTO delivery_attempts (delivery_id, attempt_number, status, http_code, response_body, response_time_ms)
SELECT
    d.id,
    s.i,
    CASE WHEN d.status = 'delivered' THEN 'success' ELSE 'failed' END,
    CASE WHEN d.status = 'delivered' THEN 200 ELSE 500 END,
    CASE WHEN d.status = 'delivered' THEN '{"ok":true}' ELSE '{"error":"timeout"}' END,
    150 + (s.i * 100)
FROM (SELECT id, status FROM deliveries LIMIT 10 * :seed_count) d
CROSS JOIN generate_series(1, 2) AS s(i)
WHERE NOT EXISTS (
    SELECT 1 FROM delivery_attempts da WHERE da.delivery_id = d.id AND da.attempt_number = s.i
);
