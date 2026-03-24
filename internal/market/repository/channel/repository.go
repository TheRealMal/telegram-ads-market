package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
	"ads-mrkt/internal/market/repository/channel/model"
	"ads-mrkt/internal/market/repository/pgxutil"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// channelSelectCols is the standard column list for market.channel SELECT queries.
const channelSelectCols = "id, access_hash, bot_member_status, admin_rights, title, username, photo"

type database interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
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
	rows, err := r.db.Query(ctx,
		"SELECT "+channelSelectCols+" FROM market.channel WHERE id = @id",
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, fmt.Errorf("get channel by id %d: %w", id, err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ChannelRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, marketerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get channel by id %d: %w", id, err)
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
		return nil, fmt.Errorf("list channels by admin user id %d: %w", userID, err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvertErr(rows, model.ChannelRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list channels by admin user id %d: %w", userID, err)
	}
	return list, nil
}

func (r *repository) UpsertChannel(ctx context.Context, channel *entity.Channel) error {
	adminRightsJSON, err := json.Marshal(channel.AdminRights)
	if err != nil {
		return fmt.Errorf("upsert channel %d: marshal admin rights: %w", channel.ID, err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO market.channel (admin_rights, id, title, username, photo, access_hash, bot_member_status)
		VALUES (@admin_rights, @id, @title, @username, @photo, @access_hash, @bot_member_status)
		ON CONFLICT (id) DO UPDATE SET
			admin_rights = CASE WHEN EXCLUDED.admin_rights = '{"bot":{},"userbot":{}}' THEN market.channel.admin_rights ELSE EXCLUDED.admin_rights END,
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
	if err != nil {
		return fmt.Errorf("upsert channel %d: %w", channel.ID, err)
	}
	return nil
}

func (r *repository) UpdateChannelBotMemberStatus(ctx context.Context, channelID int64, status entity.BotMemberStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.channel SET bot_member_status = @status, admin_rights = jsonb_set(admin_rights, '{bot}', '{}'), updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID, "status": status})
	if err != nil {
		return fmt.Errorf("update channel %d bot member status: %w", channelID, err)
	}
	return nil
}

func (r *repository) UpdateBotAdminRights(ctx context.Context, channelID int64, rights entity.BotAdminRights) error {
	rightsJSON, err := json.Marshal(rights)
	if err != nil {
		return fmt.Errorf("update channel %d bot admin rights: marshal: %w", channelID, err)
	}
	_, err = r.db.Exec(ctx, `
		UPDATE market.channel SET admin_rights = jsonb_set(admin_rights, '{bot}', @bot_rights), updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID, "bot_rights": rightsJSON})
	if err != nil {
		return fmt.Errorf("update channel %d bot admin rights: %w", channelID, err)
	}
	return nil
}

func (r *repository) UpdateUserbotAdminRights(ctx context.Context, channelID int64, rights entity.UserbotAdminRights) error {
	rightsJSON, err := json.Marshal(rights)
	if err != nil {
		return fmt.Errorf("update channel %d userbot admin rights: marshal: %w", channelID, err)
	}
	_, err = r.db.Exec(ctx, `
		UPDATE market.channel SET admin_rights = jsonb_set(admin_rights, '{userbot}', @userbot_rights), updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID, "userbot_rights": rightsJSON})
	if err != nil {
		return fmt.Errorf("update channel %d userbot admin rights: %w", channelID, err)
	}
	return nil
}

func (r *repository) ListChannelsWithBotAccess(ctx context.Context) ([]*entity.Channel, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+channelSelectCols+" FROM market.channel WHERE bot_member_status = @status_administrator ORDER BY updated_at DESC",
		pgx.NamedArgs{"status_administrator": entity.BotMemberStatusAdministrator})
	if err != nil {
		return nil, fmt.Errorf("list channels with bot access: %w", err)
	}
	defer rows.Close()

	list, err := pgxutil.CollectAndConvertErr(rows, model.ChannelRowToEntity)
	if err != nil {
		return nil, fmt.Errorf("list channels with bot access: %w", err)
	}
	return list, nil
}

func (r *repository) ResetChannelAccessHash(ctx context.Context, channelID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.channel SET access_hash = 0, updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID})
	if err != nil {
		return fmt.Errorf("reset channel %d access hash: %w", channelID, err)
	}
	return nil
}

func (r *repository) UpdateChannelPhoto(ctx context.Context, channelID int64, photo string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.channel SET photo = @photo, updated_at = NOW() WHERE id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID, "photo": photo})
	if err != nil {
		return fmt.Errorf("update channel %d photo: %w", channelID, err)
	}
	return nil
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
	if err != nil {
		return fmt.Errorf("upsert channel %d stats: %w", channelID, err)
	}
	return nil
}

func (r *repository) GetChannelStats(ctx context.Context, channelID int64) (json.RawMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT stats FROM market.channel_stats WHERE channel_id = @channel_id`,
		pgx.NamedArgs{"channel_id": channelID})
	if err != nil {
		return nil, fmt.Errorf("get channel %d stats: %w", channelID, err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ChannelStatsRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, marketerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get channel %d stats: %w", channelID, err)
	}
	return row.Stats, nil
}

func (r *repository) MergeStatsRequestedAt(ctx context.Context, channelID int64, requestedAtUnix int64) error {
	raw, err := r.GetChannelStats(ctx, channelID)
	if err != nil {
		return fmt.Errorf("merge stats requested at for channel %d: %w", channelID, err)
	}
	var statsMap map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &statsMap); err != nil {
			return fmt.Errorf("merge stats requested at for channel %d: unmarshal stats: %w", channelID, err)
		}
	}
	if statsMap == nil {
		statsMap = make(map[string]interface{})
	}
	statsMap["requested_at"] = requestedAtUnix
	merged, err := json.Marshal(statsMap)
	if err != nil {
		return fmt.Errorf("merge stats requested at for channel %d: marshal stats: %w", channelID, err)
	}
	return r.UpsertChannelStats(ctx, channelID, merged)
}
