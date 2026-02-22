package service

import (
	"context"
	"log/slog"
	"math"
	"time"

	"ads-mrkt/internal/analytics/domain"
)

const collectInterval = 1 * time.Hour

type repository interface {
	CollectSnapshot(ctx context.Context, transactionGasNanoton int64, commissionPercent float64) (*domain.Snapshot, error)
	InsertSnapshot(ctx context.Context, snap *domain.Snapshot) error
	GetLatestSnapshot(ctx context.Context) (*domain.Snapshot, error)
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*domain.Snapshot, error)
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

// Run starts the hourly analytics collector.
func (s *service) Run(ctx context.Context) {
	ticker := time.NewTicker(collectInterval)
	defer ticker.Stop()
	s.collectAndSave(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectAndSave(ctx)
		}
	}
}

func (s *service) collectAndSave(ctx context.Context) {
	snap, err := s.repo.CollectSnapshot(ctx, s.transactionGasNanoton, s.commissionPercent)
	if err != nil {
		slog.Error("analytics collect", "error", err)
		return
	}
	if err := s.repo.InsertSnapshot(ctx, snap); err != nil {
		slog.Error("analytics insert snapshot", "error", err)
		return
	}
	slog.Info("analytics snapshot saved",
		"listings", snap.ListingsCount,
		"deals", snap.DealsCount,
		"users", snap.UsersCount,
	)
}

// GetLatestSnapshot returns the most recent analytics snapshot, if any.
func (s *service) GetLatestSnapshot(ctx context.Context) (*domain.Snapshot, error) {
	return s.repo.GetLatestSnapshot(ctx)
}

// ListSnapshots returns snapshots in the given time range, ordered by recorded_at ASC.
func (s *service) ListSnapshots(ctx context.Context, from, to time.Time) ([]*domain.Snapshot, error) {
	return s.repo.ListSnapshots(ctx, from, to)
}
