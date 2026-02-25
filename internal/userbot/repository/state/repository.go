package repository

import (
	"context"
	"errors"
	"fmt"

	"ads-mrkt/internal/userbot/repository/state/model"

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

// stateStorage implements updates.stateStorage interface for PostgreSQL
type stateStorage struct {
	db database
}

func New(db database) *stateStorage {
	return &stateStorage{db: db}
}

func (s *stateStorage) GetState(ctx context.Context, userID int64) (updates.State, bool, error) {
	rows, err := s.db.Query(ctx, `
		SELECT pts, qts, date, seq
		FROM userbot.telegram_state
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
	})
	if err != nil {
		return updates.State{}, false, fmt.Errorf("failed to query state: %w", err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.StateRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

func (s *stateStorage) SetState(ctx context.Context, userID int64, state updates.State) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO userbot.telegram_state (user_id, pts, qts, date, seq, updated_at)
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

func (s *stateStorage) SetPts(ctx context.Context, userID int64, pts int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE userbot.telegram_state
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

func (s *stateStorage) SetQts(ctx context.Context, userID int64, qts int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE userbot.telegram_state
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

func (s *stateStorage) SetSeq(ctx context.Context, userID int64, seq int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE userbot.telegram_state
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

func (s *stateStorage) SetDate(ctx context.Context, userID int64, date int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE userbot.telegram_state
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

func (s *stateStorage) SetDateSeq(ctx context.Context, userID int64, date, seq int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE userbot.telegram_state
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

func (s *stateStorage) GetChannelPts(ctx context.Context, userID, channelID int64) (int, bool, error) {
	rows, err := s.db.Query(ctx, `
		SELECT pts
		FROM userbot.telegram_channel_state
		WHERE user_id = @user_id AND channel_id = @channel_id
	`, pgx.NamedArgs{
		"user_id":    userID,
		"channel_id": channelID,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to query channel pts: %w", err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.ChannelPtsOnlyRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get channel pts: %w", err)
	}

	return row.Pts, true, nil
}

func (s *stateStorage) SetChannelPts(ctx context.Context, userID, channelID int64, pts int) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO userbot.telegram_channel_state (user_id, channel_id, pts, updated_at)
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

func (s *stateStorage) ForEachChannels(ctx context.Context, userID int64, f func(ctx context.Context, channelID int64, pts int) error) error {
	rows, err := s.db.Query(ctx, `
		SELECT channel_id, pts
		FROM userbot.telegram_channel_state
		WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	channelRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.ChannelPtsRow])
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
