package user

import (
	"context"
	"errors"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/market/repository/user/model"
	"ads-mrkt/pkg/auth/role"

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

func (r *repository) UpsertUser(ctx context.Context, u *entity.User) error {
	rows, err := r.db.Query(ctx, `
		INSERT INTO market.user (id, username, photo, first_name, last_name, locale, referrer_id, allows_pm, role)
		VALUES (@id, @username, @photo, @first_name, @last_name, @locale, @referrer_id, @allows_pm, @role)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			photo = EXCLUDED.photo,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			locale = EXCLUDED.locale,
			allows_pm = EXCLUDED.allows_pm,
			updated_at = NOW()
		RETURNING role`,
		pgx.NamedArgs{
			"id":          u.ID,
			"username":    u.Username,
			"photo":       u.Photo,
			"first_name":  u.FirstName,
			"last_name":   u.LastName,
			"locale":      u.Locale,
			"referrer_id": u.ReferrerID,
			"allows_pm":   u.AllowsPM,
			"role":        string(u.Role),
		})
	if err != nil {
		return err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.RoleRow])
	if err != nil {
		return err
	}
	u.Role = role.Role(row.Role)
	return nil
}

func (r *repository) GetUserByID(ctx context.Context, id int64) (*entity.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, photo, first_name, last_name, locale, referrer_id, allows_pm, wallet_address, role
		FROM market.user WHERE id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.UserRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return model.UserRowToEntity(row), nil
}

func (r *repository) SetUserWallet(ctx context.Context, userID int64, walletAddressRaw string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.user SET wallet_address = @wallet_address, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"wallet_address": walletAddressRaw, "id": userID})
	return err
}

func (r *repository) ClearUserWallet(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE market.user SET wallet_address = NULL, updated_at = NOW() WHERE id = @id`,
		pgx.NamedArgs{"id": userID})
	return err
}
