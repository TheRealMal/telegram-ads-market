package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ads-mrkt/internal/analytics/domain"

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

type countRow struct {
	Count int64 `db:"count"`
}

type statusCountRow struct {
	Status string `db:"status"`
	Count  int64  `db:"count"`
}

type statusSumRow struct {
	Status string `db:"status"`
	Sum    int64  `db:"sum"`
}

type commissionRow struct {
	Commission int64 `db:"commission"`
}

type existsRow struct {
	Exists bool `db:"exists"`
}

type snapshotRow struct {
	ID                      int64           `db:"id"`
	RecordedAt              time.Time       `db:"recorded_at"`
	ListingsCount           int64           `db:"listings_count"`
	DealsCount              int64           `db:"deals_count"`
	DealsByStatus           json.RawMessage `db:"deals_by_status"`
	DealAmountsByStatusTon  json.RawMessage `db:"deal_amounts_by_status_ton"`
	CommissionEarnedNanoton int64           `db:"commission_earned_nanoton"`
	UsersCount              int64           `db:"users_count"`
	AvgListingsPerUser      float64         `db:"avg_listings_per_user"`
}

// CollectSnapshot runs aggregation queries against market schema and returns a snapshot (not persisted).
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
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[countRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	snap.ListingsCount = row.Count

	rows, err = r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.deal`)
	if err != nil {
		return nil, err
	}
	row, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[countRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	snap.DealsCount = row.Count

	rows, err = r.db.Query(ctx, `SELECT status::text AS status, COUNT(*) AS count FROM market.deal GROUP BY status`)
	if err != nil {
		return nil, err
	}
	statusCounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[statusCountRow])
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
	statusSums, err := pgx.CollectRows(rows, pgx.RowToStructByName[statusSumRow])
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
	commRow, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[commissionRow])
	rows.Close()
	if err != nil {
		return nil, err
	}
	snap.CommissionEarnedNanoton = commRow.Commission

	rows, err = r.db.Query(ctx, `SELECT COUNT(*) AS count FROM market.user`)
	if err != nil {
		return nil, err
	}
	row, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[countRow])
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

// HasSnapshotForDate returns true if at least one snapshot exists for the given UTC date.
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
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[existsRow])
	if err != nil {
		return false, err
	}
	return row.Exists, nil
}

// GetLatestSnapshot returns the most recent snapshot, if any.
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
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[snapshotRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return snapshotRowToDomain(row), nil
}

// ListSnapshots returns snapshots in the given time range, ordered by recorded_at ASC (for line charts).
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
	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[snapshotRow])
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Snapshot, 0, len(slice))
	for i := range slice {
		out = append(out, snapshotRowToDomain(slice[i]))
	}
	return out, nil
}

func snapshotRowToDomain(row snapshotRow) *domain.Snapshot {
	s := &domain.Snapshot{
		ID:                      row.ID,
		RecordedAt:              row.RecordedAt.Format(time.RFC3339),
		ListingsCount:           row.ListingsCount,
		DealsCount:              row.DealsCount,
		CommissionEarnedNanoton: row.CommissionEarnedNanoton,
		UsersCount:              row.UsersCount,
		AvgListingsPerUser:      row.AvgListingsPerUser,
	}
	_ = json.Unmarshal(row.DealsByStatus, &s.DealsByStatus)
	_ = json.Unmarshal(row.DealAmountsByStatusTon, &s.DealAmountsByStatusTON)
	return s
}
