package storage

import (
	"context"
	"errors"
	"hookrelay/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (db *DB) CreateAPIKey(ctx context.Context, projectID, name, keyHash, prefix string) (*domain.APIKey, error) {
	var k domain.APIKey
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO api_keys (workspace_id, name, token_hash, prefix)
		VALUES ($1, $2, $3, $4)
		RETURNING id, workspace_id, name, token_hash, prefix, expires_at, created_at
	`, projectID, name, keyHash, prefix).
		Scan(&k.ID, &k.ProjectID, &k.Name, &k.KeyHash, &k.Prefix, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (db *DB) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var k domain.APIKey
	err := db.Pool.QueryRow(ctx, `
		SELECT id, workspace_id, name, token_hash, prefix, expires_at, created_at
		FROM api_keys WHERE token_hash = $1
	`, keyHash).Scan(&k.ID, &k.ProjectID, &k.Name, &k.KeyHash, &k.Prefix, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &k, nil
}

func (db *DB) ListAPIKeys(ctx context.Context, projectID string) ([]domain.APIKey, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, workspace_id, name, token_hash, prefix, expires_at, created_at
		FROM api_keys WHERE workspace_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.APIKey
	for rows.Next() {
		var k domain.APIKey
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &k.KeyHash, &k.Prefix, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	return result, rows.Err()
}

func (db *DB) DeleteAPIKey(ctx context.Context, projectID, keyID string) error {
	res, err := db.Pool.Exec(ctx, `
		DELETE FROM api_keys WHERE id = $1 AND workspace_id = $2
	`, keyID, projectID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
