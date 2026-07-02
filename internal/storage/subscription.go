package storage

import (
	"context"
	"errors"
	"hookrelay/internal/domain"

	"github.com/jackc/pgx/v5"
)

type CreateSubscriptionParams struct {
	ProjectID     string
	Name          string
	SourceID      *string
	EndpointID    string
	MaxRetries    int
	RetryInterval int
	RetryType     string
	FilterCfg     []byte
	SigningSecret string
}

func (db *DB) CreateSubscription(ctx context.Context, p CreateSubscriptionParams) (*domain.Subscription, error) {
	if p.MaxRetries == 0 {
		p.MaxRetries = 5
	}
	if p.RetryInterval == 0 {
		p.RetryInterval = 5
	}
	if p.RetryType == "" {
		p.RetryType = domain.RetryExponential
	}
	var s domain.Subscription
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO subscriptions (project_id, name, source_id, endpoint_id, target_url, max_retries, retry_interval, retry_type, filter_cfg, secret)
		VALUES ($1, $2, $3, $4, (SELECT url FROM endpoints WHERE id = $4), $5, $6, $7, $8::jsonb, $9)
		RETURNING id, project_id, name, source_id, endpoint_id, max_retries, retry_interval, retry_type,
		          filter_cfg, is_active, COALESCE(secret, ''), created_at, updated_at
	`, p.ProjectID, p.Name, p.SourceID, p.EndpointID, p.MaxRetries, p.RetryInterval, p.RetryType, p.FilterCfg, p.SigningSecret).
		Scan(&s.ID, &s.ProjectID, &s.Name, &s.SourceID, &s.EndpointID, &s.MaxRetries, &s.RetryInterval, &s.RetryType,
			&cfgBytes, &s.IsActive, &s.SigningSecret, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.FilterCfg = cfgBytes
	return &s, nil
}

func (db *DB) GetSubscription(ctx context.Context, projectID, id string) (*domain.Subscription, error) {
	var s domain.Subscription
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, project_id, name, source_id, endpoint_id, max_retries, retry_interval, retry_type,
		       filter_cfg, is_active, COALESCE(secret, ''), created_at, updated_at
		FROM subscriptions WHERE id = $1 AND project_id = $2
	`, id, projectID).
		Scan(&s.ID, &s.ProjectID, &s.Name, &s.SourceID, &s.EndpointID, &s.MaxRetries, &s.RetryInterval, &s.RetryType,
			&cfgBytes, &s.IsActive, &s.SigningSecret, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s.FilterCfg = cfgBytes
	return &s, nil
}

func (db *DB) ListSubscriptions(ctx context.Context, projectID string) ([]domain.Subscription, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, project_id, name, source_id, endpoint_id, max_retries, retry_interval, retry_type,
		       filter_cfg, is_active, COALESCE(secret, ''), created_at, updated_at
		FROM subscriptions WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Subscription
	for rows.Next() {
		var s domain.Subscription
		var cfgBytes []byte
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.SourceID, &s.EndpointID, &s.MaxRetries, &s.RetryInterval, &s.RetryType,
			&cfgBytes, &s.IsActive, &s.SigningSecret, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.FilterCfg = cfgBytes
		result = append(result, s)
	}
	return result, rows.Err()
}

func (db *DB) ListActiveSubscriptionsForSource(ctx context.Context, projectID string, sourceID *string) ([]domain.Subscription, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, project_id, name, source_id, endpoint_id, max_retries, retry_interval, retry_type,
		       filter_cfg, is_active, COALESCE(secret, ''), created_at, updated_at
		FROM subscriptions
		WHERE project_id = $1
		  AND is_active = TRUE
		  AND (source_id = $2 OR source_id IS NULL)
	`, projectID, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Subscription
	for rows.Next() {
		var s domain.Subscription
		var cfgBytes []byte
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.SourceID, &s.EndpointID, &s.MaxRetries, &s.RetryInterval, &s.RetryType,
			&cfgBytes, &s.IsActive, &s.SigningSecret, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.FilterCfg = cfgBytes
		result = append(result, s)
	}
	return result, rows.Err()
}

func (db *DB) UpdateSubscription(ctx context.Context, projectID, id string, isActive bool) (*domain.Subscription, error) {
	var s domain.Subscription
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		UPDATE subscriptions SET is_active = $3, updated_at = NOW()
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, name, source_id, endpoint_id, max_retries, retry_interval, retry_type,
		          filter_cfg, is_active, COALESCE(secret, ''), created_at, updated_at
	`, id, projectID, isActive).
		Scan(&s.ID, &s.ProjectID, &s.Name, &s.SourceID, &s.EndpointID, &s.MaxRetries, &s.RetryInterval, &s.RetryType,
			&cfgBytes, &s.IsActive, &s.SigningSecret, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s.FilterCfg = cfgBytes
	return &s, nil
}

func (db *DB) DeleteSubscription(ctx context.Context, projectID, id string) error {
	res, err := db.Pool.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1 AND project_id = $2`, id, projectID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
