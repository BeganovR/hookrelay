package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	DB     Database
	App    Application
	Worker Worker
}

type Database struct {
	PostgresURL string `env:"DATABASE_URL,required"`
}

type Application struct {
	Port             string  `env:"PORT"               envDefault:"8080"`
	BaseURL          string  `env:"BASE_URL"           envDefault:"http://localhost:8080"`
	IngestRatePerSec float64 `env:"INGEST_RATE_PER_SEC" envDefault:"20"`
	IngestRateBurst  int     `env:"INGEST_RATE_BURST"   envDefault:"40"`
}

type Worker struct {
	Concurrency  int           `env:"WORKER_CONCURRENCY"   envDefault:"10"`
	PollInterval time.Duration `env:"WORKER_POLL_INTERVAL" envDefault:"5s"`
	BatchSize    int           `env:"WORKER_BATCH_SIZE"    envDefault:"100"`
	StuckTimeout time.Duration `env:"WORKER_STUCK_TIMEOUT" envDefault:"5m"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
