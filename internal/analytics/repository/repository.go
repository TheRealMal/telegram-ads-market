package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ads-mrkt/internal/analytics/domain"
	"ads-mrkt/internal/analytics/repository/model"
	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const nanotonPerTON = 1e9

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (context.Context, error)
	EndTx(ctx context.Context, err error, source string) error
}

type Repository struct {
	db database
}

func New(db database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CollectSnapshot(ctx context.Context, transactionGasNanoton int64, commissionPercent float64) (*domain.Snapshot, error) {
	snap := &domain.Snapshot{
		DealsByStatus:          make(map[string]int64),
		DealAmountsByStatusTON: make(map[string]float64),
	}
	mult := 1.0 + (commissionPercent / 100.0)

	var err error
	snap.ListingsCount, err = r.queryListingsCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect snapshot: %w", err)
	}

	snap.DealsCount, err = r.queryDealsCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect snapshot: %w", err)
	}

	snap.DealsByStatus, err = r.queryDealsByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect snapshot: %w", err)
	}

	snap.DealAmountsByStatusTON, err = r.queryDealAmountsByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect snapshot: %w", err)
	}

	snap.CommissionEarnedNanoton, err = r.queryCommission(ctx, transactionGasNanoton, mult)
	if err != nil {
		return nil, fmt.Errorf("collect snapshot: %w", err)
	}

	snap.UsersCount, err = r.queryUsersCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect snapshot: %w", err)
	}

	if snap.UsersCount > 0 {
		snap.AvgListingsPerUser = float64(snap.ListingsCount) / float64(snap.UsersCount)
	}
	return snap, nil
}

func (r *Repository) queryListingsCount(ctx context.Context) (int64, error) {
	rows, err := r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.listing`)
	if err != nil {
		return 0, fmt.Errorf("query listing count: %w", err)
	}
	defer rows.Close()
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CountRow])
	if err != nil {
		return 0, fmt.Errorf("scan listing count: %w", err)
	}
	return row.Count, nil
}

func (r *Repository) queryDealsCount(ctx context.Context) (int64, error) {
	rows, err := r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.deal`)
	if err != nil {
		return 0, fmt.Errorf("query deal count: %w", err)
	}
	defer rows.Close()
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CountRow])
	if err != nil {
		return 0, fmt.Errorf("scan deal count: %w", err)
	}
	return row.Count, nil
}

func (r *Repository) queryDealsByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.Query(ctx, `SELECT status::text AS status, COUNT(*) AS count FROM market.deal GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("query deals by status: %w", err)
	}
	defer rows.Close()
	statusCounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.StatusCountRow])
	if err != nil {
		return nil, fmt.Errorf("scan deals by status: %w", err)
	}
	result := make(map[string]int64, len(statusCounts))
	for _, sc := range statusCounts {
		result[sc.Status] = sc.Count
	}
	return result, nil
}

func (r *Repository) queryDealAmountsByStatus(ctx context.Context) (map[string]float64, error) {
	rows, err := r.db.Query(ctx, `SELECT status::text AS status, COALESCE(SUM(price), 0) AS sum FROM market.deal GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("query deal amounts by status: %w", err)
	}
	defer rows.Close()
	statusSums, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.StatusSumRow])
	if err != nil {
		return nil, fmt.Errorf("scan deal amounts by status: %w", err)
	}
	result := make(map[string]float64, len(statusSums))
	for _, ss := range statusSums {
		result[ss.Status] = float64(ss.Sum) / nanotonPerTON
	}
	return result, nil
}

