package config

type Config struct {
	ApiID       int    `env:"API_ID"`
	ApiHash     string `env:"API_HASH"`
	SessionFile string `env:"SESSION_FILE_PATH"`
	Phone       string `env:"PHONE"`
}
