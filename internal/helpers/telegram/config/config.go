package config

type Config struct {
	Token         string `env:"BOT_TOKEN" env-required:"true"`
	BotUsername   string `env:"BOT_USERNAME"`
	BotWebAppName string `env:"BOT_WEB_APP_NAME"`
	SecretToken   string `env:"SECRET_TOKEN"`
	RateLimit     int    `env:"RATE_LIMIT" env-default:"30"`
	ServiceChatID int64  `env:"BOT_SERVICE_CHAT_ID"` // chat ID where bot copies messages for existence checks
}
