package config

type Config struct {
	Host          string `env:"REDIS_HOST" env-default:"0.0.0.0"`
	Port          int    `env:"REDIS_PORT" env-default:"6379"`
	User          string `env:"REDIS_USER" env-default:""`
	Password      string `env:"REDIS_PASSWORD" env-default:""`
	DB            int    `env:"REDIS_DB" env-default:"0"`
	MinTLSVersion uint16 `env:"REDIS_MIN_TLS_VERSION" env-default:"769"` // 769 = TLS 1.0
	EnableTLS     bool   `env:"REDIS_ENABLE_TLS" env-default:"false"`
}
