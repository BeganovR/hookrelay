package storage

import (
	"context"
	"embed"
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
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
	query := `INSERT INTO webhooks (id, client_id, body) VALUES ($1, $2, $3::jsonb)`
	newID := uuid.New()
	_, err := s.Pool.Exec(context.Background(), query, newID, id, body)
	if err != nil {
		return err
	}
	slog.Info("Webhook saved to database")
	return nil
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(dbURL string) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		dbURL,
	)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	slog.Info("Migrations applied successfully")
	return nil
}

//func ViewDatabase(s *Storage) (pgx.Rows, error) {
//	sql := "SELECT * FROM webhooks"
//	rows, err := s.Pool.Query(context.Background(), sql)
//	if err != nil {
//		return nil, err
//	}
//	return rows, nil
//}
