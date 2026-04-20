package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	DB  Database
	App Application
}

type Database struct {
	PostgresURL string `env:"DATABASE_URL,required"`
}

type Application struct {
	Port string `env:"PORT" envDefault:"8080"`
}

func Load() (*Config, error) {
	_ = godotenv.Load() // on local, you put yours environment on .env file
	// on server, you don't have a .env file. Scan on system environment

	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
