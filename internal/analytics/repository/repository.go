package repository

import (
	"context"
	"errors"
	"time"

	"ads-mrkt/internal/analytics/domain"
	"ads-mrkt/internal/analytics/repository/model"

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

	rows, err := r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.listing`)
	if err != nil {
		return nil, err
	}
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CountRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	snap.ListingsCount = row.Count

	rows, err = r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.deal`)
	if err != nil {
		return nil, err
	}
	row, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CountRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	snap.DealsCount = row.Count

	rows, err = r.db.Query(ctx, `SELECT status::text AS status, COUNT(*) AS count FROM market.deal GROUP BY status`)
	if err != nil {
		return nil, err
	}
	statusCounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.StatusCountRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, sc := range statusCounts {
		snap.DealsByStatus[sc.Status] = sc.Count
	}

	rows, err = r.db.Query(ctx, `SELECT status::text AS status, COALESCE(SUM(price), 0) AS sum FROM market.deal GROUP BY status`)
	if err != nil {
		return nil, err
	}
	statusSums, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.StatusSumRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, ss := range statusSums {
		snap.DealAmountsByStatusTON[ss.Status] = float64(ss.Sum) / nanotonPerTON
	}

	rows, err = r.db.Query(ctx, `
		SELECT COALESCE(SUM(
			(escrow_amount - @gas) - ROUND((escrow_amount - @gas)::numeric / @mult)::bigint
		), 0) AS commission
		FROM market.deal
		WHERE status = 'completed'`,
		pgx.NamedArgs{"gas": transactionGasNanoton, "mult": mult},
	)
	if err != nil {
		return nil, err
	}
	commRow, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CommissionRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	snap.CommissionEarnedNanoton = commRow.Commission

	rows, err = r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.user`)
	if err != nil {
		return nil, err
	}
	row, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[model.CountRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	snap.UsersCount = row.Count

	if snap.UsersCount > 0 {
		snap.AvgListingsPerUser = float64(snap.ListingsCount) / float64(snap.UsersCount)
	}
	return snap, nil
}

func (r *Repository) InsertSnapshot(ctx context.Context, s *domain.Snapshot) error {
	todayUTC := time.Now().UTC().Truncate(24 * time.Hour)
	exists, err := r.HasSnapshotForDate(ctx, todayUTC)
	if err != nil {
		return err
	}
	if exists {
		return nil // already have a snapshot for today, do not insert
	}
	dealsByStatus, err := s.DealsByStatusJSON()
	if err != nil {
		return err
	}
	amountsByStatus, err := s.DealAmountsByStatusTONJSON()
	if err != nil {
		return err
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
			"listings_count":            s.ListingsCount,
			"deals_count":               s.DealsCount,
			"deals_by_status":           dealsByStatus,
			"deal_amounts_by_status_ton": amountsByStatus,
			"commission_earned_nanoton":  s.CommissionEarnedNanoton,
			"users_count":               s.UsersCount,
			"avg_listings_per_user":     s.AvgListingsPerUser,
		},
	)
	return err
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
		return false, err
	}
	defer rows.Close()
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.ExistsRow])
	if err != nil {
		return false, err
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
		return nil, err
	}
	defer rows.Close()
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.SnapshotRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
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
		return nil, err
	}
	defer rows.Close()
	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.SnapshotRow])
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Snapshot, 0, len(slice))
	for i := range slice {
		out = append(out, model.SnapshotRowToDomain(slice[i]))
	}
	return out, nil
}
