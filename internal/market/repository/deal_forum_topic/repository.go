package deal_forum_topic

import (
	"context"
	"errors"
	"time"

	"ads-mrkt/internal/market/domain/entity"

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

func (r *repository) InsertDealForumTopic(ctx context.Context, t *entity.DealForumTopic) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO market.deal_forum_topic (deal_id, lessor_chat_id, lessee_chat_id, lessor_message_thread_id, lessee_message_thread_id)
		VALUES (@deal_id, @lessor_chat_id, @lessee_chat_id, @lessor_message_thread_id, @lessee_message_thread_id)`,
		pgx.NamedArgs{
			"deal_id":                  t.DealID,
			"lessor_chat_id":           t.LessorChatID,
			"lessee_chat_id":           t.LesseeChatID,
			"lessor_message_thread_id": t.LessorMessageThreadID,
			"lessee_message_thread_id": t.LesseeMessageThreadID,
		})
	return err
}

type dealForumTopicRow struct {
	DealID                int64     `db:"deal_id"`
	LessorChatID          int64     `db:"lessor_chat_id"`
	LesseeChatID          int64     `db:"lessee_chat_id"`
	LessorMessageThreadID int64     `db:"lessor_message_thread_id"`
	LesseeMessageThreadID int64     `db:"lessee_message_thread_id"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}

func (r *repository) GetDealForumTopicByDealID(ctx context.Context, dealID int64) (*entity.DealForumTopic, error) {
	rows, err := r.db.Query(ctx, `
		SELECT deal_id, lessor_chat_id, lessee_chat_id, lessor_message_thread_id, lessee_message_thread_id, created_at, updated_at
		FROM market.deal_forum_topic WHERE deal_id = @deal_id`,
		pgx.NamedArgs{"deal_id": dealID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dealForumTopicRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entity.DealForumTopic{
		DealID:                row.DealID,
		LessorChatID:          row.LessorChatID,
		LesseeChatID:          row.LesseeChatID,
		LessorMessageThreadID: row.LessorMessageThreadID,
		LesseeMessageThreadID: row.LesseeMessageThreadID,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}, nil
}

func (r *repository) GetDealForumTopicByChatAndThread(ctx context.Context, chatID int64, messageThreadID int64) (*entity.DealForumTopic, string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT deal_id, lessor_chat_id, lessee_chat_id, lessor_message_thread_id, lessee_message_thread_id, created_at, updated_at
		FROM market.deal_forum_topic
		WHERE (lessor_chat_id = @chat_id AND lessor_message_thread_id = @thread_id)
		   OR (lessee_chat_id = @chat_id AND lessee_message_thread_id = @thread_id)`,
		pgx.NamedArgs{"chat_id": chatID, "thread_id": messageThreadID})
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dealForumTopicRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", err
	}
	t := &entity.DealForumTopic{
		DealID:                row.DealID,
		LessorChatID:          row.LessorChatID,
		LesseeChatID:          row.LesseeChatID,
		LessorMessageThreadID: row.LessorMessageThreadID,
		LesseeMessageThreadID: row.LesseeMessageThreadID,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
	side := "lessee"
	if t.LessorChatID == chatID && t.LessorMessageThreadID == messageThreadID {
		side = "lessor"
	}
	return t, side, nil
}

func (r *repository) DeleteDealForumTopic(ctx context.Context, dealID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM market.deal_forum_topic WHERE deal_id = @deal_id`, pgx.NamedArgs{"deal_id": dealID})
	return err
}
