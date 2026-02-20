package repository

import (
	"context"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
)

const dealActionLockTTL = 5 * time.Minute

// TakeDealActionLock inserts a lock for the given deal and action type if no active lock exists.
// Active = status 'locked' and expire_at > now. Returns lock ID or error if already locked.
func (r *repository) TakeDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (string, error) {
	txCtx, beginErr := r.db.BeginTx(ctx, pgx.TxOptions{})
	if beginErr != nil {
		return "", beginErr
	}
	var err error
	defer func() { _ = r.db.EndTx(txCtx, err, "TakeDealActionLock") }()

	rows, err := r.db.Query(txCtx, `
		SELECT 1 FROM market.deal_action_lock
		WHERE deal_id = $1 AND action_type = $2 AND status = 'locked' AND expire_at > NOW()
		LIMIT 1`, dealID, string(actionType))
	if err != nil {
		return "", err
	}
	hasRow := rows.Next()
	rows.Close()
	if hasRow {
		err = errors.New("deal action already locked")
		return "", err
	}

	expireAt := time.Now().Add(dealActionLockTTL)
	insertRows, err := r.db.Query(txCtx, `
		INSERT INTO market.deal_action_lock (deal_id, action_type, status, expire_at, updated_at)
		VALUES ($1, $2, 'locked', $3, NOW())
		RETURNING id`, dealID, string(actionType), expireAt)
	if err != nil {
		return "", err
	}
	defer insertRows.Close()
	var lockID string
	if !insertRows.Next() {
		err = errors.New("insert lock returned no id")
		return "", err
	}
	if err = insertRows.Scan(&lockID); err != nil {
		return "", err
	}
	err = nil
	return lockID, nil
}

// GetExpiredDealActionLock returns the ID of an expired lock (status=locked, expire_at <= now) for the given deal and action type, if any.
// Used to detect "lock was held, app crashed after posting, lock expired" and recover by finding the message in the channel.
func (r *repository) GetExpiredDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (lockID string, ok bool, err error) {
	rows, err := r.db.Query(ctx, `
		SELECT id FROM market.deal_action_lock
		WHERE deal_id = $1 AND action_type = $2 AND status = 'locked' AND expire_at <= NOW()
		ORDER BY expire_at DESC
		LIMIT 1`, dealID, string(actionType))
	if err != nil {
		return "", false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", false, nil
	}
	if err = rows.Scan(&lockID); err != nil {
		return "", false, err
	}
	return lockID, true, nil
}

// ReleaseDealActionLock sets the lock status to completed or failed.
func (r *repository) ReleaseDealActionLock(ctx context.Context, lockID string, status entity.DealActionLockStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.deal_action_lock SET status = $1, updated_at = NOW() WHERE id = $2`,
		string(status), lockID)
	return err
}
