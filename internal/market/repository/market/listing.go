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
	ID        int64           `db:"id"`
	Status    string          `db:"status"`
	UserID    int64           `db:"user_id"`
	ChannelID *int64          `db:"channel_id"`
	Type      string          `db:"type"`
	Prices    json.RawMessage `db:"prices"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

type listingReturnRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func listingRowToEntity(row listingRow) *entity.Listing {
	return &entity.Listing{
		ID:        row.ID,
		Status:    entity.ListingStatus(row.Status),
		UserID:    row.UserID,
		ChannelID: row.ChannelID,
		Type:      entity.ListingType(row.Type),
		Prices:    row.Prices,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *repository) CreateListing(ctx context.Context, l *entity.Listing) error {
	rows, err := r.db.Query(ctx, `
		INSERT INTO market.listing (status, user_id, channel_id, type, prices)
		VALUES (@status, @user_id, @channel_id, @type, @prices)
		RETURNING id, created_at, updated_at`,
		pgx.NamedArgs{
			"status":     l.Status,
			"user_id":    l.UserID,
			"channel_id": l.ChannelID,
			"type":       l.Type,
			"prices":     l.Prices,
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
		SELECT id, status, user_id, channel_id, type, prices, created_at, updated_at
		FROM market.listing WHERE id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[listingRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return listingRowToEntity(row), nil
}

func (r *repository) UpdateListing(ctx context.Context, l *entity.Listing) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.listing
		SET status = @status, user_id = @user_id, channel_id = @channel_id, type = @type, prices = @prices, updated_at = NOW()
		WHERE id = @id`,
		pgx.NamedArgs{
			"id":         l.ID,
			"status":     l.Status,
			"user_id":    l.UserID,
			"channel_id": l.ChannelID,
			"type":       l.Type,
			"prices":     l.Prices,
		})
	return err
}

// ListListingsByUserID returns listings for a user (optional filter by type).
func (r *repository) ListListingsByUserID(ctx context.Context, userID int64, typ *entity.ListingType) ([]*entity.Listing, error) {
	q := `
		SELECT id, status, user_id, channel_id, type, prices, created_at, updated_at
		FROM market.listing WHERE user_id = @user_id`
	args := pgx.NamedArgs{"user_id": userID}
	if typ != nil {
		q += ` AND type = @type`
		args["type"] = string(*typ)
	}
	q += ` ORDER BY updated_at DESC`

	rows, err := r.db.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[listingRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Listing, 0, len(slice))
	for _, row := range slice {
		list = append(list, listingRowToEntity(row))
	}
	return list, nil
}

// ListListingsAll returns all listings (optional filter by type). Used for public discovery.
func (r *repository) ListListingsAll(ctx context.Context, typ *entity.ListingType) ([]*entity.Listing, error) {
	q := `
		SELECT id, status, user_id, channel_id, type, prices, created_at, updated_at
		FROM market.listing WHERE 1=1`
	args := pgx.NamedArgs{}
	if typ != nil {
		q += ` AND type = @type`
		args["type"] = string(*typ)
	}
	q += ` ORDER BY updated_at DESC`

	rows, err := r.db.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[listingRow])
	if err != nil {
		return nil, err
	}
	list := make([]*entity.Listing, 0, len(slice))
	for _, row := range slice {
		list = append(list, listingRowToEntity(row))
	}
	return list, nil
}
