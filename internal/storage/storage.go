package storage

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(dbURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	slog.Info("Database connected successfully")
	return pool, nil
}

type Storage struct {
	Pool *pgxpool.Pool
}

func NewStorage(dbPool *pgxpool.Pool) *Storage {
	return &Storage{
		Pool: dbPool,
	}
}

func (s *Storage) SaveWebhook(id string, body []byte) error {
	//TODO: Here we are do a SQL request to save webhook

	slog.Info("Webhook saved to database")
	return nil
}
