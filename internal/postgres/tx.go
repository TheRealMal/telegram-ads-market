package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type txKey struct{}

type quierier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

func (p *postgres) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (context.Context, error) {
	slog.Debug("StartTx", "caller", getCallerFuncName())

	if _, exists := ctx.Value(txKey{}).(pgx.Tx); exists {
		return nil, fmt.Errorf("postgres: duplicate caller function name %q", "BeginTx")
	}

	tx, err := p.pg.BeginTx(ctx, txOptions)
	if err != nil {
		return ctx, fmt.Errorf("begin tx: %w", err)
	}

	return context.WithValue(ctx, txKey{}, tx), nil
}

func (p *postgres) EndTx(ctx context.Context, err error, source string) error {
	slog.Debug("EndTx", "caller", getCallerFuncName())

	_, exists := ctx.Value(txKey{}).(pgx.Tx)
	if !exists {
		return fmt.Errorf("postgres: tx not found in context")
	}

	if err != nil {
		slog.Error(source, "error", err)

		if errRollback := p.rollbackIfTx(ctx); errRollback != nil {
			slog.Error(source, "error", errRollback)
		}

		return err
	}

	if err = p.commitTx(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (p *postgres) acquireQuerier(ctx context.Context) quierier {
	q := quierier(p.pg)

	if val, exists := ctx.Value(txKey{}).(pgx.Tx); exists {
		q = val
	}

	return q
}

func (p *postgres) commitTx(ctx context.Context) error {
	slog.Debug("EndTx")

	tx, exists := ctx.Value(txKey{}).(pgx.Tx)
	if !exists {
		return fmt.Errorf("postgres: tx not found in context")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (p *postgres) rollbackIfTx(ctx context.Context) error {
	slog.Debug("rollback")

	tx, exists := ctx.Value(txKey{}).(pgx.Tx)
	if !exists {
		return nil
	}

	if err := tx.Rollback(ctx); err != nil {
		return fmt.Errorf("rollback tx: %w", err)
	}

	return nil
}
