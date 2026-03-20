package userbot_state

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type database interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type repository struct {
	db database
}

func New(db database) *repository {
	return &repository{db: db}
}

type userbotUserIDRow struct {
	UserID int64 `db:"user_id"`
}

func (r *repository) GetUserbotUserID(ctx context.Context) (int64, error) {
	rows, err := r.db.Query(ctx, `SELECT user_id FROM userbot.telegram_state LIMIT 1`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[userbotUserIDRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("userbot state not found")
		}
		return 0, err
	}
	return row.UserID, nil
}
