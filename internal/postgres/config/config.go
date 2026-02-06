package config

import "time"

type Config struct {
	Host           string        `env:"DB_HOST" env-default:"localhost"`
	Port           int           `env:"DB_PORT" env-default:"5432"`
	Name           string        `env:"DB_NAME" env-default:"postgres"`
	User           string        `env:"DB_USER" env-default:"postgres"`
	Password       string        `env:"DB_PASSWORD" env-default:"postgres"`
	ConnectTimeout time.Duration `env:"DB_CONNECT_TIMEOUT" env-default:"5s"`
	PingTimeout    time.Duration `env:"DB_PING_TIMEOUT" env-default:"5s"`
	SSLMode        string        `env:"DB_SSL_MODE" env-default:"disable"`
	LockTimeout    time.Duration `env:"DB_LOCK_TIMEOUT" env-default:"3s"`
}
