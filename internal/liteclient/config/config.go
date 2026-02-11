package config

var (
	// GlobalConfigURL is used when GlobalConfigDir is not set (e.g. local dev).
	GlobalConfigURL map[bool]string = map[bool]string{
		true:  "https://ton-blockchain.github.io/testnet-global.config.json",
		false: "https://ton.org/global-config.json",
	}
	// GlobalConfigFilename is the filename under GlobalConfigDir for each network.
	GlobalConfigFilename = map[bool]string{
		true:  "testnet-global.config.json",
		false: "global-config.json",
	}
)

type Config struct {
	LiteserverHost  string `env:"HOST"`
	LiteserverKey   string `env:"KEY"`
	GlobalConfigDir string `env:"GLOBAL_CONFIG_DIR"` // If set, load global config from this dir; otherwise fetch from URL
}
