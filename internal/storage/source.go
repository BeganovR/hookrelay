package storage

import (
	"context"
	"errors"
	"hookrelay/internal/domain"

	"github.com/jackc/pgx/v5"
)

type CreateSourceParams struct {
	ProjectID    string
	Name         string
	UID          string
	VerifierType string
	VerifierCfg  []byte
}

type UpdateSourceParams struct {
	Name         *string
	VerifierType *string
	VerifierCfg  []byte
	IsActive     *bool
}

func (db *DB) CreateSource(ctx context.Context, p CreateSourceParams) (*domain.Source, error) {
	var s domain.Source
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO sources (workspace_id, provider_id, name, slug, verifier_type, verifier_cfg)
		VALUES ($1, (SELECT id FROM providers WHERE name = 'custom'), $2, $3, $4, $5::jsonb)
		RETURNING id, workspace_id, name, slug, verifier_type, verifier_cfg, is_active, created_at, updated_at
	`, p.ProjectID, p.Name, p.UID, p.VerifierType, p.VerifierCfg).
		Scan(&s.ID, &s.ProjectID, &s.Name, &s.UID, &s.VerifierType, &cfgBytes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.VerifierCfg = cfgBytes
	return &s, nil
}

func (db *DB) GetSourceByUID(ctx context.Context, uid string) (*domain.Source, error) {
	return db.getSource(ctx, `
		SELECT id, workspace_id, name, slug, verifier_type, verifier_cfg, is_active, created_at, updated_at
		FROM sources WHERE slug = $1
	`, uid)
}

func (db *DB) GetSource(ctx context.Context, projectID, id string) (*domain.Source, error) {
	return db.getSource(ctx, `
		SELECT id, workspace_id, name, slug, verifier_type, verifier_cfg, is_active, created_at, updated_at
		FROM sources WHERE id = $1 AND workspace_id = $2
	`, id, projectID)
}

func (db *DB) getSource(ctx context.Context, query string, args ...any) (*domain.Source, error) {
	var s domain.Source
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, query, args...).
		Scan(&s.ID, &s.ProjectID, &s.Name, &s.UID, &s.VerifierType, &cfgBytes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s.VerifierCfg = cfgBytes
	return &s, nil
}

func (db *DB) ListSources(ctx context.Context, projectID string) ([]domain.Source, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, workspace_id, name, slug, verifier_type, verifier_cfg, is_active, created_at, updated_at
		FROM sources WHERE workspace_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Source
	for rows.Next() {
		var s domain.Source
		var cfgBytes []byte
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.UID, &s.VerifierType, &cfgBytes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.VerifierCfg = cfgBytes
		result = append(result, s)
	}
	return result, rows.Err()
}

func (db *DB) UpdateSource(ctx context.Context, projectID, id string, p UpdateSourceParams) (*domain.Source, error) {
	var s domain.Source
	var cfgBytes []byte
	err := db.Pool.QueryRow(ctx, `
		UPDATE sources
		SET name          = COALESCE($3, name),
		    verifier_type = COALESCE($4, verifier_type),
		    verifier_cfg  = COALESCE($5::jsonb, verifier_cfg),
		    is_active     = COALESCE($6, is_active),
		    updated_at    = NOW()
		WHERE id = $1 AND workspace_id = $2
		RETURNING id, workspace_id, name, slug, verifier_type, verifier_cfg, is_active, created_at, updated_at
	`, id, projectID, p.Name, p.VerifierType, p.VerifierCfg, p.IsActive).
		Scan(&s.ID, &s.ProjectID, &s.Name, &s.UID, &s.VerifierType, &cfgBytes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s.VerifierCfg = cfgBytes
	return &s, nil
}

func (db *DB) DeleteSource(ctx context.Context, projectID, id string) error {
	res, err := db.Pool.Exec(ctx, `DELETE FROM sources WHERE id = $1 AND workspace_id = $2`, id, projectID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
