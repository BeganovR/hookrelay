package storage

import (
	"context"
	"errors"
	"time"

	"hookrelay/internal/domain"

	"github.com/jackc/pgx/v5"
)

type CreateDeliveryParams struct {
	ProjectID      string
	EventID        string
	EndpointID     string
	SubscriptionID *string
	MaxRetries     int
	RetryInterval  int
	RetryType      string
}

func (db *DB) CreateDelivery(ctx context.Context, p CreateDeliveryParams) (*domain.EventDelivery, error) {
	var d domain.EventDelivery
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO deliveries
			(project_id, event_id, endpoint_id, subscription_id, max_retries, retry_interval, retry_type, next_retry_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, project_id, event_id, endpoint_id, subscription_id,
		          status, attempts_count, max_retries, retry_interval, retry_type, next_retry_at, created_at
	`, p.ProjectID, p.EventID, p.EndpointID, p.SubscriptionID,
		p.MaxRetries, p.RetryInterval, p.RetryType).
		Scan(&d.ID, &d.ProjectID, &d.EventID, &d.EndpointID, &d.SubscriptionID,
			&d.Status, &d.RetryCount, &d.MaxRetries, &d.RetryInterval, &d.RetryType,
			&d.ScheduledAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (db *DB) GetDelivery(ctx context.Context, projectID, id string) (*domain.EventDelivery, error) {
	var d domain.EventDelivery
	err := db.Pool.QueryRow(ctx, `
		SELECT id, project_id, event_id, endpoint_id, subscription_id,
		       status, attempts_count, max_retries, retry_interval, retry_type, next_retry_at, created_at
		FROM deliveries WHERE id = $1 AND project_id = $2
	`, id, projectID).
		Scan(&d.ID, &d.ProjectID, &d.EventID, &d.EndpointID, &d.SubscriptionID,
			&d.Status, &d.RetryCount, &d.MaxRetries, &d.RetryInterval, &d.RetryType,
			&d.ScheduledAt, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (db *DB) ListDeliveriesForEvent(ctx context.Context, projectID, eventID string) ([]domain.EventDelivery, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, project_id, event_id, endpoint_id, subscription_id,
		       status, attempts_count, max_retries, retry_interval, retry_type, next_retry_at, created_at
		FROM deliveries WHERE project_id = $1 AND event_id = $2
		ORDER BY created_at ASC
	`, projectID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.EventDelivery
	for rows.Next() {
		var d domain.EventDelivery
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.EventID, &d.EndpointID, &d.SubscriptionID,
			&d.Status, &d.RetryCount, &d.MaxRetries, &d.RetryInterval, &d.RetryType,
			&d.ScheduledAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (db *DB) ClaimPendingDeliveries(ctx context.Context, batchSize int) ([]domain.PendingDelivery, error) {
	rows, err := db.Pool.Query(ctx, `
		WITH claimed AS (
			SELECT ed.id
			FROM deliveries ed
			WHERE ed.status = 'pending' AND ed.next_retry_at <= NOW()
			ORDER BY ed.next_retry_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		),
		updated AS (
			UPDATE deliveries ed
			SET status = 'processing', next_retry_at = NOW()
			FROM claimed
			WHERE ed.id = claimed.id
			RETURNING ed.id, ed.project_id, ed.event_id, ed.endpoint_id, ed.subscription_id,
			          ed.status, ed.attempts_count, ed.max_retries, ed.retry_interval, ed.retry_type,
			          ed.next_retry_at, ed.created_at
		)
		SELECT u.id, u.project_id, u.event_id, u.endpoint_id, u.subscription_id,
		       u.status, u.attempts_count, u.max_retries, u.retry_interval, u.retry_type,
		       u.next_retry_at, u.created_at,
		       e.payload, e.headers,
		       ep.url, ep.http_timeout, ep.auth_type, ep.auth_cfg,
		       COALESCE(sub.secret, '')
		FROM updated u
		JOIN events e      ON e.id  = u.event_id
		JOIN endpoints ep  ON ep.id = u.endpoint_id
		LEFT JOIN subscriptions sub ON sub.id = u.subscription_id
	`, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.PendingDelivery
	for rows.Next() {
		var pd domain.PendingDelivery
		var payBytes, hdrBytes, authCfgBytes []byte
		if err := rows.Scan(
			&pd.ID, &pd.ProjectID, &pd.EventID, &pd.EndpointID, &pd.SubscriptionID,
			&pd.Status, &pd.RetryCount, &pd.MaxRetries, &pd.RetryInterval, &pd.RetryType,
			&pd.ScheduledAt, &pd.CreatedAt,
			&payBytes, &hdrBytes,
			&pd.EndpointURL, &pd.EndpointTimeout, &pd.EndpointAuthType, &authCfgBytes,
			&pd.SigningSecret,
		); err != nil {
			return nil, err
		}
		pd.EventPayload = payBytes
		pd.EventHeaders = hdrBytes
		pd.EndpointAuthCfg = authCfgBytes
		result = append(result, pd)
	}
	return result, rows.Err()
}

func (db *DB) MarkDeliverySuccess(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE deliveries SET status = 'success' WHERE id = $1`, id)
	return err
}

func (db *DB) MarkDeliveryDiscarded(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE deliveries SET status = 'discarded' WHERE id = $1`, id)
	return err
}

func (db *DB) ResetDeliveryForRetry(ctx context.Context, projectID, id string) error {
	res, err := db.Pool.Exec(ctx, `
		UPDATE deliveries
		SET status = 'pending', next_retry_at = NOW()
		WHERE id = $1 AND project_id = $2 AND status = 'discarded'
	`, id, projectID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (db *DB) ScheduleDeliveryRetry(ctx context.Context, id string, scheduledAt time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE deliveries
		SET status = 'pending', attempts_count = attempts_count + 1, next_retry_at = $2
		WHERE id = $1
	`, id, scheduledAt)
	return err
}

func (db *DB) RecoverStuckDeliveries(ctx context.Context, stuckBefore time.Time) (int64, error) {
	tag, err := db.Pool.Exec(ctx, `
		UPDATE deliveries SET status = 'pending'
		WHERE status = 'processing' AND next_retry_at < $1
	`, stuckBefore)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (db *DB) CreateDeliveryAttempt(ctx context.Context, deliveryID string, statusCode *int, durationMs *int, responseBody *string, errMsg *string) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO delivery_attempts (delivery_id, attempt_number, status, http_code, response_time_ms, response_body, error)
		SELECT $1,
		       COALESCE((SELECT MAX(attempt_number) FROM delivery_attempts WHERE delivery_id = $1), 0) + 1,
		       CASE
		           WHEN $2::int IS NOT NULL AND $2::int BETWEEN 200 AND 299 THEN 'success'
		           WHEN $5::text IS NOT NULL AND ($5::text ILIKE '%timeout%' OR $5::text ILIKE '%deadline exceeded%') THEN 'timeout'
		           ELSE 'fail'
		       END,
		       $2, COALESCE($3, 0), $4, $5
	`, deliveryID, statusCode, durationMs, responseBody, errMsg)
	return err
}

func (db *DB) ListDeliveryAttempts(ctx context.Context, deliveryID string) ([]domain.DeliveryAttempt, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, delivery_id, http_code, response_body, error, response_time_ms, created_at
		FROM delivery_attempts WHERE delivery_id = $1
		ORDER BY created_at ASC
	`, deliveryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.DeliveryAttempt
	for rows.Next() {
		var a domain.DeliveryAttempt
		if err := rows.Scan(&a.ID, &a.DeliveryID, &a.StatusCode, &a.ResponseBody,
			&a.Error, &a.DurationMs, &a.AttemptedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
