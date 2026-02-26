package config

type Config struct {
	Address   string `env:"ADDR" env-default:"http://127.0.0.1:8200"`
	Token     string `env:"TOKEN" env-default:""`
	MountPath string `env:"MOUNT_PATH" env-default:"secret"`
}
