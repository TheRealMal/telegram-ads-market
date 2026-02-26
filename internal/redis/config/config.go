package config

type Config struct {
	Host          string `env:"HOST" env-default:"0.0.0.0"`
	Port          int    `env:"PORT" env-default:"6379"`
	User          string `env:"USER" env-default:""`
	Password      string `env:"PASSWORD" env-default:""`
	DB            int    `env:"DB" env-default:"0"`
	MinTLSVersion uint16 `env:"MIN_TLS_VERSION" env-default:"769"` // 769 = TLS 1.0
	EnableTLS     bool   `env:"ENABLE_TLS" env-default:"false"`
}
