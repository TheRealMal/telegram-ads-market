package main

import (
	"context"
	"log/slog"
	"os"

	"ads-mrkt/cmd/bot"
	"ads-mrkt/cmd/market"
	"ads-mrkt/cmd/userbot"
	"ads-mrkt/internal/config"

	"github.com/pkg/errors"
	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"
)

// @title						Swagger for Ads Market API
// @version					1.0
// @license.name				Apache 2.0
// @license.url				http://www.apache.org/licenses/LICENSE-2.0.html
// @schemes					http
// @host						localhost:8080
// @BasePath					/api/v1
// @Security					JWT
// @securityDefinitions.apiKey	JWT
// @in							header
// @name						Authorization
// @description				Bearer token
func main() {
	conf, err := config.Load()
	if err != nil {
		panic(err)
	}

	if conf.LogLevel == "debug" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})))
	}

	ctx := context.Background()

	rootCmd := &cobra.Command{
		Use:     "ads-mrkt",
		Version: version.Version,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Usage()
		},
	}

	rootCmd.AddCommand(
		market.ApiCmd(ctx, conf),
		bot.BotCmd(ctx, conf),
		userbot.UserbotCmd(ctx, conf),
	)

	if err := errors.Wrap(rootCmd.ExecuteContext(ctx), "error executing root cmd"); err != nil {
		slog.Error("failed to execute command", "error", err)

		return
	}
}
