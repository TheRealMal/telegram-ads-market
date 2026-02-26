package config

import "time"

type Config struct {
	Host           string        `env:"HOST" env-default:"localhost"`
	Port           int           `env:"PORT" env-default:"5432"`
	Name           string        `env:"NAME" env-default:"postgres"`
	User           string        `env:"USER" env-default:"postgres"`
	Password       string        `env:"PASSWORD" env-default:"postgres"`
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT" env-default:"5s"`
	PingTimeout    time.Duration `env:"PING_TIMEOUT" env-default:"5s"`
	SSLMode        string        `env:"SSL_MODE" env-default:"disable"`
	LockTimeout    time.Duration `env:"LOCK_TIMEOUT" env-default:"3s"`
}
