package config

import (
	dbconfig "ads-mrkt/internal/postgres/config"
	serverconfig "ads-mrkt/internal/server/config"
	userbotconfig "ads-mrkt/internal/userbot/config"
	authconfig "ads-mrkt/pkg/auth/config"
	healthconfig "ads-mrkt/pkg/health/config"
)

type Config struct {
	LogLevel string `env:"LOG_LEVEL" env-default:"info"`

	UserBot  userbotconfig.Config `env-prefix:"USER_BOT_"`
	Database dbconfig.Config
	Server   serverconfig.Config
	Health   healthconfig.Config
	Auth     authconfig.Config
}

func (c *Config) InternalHandling() {
	c.Server.InternalHandling()
}
