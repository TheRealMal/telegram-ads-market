package channel

import (
	"context"
	"encoding/json"
	"errors"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/market/repository/channel/model"

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

func (r *repository) GetChannelByID(ctx context.Context, id int64) (*entity.Channel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, access_hash, bot_member_status, admin_rights, title, username, photo
		FROM market.channel WHERE id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ChannelRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return model.ChannelRowToEntity(row)
}

func (r *repository) ListChannelsByAdminUserID(ctx context.Context, userID int64) ([]*entity.Channel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.access_hash, c.bot_member_status, c.admin_rights, c.title, c.username, c.photo
		FROM market.channel c
		INNER JOIN market.channel_admin ca ON ca.channel_id = c.id
		WHERE ca.user_id = @user_id
		ORDER BY c.updated_at DESC`,
		pgx.NamedArgs{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.ChannelRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Channel, 0, len(slice))
	for _, row := range slice {
		ch, err := model.ChannelRowToEntity(row)
		if err != nil {
			return nil, err
		}
		list = append(list, ch)
	}
	return list, nil
}

func (r *repository) UpsertChannel(ctx context.Context, channel *entity.Channel) error {
	adminRightsJSON, err := json.Marshal(channel.AdminRights)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO market.channel (admin_rights, id, title, username, photo, access_hash, bot_member_status)
		VALUES (@admin_rights, @id, @title, @username, @photo, @access_hash, @bot_member_status)
		ON CONFLICT (id) DO UPDATE SET
			admin_rights = EXCLUDED.admin_rights,
			title = EXCLUDED.title,
			username = EXCLUDED.username,
			photo = CASE WHEN EXCLUDED.photo = '' THEN market.channel.photo ELSE EXCLUDED.photo END,
			access_hash = CASE WHEN EXCLUDED.access_hash = 0 THEN market.channel.access_hash ELSE EXCLUDED.access_hash END,
			bot_member_status = CASE WHEN EXCLUDED.bot_member_status = '' THEN market.channel.bot_member_status ELSE EXCLUDED.bot_member_status END,
			updated_at = NOW();
	`, pgx.NamedArgs{
		"admin_rights":      adminRightsJSON,
		"id":                channel.ID,
		"title":             channel.Title,
		"username":          channel.Username,
		"photo":             channel.Photo,
		"access_hash":       channel.AccessHash,
		"bot_member_status": channel.BotMemberStatus,
	})
	return err
}

func (r *repository) UpdateChannelBotMemberStatus(ctx context.Context, channelID int64, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.channel SET bot_member_status = @status, updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID, "status": status})
	return err
}

func (r *repository) ListChannelsWithBotAccess(ctx context.Context) ([]*entity.Channel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, access_hash, bot_member_status, admin_rights, title, username, photo
		FROM market.channel
		WHERE bot_member_status = 'administrator'
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.ChannelRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Channel, 0, len(slice))
	for _, row := range slice {
		ch, err := model.ChannelRowToEntity(row)
		if err != nil {
			return nil, err
		}
		list = append(list, ch)
	}
	return list, nil
}

func (r *repository) ResetChannelAccessHash(ctx context.Context, channelID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.channel SET access_hash = 0, updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID})
	return err
}

func (r *repository) UpdateChannelPhoto(ctx context.Context, channelID int64, photo string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.channel SET photo = @photo, updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID, "photo": photo})
	return err
}

func (r *repository) UpsertChannelStats(ctx context.Context, channelID int64, stats json.RawMessage) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO market.channel_stats (channel_id, stats)
		VALUES (@channel_id, @stats)
		ON CONFLICT (channel_id) DO UPDATE SET
			stats = EXCLUDED.stats,
			updated_at = NOW();
	`, pgx.NamedArgs{
		"channel_id": channelID,
		"stats":      stats,
	})
	return err
}

func (r *repository) GetChannelStats(ctx context.Context, channelID int64) (json.RawMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT stats FROM market.channel_stats WHERE channel_id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ChannelStatsRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.Stats, nil
}

func (r *repository) MergeStatsRequestedAt(ctx context.Context, channelID int64, requestedAtUnix int64) error {
	raw, err := r.GetChannelStats(ctx, channelID)
	if err != nil {
		return err
	}
	var statsMap map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &statsMap); err != nil {
			return err
		}
	}
	if statsMap == nil {
		statsMap = make(map[string]interface{})
	}
	statsMap["requested_at"] = requestedAtUnix
	merged, err := json.Marshal(statsMap)
	if err != nil {
		return err
	}
	return r.UpsertChannelStats(ctx, channelID, merged)
}
