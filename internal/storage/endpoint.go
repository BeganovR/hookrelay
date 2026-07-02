package storage

import (
	"context"
	"errors"
	"hookrelay/internal/domain"

	"github.com/jackc/pgx/v5"
)

type CreateEndpointParams struct {
	ProjectID   string
	Name        string
	URL         string
	Description string
	HTTPTimeout int
	AuthType    string
	AuthCfg     []byte
}

type UpdateEndpointParams struct {
	Name        *string
	URL         *string
	Description *string
	HTTPTimeout *int
	AuthType    *string
	AuthCfg     []byte
	IsActive    *bool
}

func (db *DB) CreateEndpoint(ctx context.Context, p CreateEndpointParams) (*domain.Endpoint, error) {
	if p.HTTPTimeout == 0 {
		p.HTTPTimeout = 30
	}
	var e domain.Endpoint
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO endpoints (project_id, name, url, description, http_timeout, auth_type, auth_cfg)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		RETURNING id, project_id, name, url, description, http_timeout, auth_type, auth_cfg, is_active, created_at, updated_at
	`, p.ProjectID, p.Name, p.URL, p.Description, p.HTTPTimeout, p.AuthType, p.AuthCfg).
		Scan(&e.ID, &e.ProjectID, &e.Name, &e.URL, &e.Description, &e.HTTPTimeout, &e.AuthType, &cfgBytes, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.AuthCfg = cfgBytes
	return &e, nil
}

func (db *DB) GetEndpoint(ctx context.Context, projectID, id string) (*domain.Endpoint, error) {
	var e domain.Endpoint
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, project_id, name, url, description, http_timeout, auth_type, auth_cfg, is_active, created_at, updated_at
		FROM endpoints WHERE id = $1 AND project_id = $2
	`, id, projectID).
		Scan(&e.ID, &e.ProjectID, &e.Name, &e.URL, &e.Description, &e.HTTPTimeout, &e.AuthType, &cfgBytes, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	e.AuthCfg = cfgBytes
	return &e, nil
}

func (db *DB) ListEndpoints(ctx context.Context, projectID string) ([]domain.Endpoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, project_id, name, url, description, http_timeout, auth_type, auth_cfg, is_active, created_at, updated_at
		FROM endpoints WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Endpoint
	for rows.Next() {
		var e domain.Endpoint
		var cfgBytes []byte
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.URL, &e.Description, &e.HTTPTimeout, &e.AuthType, &cfgBytes, &e.IsActive, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.AuthCfg = cfgBytes
		result = append(result, e)
	}
	return result, rows.Err()
}

func (db *DB) UpdateEndpoint(ctx context.Context, projectID, id string, p UpdateEndpointParams) (*domain.Endpoint, error) {
	var e domain.Endpoint
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		UPDATE endpoints
		SET name         = COALESCE($3, name),
		    url          = COALESCE($4, url),
		    description  = COALESCE($5, description),
		    http_timeout = COALESCE($6, http_timeout),
		    auth_type    = COALESCE($7, auth_type),
		    auth_cfg     = COALESCE($8::jsonb, auth_cfg),
		    is_active    = COALESCE($9, is_active),
		    updated_at   = NOW()
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, name, url, description, http_timeout, auth_type, auth_cfg, is_active, created_at, updated_at
	`, id, projectID, p.Name, p.URL, p.Description, p.HTTPTimeout, p.AuthType, p.AuthCfg, p.IsActive).
		Scan(&e.ID, &e.ProjectID, &e.Name, &e.URL, &e.Description, &e.HTTPTimeout, &e.AuthType, &cfgBytes, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	e.AuthCfg = cfgBytes
	return &e, nil
}

func (db *DB) DeleteEndpoint(ctx context.Context, projectID, id string) error {
	res, err := db.Pool.Exec(ctx, `DELETE FROM endpoints WHERE id = $1 AND project_id = $2`, id, projectID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
