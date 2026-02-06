package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/updates"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (context.Context, error)
	EndTx(ctx context.Context, err error, source string) error
}

// stateRow represents a row from telegram_state table
type stateRow struct {
	Pts  int `db:"pts"`
	Qts  int `db:"qts"`
	Date int `db:"date"`
	Seq  int `db:"seq"`
}

// channelPtsRow represents a row from telegram_channel_state table
type channelPtsRow struct {
	ChannelID int64 `db:"channel_id"`
	Pts       int   `db:"pts"`
}

// channelPtsOnlyRow is used for GetChannelPts (single column).
type channelPtsOnlyRow struct {
	Pts int `db:"pts"`
}

// stateStorage implements updates.stateStorage interface for PostgreSQL
type stateStorage struct {
	db database
}

func New(db database) *stateStorage {
	return &stateStorage{db: db}
}

// GetState retrieves the current state from the database for a specific user
func (s *stateStorage) GetState(ctx context.Context, userID int64) (updates.State, bool, error) {
	rows, err := s.db.Query(ctx, `
		SELECT pts, qts, date, seq
		FROM telegram_state
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
	})
	if err != nil {
		return updates.State{}, false, fmt.Errorf("failed to query state: %w", err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[stateRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return zero state if no row exists
			return updates.State{
				Pts:  0,
				Qts:  0,
				Date: 0,
				Seq:  0,
			}, false, nil
		}
		return updates.State{}, false, fmt.Errorf("failed to get state: %w", err)
	}

	return updates.State{
		Pts:  row.Pts,
		Qts:  row.Qts,
		Date: row.Date,
		Seq:  row.Seq,
	}, true, nil
}

// SetState stores the state in the database for a specific user
func (s *stateStorage) SetState(ctx context.Context, userID int64, state updates.State) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO telegram_state (user_id, pts, qts, date, seq, updated_at)
		VALUES (@user_id, @pts, @qts, @date, @seq, NOW())
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			pts = EXCLUDED.pts,
			qts = EXCLUDED.qts,
			date = EXCLUDED.date,
			seq = EXCLUDED.seq,
			updated_at = NOW()
	`, pgx.NamedArgs{
		"user_id": userID,
		"pts":     state.Pts,
		"qts":     state.Qts,
		"date":    state.Date,
		"seq":     state.Seq,
	})
	if err != nil {
		return fmt.Errorf("failed to set state: %w", err)
	}

	log.Debug("State updated", "user_id", userID, "pts", state.Pts, "qts", state.Qts, "date", state.Date, "seq", state.Seq)
	return nil
}

// SetPts updates only the pts value for a specific user
func (s *stateStorage) SetPts(ctx context.Context, userID int64, pts int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE telegram_state
		SET pts = @pts, updated_at = NOW()
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
		"pts":     pts,
	})
	if err != nil {
		return fmt.Errorf("failed to set pts: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user state does not exist for user_id %d", userID)
	}

	return nil
}

// SetQts updates only the qts value for a specific user
func (s *stateStorage) SetQts(ctx context.Context, userID int64, qts int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE telegram_state
		SET qts = @qts, updated_at = NOW()
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
		"qts":     qts,
	})
	if err != nil {
		return fmt.Errorf("failed to set qts: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user state does not exist for user_id %d", userID)
	}

	return nil
}

// SetSeq updates only the seq value for a specific user
func (s *stateStorage) SetSeq(ctx context.Context, userID int64, seq int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE telegram_state
		SET seq = @seq, updated_at = NOW()
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
		"seq":     seq,
	})
	if err != nil {
		return fmt.Errorf("failed to set seq: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user state does not exist for user_id %d", userID)
	}

	return nil
}

// SetDate updates only the date value for a specific user
func (s *stateStorage) SetDate(ctx context.Context, userID int64, date int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE telegram_state
		SET date = @date, updated_at = NOW()
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
		"date":    date,
	})
	if err != nil {
		return fmt.Errorf("failed to set date: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user state does not exist for user_id %d", userID)
	}

	return nil
}

// SetDateSeq updates both date and seq values for a specific user
func (s *stateStorage) SetDateSeq(ctx context.Context, userID int64, date, seq int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE telegram_state
		SET date = @date, seq = @seq, updated_at = NOW()
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
		"date":    date,
		"seq":     seq,
	})
	if err != nil {
		return fmt.Errorf("failed to set date and seq: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user state does not exist for user_id %d", userID)
	}

	return nil
}

// GetChannelPts retrieves channel-specific pts from the database
func (s *stateStorage) GetChannelPts(ctx context.Context, userID, channelID int64) (int, bool, error) {
	rows, err := s.db.Query(ctx, `
		SELECT pts
		FROM telegram_channel_state
		WHERE user_id = @user_id AND channel_id = @channel_id
	`, pgx.NamedArgs{
		"user_id":    userID,
		"channel_id": channelID,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to query channel pts: %w", err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[channelPtsOnlyRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get channel pts: %w", err)
	}

	return row.Pts, true, nil
}

// SetChannelPts stores channel-specific pts in the database
func (s *stateStorage) SetChannelPts(ctx context.Context, userID, channelID int64, pts int) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO telegram_channel_state (user_id, channel_id, pts, updated_at)
		VALUES (@user_id, @channel_id, @pts, NOW())
		ON CONFLICT (user_id, channel_id)
		DO UPDATE SET
			pts = EXCLUDED.pts,
			updated_at = NOW()
	`, pgx.NamedArgs{
		"user_id":    userID,
		"channel_id": channelID,
		"pts":        pts,
	})
	if err != nil {
		return fmt.Errorf("failed to set channel pts: %w", err)
	}

	return nil
}

// ForEachChannels iterates over all channels for a user and calls f for each
func (s *stateStorage) ForEachChannels(ctx context.Context, userID int64, f func(ctx context.Context, channelID int64, pts int) error) error {
	rows, err := s.db.Query(ctx, `
		SELECT channel_id, pts
		FROM telegram_channel_state
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	channelRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[channelPtsRow])
	if err != nil {
		return fmt.Errorf("failed to collect channel rows: %w", err)
	}

	for _, row := range channelRows {
		if err := f(ctx, row.ChannelID, row.Pts); err != nil {
			return fmt.Errorf("callback error for channel %d: %w", row.ChannelID, err)
		}
	}

	return nil
}
