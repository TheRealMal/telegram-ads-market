package repository

import (
	"context"
	"errors"

	"ads-mrkt/internal/market/domain/entity"

	"github.com/jackc/pgx/v5"
)

type userRow struct {
	ID         int64  `db:"id"`
	Username   string `db:"username"`
	Photo      string `db:"photo"`
	FirstName  string `db:"first_name"`
	LastName   string `db:"last_name"`
	Locale     string `db:"locale"`
	ReferrerID int64  `db:"referrer_id"`
	AllowsPM   bool   `db:"allows_pm"`
}

func (r *repository) UpsertUser(ctx context.Context, u *entity.User) error {
	// Not updating referrer_id as it's not updatable
	_, err := r.db.Exec(ctx, `
		INSERT INTO market.user (id, username, photo, first_name, last_name, locale, referrer_id, allows_pm)
		VALUES (@id, @username, @photo, @first_name, @last_name, @locale, @referrer_id, @allows_pm)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			photo = EXCLUDED.photo,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			locale = EXCLUDED.locale,
			allows_pm = EXCLUDED.allows_pm,
			updated_at = NOW()`,
		pgx.NamedArgs{
			"id":          u.ID,
			"username":    u.Username,
			"photo":       u.Photo,
			"first_name":  u.FirstName,
			"last_name":   u.LastName,
			"locale":      u.Locale,
			"referrer_id": u.ReferrerID,
			"allows_pm":   u.AllowsPM,
		})
	return err
}

func (r *repository) GetUserByID(ctx context.Context, id int64) (*entity.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, photo, first_name, last_name, locale, referrer_id, allows_pm
		FROM market.user WHERE id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[userRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &entity.User{
		ID:         row.ID,
		Username:   row.Username,
		Photo:      row.Photo,
		FirstName:  row.FirstName,
		LastName:   row.LastName,
		Locale:     row.Locale,
		ReferrerID: row.ReferrerID,
		AllowsPM:   row.AllowsPM,
	}, nil
}
