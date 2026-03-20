package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Load provide app config.
func Load() (*Config, error) {
	if err := loadEnv(); err != nil {
		return nil, err
	}

	return makeConfig()
}

func loadEnv() error {
	envFile := envFile

	if customEnvFile, ok := os.LookupEnv(envFileEnv); ok {
		envFile = customEnvFile
	}

	if _, err := os.Stat(envFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("os file stat: %w", err)
	}

	if err := godotenv.Load(envFile); err != nil {
		return fmt.Errorf("on load config from %s: %w", envFile, err)
	}

	return nil
}

func makeConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read environment: %w", err)
	}

	cfg.InternalHandling()

	return &cfg, nil
}
