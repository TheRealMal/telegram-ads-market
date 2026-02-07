package config

type Config struct {
	JWTSecret     string `env:"JWT_SECRET" env-required:"true"`
	JWTTimeToLive int    `env:"JWT_TIME_TO_LIVE" env-default:"24"` // hours
}
