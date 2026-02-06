package config

import "time"

type Config struct {
	CheckInterval time.Duration `env:"HEALTHCHECK_CHECK_INTERVAL" env-default:"5s"`
	PingTimeout   time.Duration `env:"HEALTHCHECK_PING_TIMEOUT" env-default:"5s"`
	InstanceID    string        `env:"_" env-default:"local"`
}
