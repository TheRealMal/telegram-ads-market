package repository

import (
	"context"

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
