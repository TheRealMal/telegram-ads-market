package config

import (
	telegramconfig "ads-mrkt/internal/helpers/telegram/config"
	liteclientconfig "ads-mrkt/internal/liteclient/config"
	dbconfig "ads-mrkt/internal/postgres/config"
	redisconfig "ads-mrkt/internal/redis/config"
	serverconfig "ads-mrkt/internal/server/config"
	userbotconfig "ads-mrkt/internal/userbot/config"
	authconfig "ads-mrkt/pkg/auth/config"
	healthconfig "ads-mrkt/pkg/health/config"
)

type Config struct {
	LogLevel string `env:"LOG_LEVEL" env-default:"info"`

	UserBot    userbotconfig.Config `env-prefix:"USER_BOT_"`
	Database   dbconfig.Config
	Server     serverconfig.Config
	Health     healthconfig.Config
	Auth       authconfig.Config
	Redis      redisconfig.Config
	Telegram   telegramconfig.Config   `env-prefix:"TELEGRAM_"`
	Liteclient liteclientconfig.Config `env-prefix:"LITECLIENT_"`
	Market     MarketConfig `env-prefix:"MARKET_"`
	IsPublic   bool `env:"IS_PUBLIC" env-default:"false"`
	IsTestnet  bool `env:"IS_TESTNET" env-default:"false"`
}

// MarketConfig holds market/escrow-related config.
type MarketConfig struct {
	Escrow EscrowConfig `env-prefix:"ESCROW_"`
}


// EscrowConfig holds escrow amount computation: escrow_amount = price + (TransactionGasTON * 1e9) + (price * CommissionPercent / 100).
type EscrowConfig struct {
	TransactionGasTON  float64 `env:"TRANSACTION_GAS_TON" env-default:"0.1"`
	CommissionPercent  float64 `env:"COMMISSION_PERCENT" env-default:"2"`
}

func (c *Config) InternalHandling() {
	c.Server.InternalHandling()
}
