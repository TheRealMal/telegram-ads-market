package service

import (
	"context"
	"log/slog"
	"math"
	"time"

	"ads-mrkt/internal/analytics/domain"
	"ads-mrkt/internal/worker"
)

const collectInterval = 1 * time.Hour

const snapshotHourUTC = 3

type repository interface {
	CollectSnapshot(ctx context.Context, transactionGasNanoton int64, commissionPercent float64) (*domain.Snapshot, error)
	InsertSnapshot(ctx context.Context, snap *domain.Snapshot) error
	GetLatestSnapshot(ctx context.Context) (*domain.Snapshot, error)
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*domain.Snapshot, error)
	HasSnapshotForDate(ctx context.Context, date time.Time) (bool, error)
}

type service struct {
	repo                  repository
	transactionGasNanoton int64
	commissionPercent     float64
}

func New(repo repository, transactionGasTON float64, commissionPercent float64) *service {
	nanotonPerTON := 1e9
	transactionGasNanoton := int64(math.Round(transactionGasTON * nanotonPerTON))
	return &service{
		repo:                  repo,
		transactionGasNanoton: transactionGasNanoton,
		commissionPercent:     commissionPercent,
	}
}

func (s *service) RunSnapshotWorker(ctx context.Context) {
	worker.RunTicker(ctx, "analytics_snapshot", collectInterval, true, func(ctx context.Context, logger *slog.Logger) {
		s.collectAndSave(ctx, logger)
	})
}

func (s *service) collectAndSave(ctx context.Context, logger *slog.Logger) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	exists, err := s.repo.HasSnapshotForDate(ctx, today)
	if err != nil {
		logger.Error("check snapshot for date", "error", err)
		return
	}
	if exists {
		return // already have today's snapshot
	}

	// Only take snapshot after 3:00 AM UTC today (fetch at 3:00 AM; catch-up if we're past that and none exists).
	snapshotTimeToday := time.Date(now.Year(), now.Month(), now.Day(), snapshotHourUTC, 0, 0, 0, time.UTC)
	if now.Before(snapshotTimeToday) {
		return // wait until 3:00 AM UTC
	}

	snap, err := s.repo.CollectSnapshot(ctx, s.transactionGasNanoton, s.commissionPercent)
	if err != nil {
		logger.Error("collect", "error", err)
		return
	}
	if err := s.repo.InsertSnapshot(ctx, snap); err != nil {
		logger.Error("insert snapshot", "error", err)
		return
	}
	logger.Info("snapshot saved",
		"listings", snap.ListingsCount,
		"deals", snap.DealsCount,
		"users", snap.UsersCount,
	)
}

func (s *service) GetLatestSnapshot(ctx context.Context) (*domain.Snapshot, error) {
	return s.repo.GetLatestSnapshot(ctx)
}

func (s *service) ListSnapshots(ctx context.Context, from, to time.Time) ([]*domain.Snapshot, error) {
	return s.repo.ListSnapshots(ctx, from, to)
}
