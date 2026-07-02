-- name: events_last_hour
SELECT e.id, e.source_id, e.payload_size, e.sender_ip, e.received_at
FROM events e
WHERE e.received_at >= now() - interval '2 hours'
ORDER BY e.received_at DESC
LIMIT 100;

-- name: pending_deliveries
SELECT d.id, d.event_id, d.attempts_count, d.next_retry_at
FROM deliveries d
WHERE d.subscription_id = (SELECT id FROM subscriptions ORDER BY id LIMIT 1)
  AND d.status = 'pending'
ORDER BY d.next_retry_at ASC NULLS LAST
LIMIT 50;

-- name: delivery_success_rate
SELECT
    s.target_url,
    COUNT(*) FILTER (WHERE d.status = 'delivered') AS delivered,
    COUNT(*) FILTER (WHERE d.status = 'failed')    AS failed,
    COUNT(*)                                        AS total
FROM subscriptions s
JOIN deliveries d ON d.subscription_id = s.id
GROUP BY s.id, s.target_url
ORDER BY total DESC
LIMIT 20;

-- name: top_sources_by_volume
SELECT
    s.slug,
    COUNT(e.id)       AS event_count,
    MAX(e.received_at) AS last_event_at
FROM sources s
JOIN events e ON e.source_id = s.id
WHERE e.received_at >= now() - interval '12 hours'
GROUP BY s.id, s.slug
ORDER BY event_count DESC
LIMIT 10;

-- name: workspace_audit_log
SELECT al.id, al.user_id, al.action, al.ip_address, al.changes, al.created_at
FROM audit_logs al
WHERE al.workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1)
ORDER BY al.created_at DESC
LIMIT 50;

-- name: subscription_attempt_history
SELECT
    e.id              AS event_id,
    d.status          AS delivery_status,
    da.attempt_number,
    da.http_code,
    da.response_time_ms,
    da.created_at     AS attempted_at
FROM subscriptions sub
JOIN deliveries d       ON d.subscription_id = sub.id
JOIN delivery_attempts da ON da.delivery_id  = d.id
JOIN events e           ON e.id              = d.event_id
WHERE sub.id = (SELECT id FROM subscriptions ORDER BY id LIMIT 1)
ORDER BY da.created_at DESC
LIMIT 100;
