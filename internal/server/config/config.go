package config

import "time"

type Config struct {
	ListenAddr      string        `env:"LISTEN_ADDR" env-default:"localhost"`
	LocalPortPrefix int           `env:"LOCAL_PORT_PREFIX" env-default:"0"` //  will be summing with ports below
	ListenPort      int           `env:"LISTEN_PORT" env-default:"8080"`
	MetricsPort     int           `env:"METRICS_PORT" env-default:"8081"`
	SwaggerPort     int           `env:"SWAGGER_PORT" env-default:"8082"`
	ProbesPort      int           `env:"PROBES_PORT" env-default:"8083"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" env-default:"5s"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT" env-default:"5s"`
	ClientDomain    string        `env:"CLIENT_DOMAIN" env-default:"http://localhost"`
}

func (c *Config) InternalHandling() {
	c.ListenPort += c.LocalPortPrefix
	c.MetricsPort += c.LocalPortPrefix
	c.SwaggerPort += c.LocalPortPrefix
	c.ProbesPort += c.LocalPortPrefix
}
