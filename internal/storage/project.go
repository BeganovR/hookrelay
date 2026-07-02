package storage

import (
	"context"
	"errors"
	"hookrelay/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (db *DB) CreateProject(ctx context.Context, name, description string) (*domain.Project, error) {
	var p domain.Project
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at
	`, name, description).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	var p domain.Project
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM workspaces WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (db *DB) UpdateProject(ctx context.Context, id, name, description string) (*domain.Project, error) {
	var p domain.Project
	err := db.Pool.QueryRow(ctx, `
		UPDATE workspaces
		SET name = COALESCE(NULLIF($2,''), name),
		    description = COALESCE(NULLIF($3,''), description),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, description, created_at, updated_at
	`, id, name, description).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (db *DB) DeleteProject(ctx context.Context, id string) error {
	res, err := db.Pool.Exec(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
