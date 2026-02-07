package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
)

type listingRow struct {
	ID          int64           `db:"id"`
	Status      string          `db:"status"`
	UserID      int64           `db:"user_id"`
	ChannelID   *int64          `db:"channel_id"`
	Type        string          `db:"type"`
	Prices      json.RawMessage `db:"prices"`
	Categories  json.RawMessage `db:"categories"`
	Description *string         `db:"description"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

type listingWithChannelRow struct {
	listingRow
	ChannelTitle     *string `db:"channel_title"`
	ChannelUsername  *string `db:"channel_username"`
	ChannelFollowers *int64  `db:"channel_followers"`
}

func stringFromPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

type listingReturnRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func listingRowToEntity(row listingRow) *entity.Listing {
	l := &entity.Listing{
		ID:          row.ID,
		Status:      entity.ListingStatus(row.Status),
		UserID:      row.UserID,
		ChannelID:   row.ChannelID,
		Type:        entity.ListingType(row.Type),
		Prices:      row.Prices,
		Description: stringFromPtr(row.Description),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if len(row.Categories) > 0 {
		l.Categories = row.Categories
	}
	return l
}

func listingWithChannelRowToEntity(row listingWithChannelRow) *entity.Listing {
	l := listingRowToEntity(row.listingRow)
	l.ChannelTitle = row.ChannelTitle
	l.ChannelUsername = row.ChannelUsername
	l.ChannelFollowers = row.ChannelFollowers
	return l
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

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[listingReturnRow])
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
		       c.title AS channel_title, c.username AS channel_username,
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

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[listingWithChannelRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return listingWithChannelRowToEntity(row), nil
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

// DeleteListing deletes a listing by ID.
func (r *repository) DeleteListing(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM market.listing WHERE id = @id`, pgx.NamedArgs{"id": id})
	return err
}

// ListListingsByUserID returns listings for a user (optional filter by type).
func (r *repository) ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error) {
	q := `
		SELECT l.id, l.status, l.user_id, l.channel_id, l.type, l.prices, l.categories, l.description, l.created_at, l.updated_at,
		       c.title AS channel_title, c.username AS channel_username,
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

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[listingWithChannelRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Listing, 0, len(slice))
	for _, row := range slice {
		list = append(list, listingWithChannelRowToEntity(row))
	}
	return list, nil
}

type listingExistsRow struct {
	One int `db:"one"`
}

// IsChannelHasActiveListing returns true if the channel has at least one active listing.
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
	_, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[listingExistsRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListListingsAll returns active listings only (optional filter by type, categories, min followers). Used for public discovery.
func (r *repository) ListListingsAll(ctx context.Context, typ *entity.ListingType, categories []string, minFollowers *int64) ([]*entity.Listing, error) {
	q := `
		SELECT l.id, l.status, l.user_id, l.channel_id, l.type, l.prices, l.categories, l.description, l.created_at, l.updated_at,
		       c.title AS channel_title, c.username AS channel_username,
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

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[listingWithChannelRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Listing, 0, len(slice))
	for _, row := range slice {
		list = append(list, listingWithChannelRowToEntity(row))
	}
	return list, nil
}
