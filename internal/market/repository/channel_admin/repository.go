package channel_admin

import (
	"context"
	"errors"

	"ads-mrkt/internal/market/repository/channel_admin/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (context.Context, error)
	EndTx(ctx context.Context, err error, source string) error
}

type repository struct {
	db database
}

func New(db database) *repository {
	return &repository{db: db}
}

func (r *repository) DeleteChannelAdmins(ctx context.Context, channelID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM market.channel_admin WHERE channel_id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID})
	return err
}

func (r *repository) UpsertChannelAdmin(ctx context.Context, userID, channelID int64, role string) error {
	_, err := r.db.Exec(ctx, `
		WITH ensure_user AS (
			INSERT INTO market.user (id, username, photo, first_name, last_name, locale)
			VALUES (@user_id, '', '', '', '', '')
			ON CONFLICT (id) DO NOTHING
			RETURNING id
		)
		INSERT INTO market.channel_admin (user_id, channel_id, role)
		VALUES (@user_id, @channel_id, @role::market.role)
		ON CONFLICT (user_id, channel_id) DO UPDATE SET role = EXCLUDED.role, updated_at = NOW()`,
		pgx.NamedArgs{
			"user_id":    userID,
			"channel_id": channelID,
			"role":       role,
		})
	return err
}

func (r *repository) IsChannelAdmin(ctx context.Context, userID, channelID int64) (bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 1 AS one FROM market.channel_admin
		WHERE user_id = @user_id AND channel_id = @channel_id
		LIMIT 1`,
		pgx.NamedArgs{
			"user_id":    userID,
			"channel_id": channelID,
		})
	if err != nil {
		return false, err
	}
	defer rows.Close()

	_, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ChannelAdminExistsRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
