package storage

import (
	"context"
	"errors"
	"hookrelay/internal/domain"

	"github.com/jackc/pgx/v5"
)

type CreateEventParams struct {
	ProjectID      string
	SourceID       *string
	EventType      string
	Headers        []byte
	Payload        []byte
	SenderIP       string
	IdempotencyKey *string
}

func (db *DB) CreateEvent(ctx context.Context, p CreateEventParams) (*domain.Event, error) {
	var e domain.Event
	var hdrBytes, payBytes []byte
	err := db.Pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO events (project_id, source_id, event_type, headers, payload, payload_size, sender_ip, idempotency_key)
			VALUES ($1, $2, $3, COALESCE($4::jsonb, '{}'::jsonb), $5::jsonb, octet_length($6), $7::inet, $8)
			ON CONFLICT (project_id, idempotency_key) WHERE idempotency_key IS NOT NULL
			DO NOTHING
			RETURNING id, project_id, source_id, event_type, headers, payload, sender_ip::text, idempotency_key, received_at AS created_at
		)
		SELECT id, project_id, source_id, event_type, headers, payload, sender_ip::text, idempotency_key, created_at, true AS is_new
		FROM inserted
		UNION ALL
		SELECT id, project_id, source_id, event_type, headers, payload, sender_ip::text, idempotency_key, received_at AS created_at, false AS is_new
		FROM events
		WHERE project_id = $1 AND idempotency_key = $8 AND $8 IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM inserted)
		LIMIT 1
	`, p.ProjectID, p.SourceID, p.EventType, p.Headers, p.Payload, p.Payload, p.SenderIP, p.IdempotencyKey).
		Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.EventType, &hdrBytes, &payBytes, &e.SenderIP, &e.IdempotencyKey, &e.CreatedAt, &e.IsNew)
	if err != nil {
		return nil, err
	}
	e.Headers = hdrBytes
	e.Payload = payBytes
	return &e, nil
}

func (db *DB) GetEvent(ctx context.Context, projectID, id string) (*domain.Event, error) {
	var e domain.Event
	var hdrBytes, payBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, project_id, source_id, event_type, headers, payload, sender_ip::text, idempotency_key, received_at AS created_at
		FROM events WHERE id = $1 AND project_id = $2
	`, id, projectID).
		Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.EventType, &hdrBytes, &payBytes, &e.SenderIP, &e.IdempotencyKey, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	e.Headers = hdrBytes
	e.Payload = payBytes
	return &e, nil
}

func (db *DB) ListEvents(ctx context.Context, projectID string, limit int) ([]domain.Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT id, project_id, source_id, event_type, headers, payload, sender_ip::text, idempotency_key, received_at AS created_at
		FROM events WHERE project_id = $1
		ORDER BY received_at DESC, id DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Event
	for rows.Next() {
		var e domain.Event
		var hdrBytes, payBytes []byte
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.EventType, &hdrBytes, &payBytes, &e.SenderIP, &e.IdempotencyKey, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Headers = hdrBytes
		e.Payload = payBytes
		result = append(result, e)
	}
	return result, rows.Err()
}
