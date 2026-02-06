package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type channelAdminExistsRow struct {
	One int `db:"one"`
}

// DeleteChannelAdmins removes all admin records for the given channel.
// Used when syncing admins from Telegram so the DB reflects current state.
func (r *repository) DeleteChannelAdmins(ctx context.Context, channelID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM market.channel_admin WHERE channel_id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID})
	return err
}

// UpsertChannelAdmin inserts or updates a channel admin. Role must be 'owner' or 'admin' (market.role).
// Ensures the user exists in market.user via CTE (insert with minimal fields if missing) so the channel_admin FK is satisfied.
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

// IsChannelAdmin returns true if the user is an admin (owner or admin role) of the channel.
// Lessors can only be admins of channels.
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

	_, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[channelAdminExistsRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