func (r *Repository) queryCommission(ctx context.Context, transactionGasNanoton int64, mult float64) (int64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(SUM(
			(escrow_amount - @gas) - ROUND((escrow_amount - @gas)::numeric / @mult)::bigint
		), 0) AS commission
		FROM market.deal
		WHERE status = @status_completed`,
		pgx.NamedArgs{"gas": transactionGasNanoton, "mult": mult, "status_completed": string(entity.DealStatusCompleted)},
	)
	if err != nil {
		return 0, fmt.Errorf("query commission: %w", err)
	}
	defer rows.Close()
	commRow, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CommissionRow])
	if err != nil {
		return 0, fmt.Errorf("scan commission: %w", err)
	}
	return commRow.Commission, nil
}

func (r *Repository) queryUsersCount(ctx context.Context) (int64, error) {
	rows, err := r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.user`)
	if err != nil {
		return 0, fmt.Errorf("query user count: %w", err)
	}
	defer rows.Close()
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CountRow])
	if err != nil {
		return 0, fmt.Errorf("scan user count: %w", err)
	}
	return row.Count, nil
}

func (r *Repository) InsertSnapshot(ctx context.Context, s *domain.Snapshot) error {
	todayUTC := time.Now().UTC().Truncate(24 * time.Hour)
	exists, err := r.HasSnapshotForDate(ctx, todayUTC)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	if exists {
		return nil // already have a snapshot for today, do not insert
	}
	dealsByStatus, err := s.DealsByStatusJSON()
	if err != nil {
		return fmt.Errorf("insert snapshot: marshal deals by status: %w", err)
	}
	amountsByStatus, err := s.DealAmountsByStatusTONJSON()
	if err != nil {
		return fmt.Errorf("insert snapshot: marshal deal amounts by status: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO analytic.snapshot (
			recorded_at,
			listings_count,
			deals_count,
			deals_by_status,
			deal_amounts_by_status_ton,
			commission_earned_nanoton,
			users_count,
			avg_listings_per_user
		) VALUES (
			NOW(),
			@listings_count, @deals_count, @deals_by_status, @deal_amounts_by_status_ton,
			@commission_earned_nanoton, @users_count, @avg_listings_per_user
		)`,
		pgx.NamedArgs{
			"listings_count":             s.ListingsCount,
			"deals_count":                s.DealsCount,
			"deals_by_status":            dealsByStatus,
			"deal_amounts_by_status_ton": amountsByStatus,
			"commission_earned_nanoton":  s.CommissionEarnedNanoton,
			"users_count":                s.UsersCount,
			"avg_listings_per_user":      s.AvgListingsPerUser,
		},
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

func (r *Repository) HasSnapshotForDate(ctx context.Context, date time.Time) (bool, error) {
	utcDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	rows, err := r.db.Query(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM analytic.snapshot
			WHERE (recorded_at AT TIME ZONE 'UTC')::date = @date
		) AS exists`,
		pgx.NamedArgs{"date": utcDate},
	)
	if err != nil {
		return false, fmt.Errorf("has snapshot for date %v: %w", date, err)
	}
	defer rows.Close()
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.ExistsRow])
	if err != nil {
		return false, fmt.Errorf("has snapshot for date %v: %w", date, err)
	}
	return row.Exists, nil
}

func (r *Repository) GetLatestSnapshot(ctx context.Context) (*domain.Snapshot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, recorded_at, listings_count, deals_count,
		       deals_by_status, deal_amounts_by_status_ton,
		       commission_earned_nanoton, users_count, avg_listings_per_user
		FROM analytic.snapshot
		ORDER BY recorded_at DESC
		LIMIT 1`,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	defer rows.Close()
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.SnapshotRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	return model.SnapshotRowToDomain(row), nil
}

func (r *Repository) ListSnapshots(ctx context.Context, from, to time.Time) ([]*domain.Snapshot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, recorded_at, listings_count, deals_count,
		       deals_by_status, deal_amounts_by_status_ton,
		       commission_earned_nanoton, users_count, avg_listings_per_user
		FROM analytic.snapshot
		WHERE recorded_at >= @from AND recorded_at <= @to
		ORDER BY recorded_at ASC`,
		pgx.NamedArgs{"from": from, "to": to},
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.SnapshotRow])
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	out := make([]*domain.Snapshot, 0, len(slice))
	for i := range slice {
		out = append(out, model.SnapshotRowToDomain(slice[i]))
	}
	return out, nil
}
