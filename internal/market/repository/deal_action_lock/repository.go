package deal_action_lock

import (
	"context"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (context.Context, error)
	EndTx(ctx context.Context, err error, source string) error
}

const dealActionLockTTL = 5 * time.Minute

type repository struct {
	db database
}

func New(db database) *repository {
	return &repository{db: db}
}

type dealActionLockRow struct {
	ID         string    `db:"id"`
	DealID     int64     `db:"deal_id"`
	ActionType string    `db:"action_type"`
	Status     string    `db:"status"`
	ExpireAt   time.Time `db:"expire_at"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type dealActionLockExistsRow struct {
	One int `db:"one"`
}

type dealActionLockReturnRow struct {
	ID string `db:"id"`
}

func (r *repository) TakeDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (string, error) {
	txCtx, beginErr := r.db.BeginTx(ctx, pgx.TxOptions{})
	if beginErr != nil {
		return "", beginErr
	}
	var err error
	defer func() { _ = r.db.EndTx(txCtx, err, "TakeDealActionLock") }()

	rows, err := r.db.Query(txCtx, `
		SELECT 1 AS one FROM market.deal_action_lock
		WHERE deal_id = @deal_id AND action_type = @action_type AND status = 'locked' AND expire_at > NOW()
		LIMIT 1`,
		pgx.NamedArgs{
			"deal_id":     dealID,
			"action_type": string(actionType),
		},
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	active, err := pgx.CollectRows(rows, pgx.RowToStructByName[dealActionLockExistsRow])
	if err != nil {
		return "", err
	}
	if len(active) > 0 {
		err = errors.New("deal action already locked")
		return "", err
	}

	expireAt := time.Now().Add(dealActionLockTTL)
	insertRows, err := r.db.Query(txCtx, `
		INSERT INTO market.deal_action_lock (deal_id, action_type, status, expire_at, updated_at)
		VALUES (@deal_id, @action_type, 'locked', @expire_at, NOW())
		RETURNING id`,
		pgx.NamedArgs{
			"deal_id":     dealID,
			"action_type": string(actionType),
			"expire_at":   expireAt,
		},
	)
	if err != nil {
		return "", err
	}
	defer insertRows.Close()
	row, err := pgx.CollectExactlyOneRow(insertRows, pgx.RowToStructByName[dealActionLockReturnRow])
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func (r *repository) GetLastDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (*entity.DealActionLock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, deal_id, action_type, status, expire_at, created_at, updated_at
		FROM market.deal_action_lock
		WHERE deal_id = @deal_id AND action_type = @action_type
		ORDER BY created_at DESC
		LIMIT 1`,
		pgx.NamedArgs{
			"deal_id":     dealID,
			"action_type": string(actionType),
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dealActionLockRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entity.DealActionLock{
		ID:         row.ID,
		DealID:     row.DealID,
		ActionType: entity.DealActionType(row.ActionType),
		Status:     entity.DealActionLockStatus(row.Status),
		ExpireAt:   row.ExpireAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func (r *repository) GetExpiredDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (lockID string, ok bool, err error) {
	lock, err := r.GetLastDealActionLock(ctx, dealID, actionType)
	if err != nil || lock == nil {
		return "", false, err
	}
	if lock.Status != entity.DealActionLockStatusLocked || lock.ExpireAt.After(time.Now()) {
		return "", false, nil
	}
	return lock.ID, true, nil
}

func (r *repository) ReleaseDealActionLock(ctx context.Context, lockID string, status entity.DealActionLockStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal_action_lock SET status = @status, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{
			"status": string(status),
			"id":     lockID,
		},
	)
	return err
}
