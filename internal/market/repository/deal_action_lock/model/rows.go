package model

import (
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

type DealActionLockRow struct {
	ID         string    `db:"id"`
	DealID     int64     `db:"deal_id"`
	ActionType string    `db:"action_type"`
	Status     string    `db:"status"`
	ExpireAt   time.Time `db:"expire_at"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type DealActionLockExistsRow struct {
	One int `db:"one"`
}

type DealActionLockReturnRow struct {
	ID string `db:"id"`
}

func DealActionLockRowToEntity(row DealActionLockRow) *entity.DealActionLock {
	return &entity.DealActionLock{
		ID:         row.ID,
		DealID:     row.DealID,
		ActionType: entity.DealActionType(row.ActionType),
		Status:     entity.DealActionLockStatus(row.Status),
		ExpireAt:   row.ExpireAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
