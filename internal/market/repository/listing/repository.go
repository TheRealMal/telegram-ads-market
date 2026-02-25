package listing

import (
	"context"
	"encoding/json"
	"errors"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/market/repository/listing/model"

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

func (r *repository) CreateListing(ctx context.Context, l *entity.Listing) error {
	categories := l.Categories
	if categories == nil {
		categories = json.RawMessage("[]")
	}
	rows, err := r.db.Query(ctx, `
		INSERT INTO market.listing (status, user_id, channel_id, type, prices, categories, description)
		VALUES (@status, @user_id, @channel_id, @type, @prices, @categories, @description)
		RETURNING id, created_at, updated_at`,
		pgx.NamedArgs{
			"status":      l.Status,
			"user_id":     l.UserID,
			"channel_id":  l.ChannelID,
			"type":        l.Type,
			"prices":      l.Prices,
			"categories":  categories,
			"description": l.Description,
		})
	if err != nil {
		return err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ListingReturnRow])
	if err != nil {
		return err
	}
	l.ID = row.ID
	l.CreatedAt = row.CreatedAt
	l.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *repository) GetListingByID(ctx context.Context, id int64) (*entity.Listing, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.id, l.status, l.user_id, l.channel_id, l.type, l.prices, l.categories, l.description, l.created_at, l.updated_at,
		       c.title AS channel_title, c.username AS channel_username, c.photo AS channel_photo,
		       (cs.stats->'Followers'->>'Current')::bigint AS channel_followers
		FROM market.listing l
		LEFT JOIN market.channel c ON c.id = l.channel_id
		LEFT JOIN market.channel_stats cs ON cs.channel_id = l.channel_id
		WHERE l.id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ListingWithChannelRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return model.ListingWithChannelRowToEntity(row), nil
}

func (r *repository) UpdateListing(ctx context.Context, l *entity.Listing) error {
	categories := l.Categories
	if categories == nil {
		categories = json.RawMessage("[]")
	}
	_, err := r.db.Exec(ctx, `
		UPDATE market.listing
		SET status = @status, user_id = @user_id, channel_id = @channel_id, type = @type, prices = @prices, categories = @categories, description = @description, updated_at = NOW()
		WHERE id = @id`,
		pgx.NamedArgs{
			"id":          l.ID,
			"status":      l.Status,
			"user_id":     l.UserID,
			"channel_id":  l.ChannelID,
			"type":        l.Type,
			"prices":      l.Prices,
			"categories":  categories,
			"description": l.Description,
		})
	return err
}

func (r *repository) DeleteListing(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM market.listing WHERE id = @id`, pgx.NamedArgs{"id": id})
	return err
}

func (r *repository) ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error) {
	q := `
		SELECT l.id, l.status, l.user_id, l.channel_id, l.type, l.prices, l.categories, l.description, l.created_at, l.updated_at,
		       c.title AS channel_title, c.username AS channel_username, c.photo AS channel_photo,
		       (cs.stats->'Followers'->>'Current')::bigint AS channel_followers
		FROM market.listing l
		LEFT JOIN market.channel c ON c.id = l.channel_id
		LEFT JOIN market.channel_stats cs ON cs.channel_id = l.channel_id
		WHERE l.user_id = @user_id`
	args := pgx.NamedArgs{"user_id": userID}
	if typ != nil {
		q += ` AND l.type = @type`
		args["type"] = string(*typ)
	}
	q += ` ORDER BY l.updated_at DESC`

	rows, err := r.db.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.ListingWithChannelRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Listing, 0, len(slice))
	for _, row := range slice {
		list = append(list, model.ListingWithChannelRowToEntity(row))
	}
	return list, nil
}

func (r *repository) IsChannelHasActiveListing(ctx context.Context, channelID int64) (bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 1 AS one FROM market.listing
		WHERE channel_id = @channel_id AND status = 'active'
		LIMIT 1`,
		pgx.NamedArgs{"channel_id": channelID})
	if err != nil {
		return false, err
	}
	defer rows.Close()
	_, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ListingExistsRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *repository) ListListingsAll(ctx context.Context, typ *entity.ListingType, categories []string, minFollowers *int64) ([]*entity.Listing, error) {
	q := `
		SELECT l.id, l.status, l.user_id, l.channel_id, l.type, l.prices, l.categories, l.description, l.created_at, l.updated_at,
		       c.title AS channel_title, c.username AS channel_username, c.photo AS channel_photo,
		       (cs.stats->'Followers'->>'Current')::bigint AS channel_followers
		FROM market.listing l
		LEFT JOIN market.channel c ON c.id = l.channel_id
		LEFT JOIN market.channel_stats cs ON cs.channel_id = l.channel_id
		WHERE l.status = 'active'`
	args := pgx.NamedArgs{}
	if typ != nil {
		q += ` AND l.type = @type`
		args["type"] = string(*typ)
	}
	if len(categories) > 0 {
		q += ` AND l.categories ?| @categories_filter`
		args["categories_filter"] = categories
	}
	if minFollowers != nil && *minFollowers > 0 {
		q += ` AND (COALESCE((cs.stats->'Followers'->>'Current')::bigint, 0) >= @min_followers)`
		args["min_followers"] = *minFollowers
	}
	q += ` ORDER BY l.updated_at DESC`

	rows, err := r.db.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.ListingWithChannelRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Listing, 0, len(slice))
	for _, row := range slice {
		list = append(list, model.ListingWithChannelRowToEntity(row))
	}
	return list, nil
}
