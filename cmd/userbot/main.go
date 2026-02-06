package main

import (
	"ads-mrkt/internal/config"
	marketrepository "ads-mrkt/internal/market/repository/market"
	"ads-mrkt/internal/postgres"
	userbotrepository "ads-mrkt/internal/userbot/repository/state"
	userbotservice "ads-mrkt/internal/userbot/service/userbot"
	"context"
	"log"
	"log/slog"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", "error", err)
	}

	ctx := context.Background()

	slog.Info("initializing database")
	db, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		log.Fatal("failed to initialize db connection", "error", err)
	}

	stateStorage := userbotrepository.New(db)
	marketRepository := marketrepository.New(db)

	slog.Info("initializing bot")
	b := userbotservice.New(cfg.UserBot, stateStorage, marketRepository)

	slog.Info("polling")
	if err := b.Start(ctx); err != nil {
		log.Fatal("failed to start polling", "error", err)
	}
}
