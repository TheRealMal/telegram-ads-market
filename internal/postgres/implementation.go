package postgres

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (p *postgres) Exec(ctx context.Context, stmt string, args ...any) (res pgconn.CommandTag, err error) {
	slog.Debug("Statement", "stmt", stmt, "args", args)
	before := time.Now()

	callerName := getCallerFuncName()

	// This closure collects Prometheus metrics depending on the status of the
	// query. It captures the named return parameter err to count errors.
	defer func() {
		labels := []string{callerName}

		latency := time.Since(before).Seconds()

		promDatabaseQueriesTotal.WithLabelValues(labels...).Inc()

		if err != nil {
			promDatabaseErrorsTotal.WithLabelValues(labels...).Inc()
		}

		promDatabaseQueryLatency.WithLabelValues(labels...).Observe(latency)
	}()

	return p.acquireQuerier(ctx).Exec(ctx, stmt, args...)
}

func (p *postgres) Query(ctx context.Context, stmt string, args ...any) (rows pgx.Rows, err error) {
	slog.Debug("Statement", "stmt", stmt, "args", args)
	before := time.Now()

	callerName := getCallerFuncName()

	// This closure collects Prometheus metrics depending on the status of the
	// query. It captures the named return parameter err to count errors.
	defer func() {
		labels := []string{callerName}

		latency := time.Since(before).Seconds()

		promDatabaseQueriesTotal.WithLabelValues(labels...).Inc()

		if err != nil {
			promDatabaseErrorsTotal.WithLabelValues(labels...).Inc()
		}

		promDatabaseQueryLatency.WithLabelValues(labels...).Observe(latency)
	}()

	return p.acquireQuerier(ctx).Query(ctx, stmt, args...)
}

func (p *postgres) SendBatch(ctx context.Context, batch *pgx.Batch) (err error) {
	slog.Debug("SendBatch")
	before := time.Now()

	callerName := getCallerFuncName()

	// This closure collects Prometheus metrics depending on the status of the
	// query. It captures the named return parameter err to count errors.
	defer func() {
		labels := []string{callerName}

		latency := time.Since(before).Seconds()

		promDatabaseQueriesTotal.WithLabelValues(labels...).Inc()

		if err != nil {
			promDatabaseErrorsTotal.WithLabelValues(labels...).Inc()
		}

		promDatabaseQueryLatency.WithLabelValues(labels...).Observe(latency)
	}()

	br := p.acquireQuerier(ctx).SendBatch(ctx, batch)
	defer br.Close()

	// Check inserted errors
	_, err = br.Exec()

	return err
}

// getCallerFuncName returns the name of the function that called the current method.
func getCallerFuncName() string {
	// Get the call stack.
	// 2 means we skip the current call (getCallerFuncName).
	pc, filePath, _, ok := runtime.Caller(2)
	if !ok {
		// Because everything starts from main
		return "main"
	}

	// Get the function information.
	funcName := runtime.FuncForPC(pc).Name()

	// Remove the package path, leaving only the function name.
	// For example, "main.example" -> "example".
	funcParts := strings.Split(funcName, ".")

	// Split the file path to get only the folder where the file is stored and the file name itself.
	fileParts := strings.Split(filePath, "/")

	return fileParts[len(fileParts)-2] + "/" + fileParts[len(fileParts)-1] + "/" + funcParts[len(funcParts)-1]
}
