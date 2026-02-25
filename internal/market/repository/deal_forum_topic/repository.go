package deal_forum_topic

import (
	"context"
	"errors"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/market/repository/deal_forum_topic/model"

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

func (r *repository) GetDealForumTopicByDealID(ctx context.Context, dealID int64) (*entity.DealForumTopic, error) {
	rows, err := r.db.Query(ctx, `
		SELECT deal_id, lessor_chat_id, lessee_chat_id, lessor_message_thread_id, lessee_message_thread_id, created_at, updated_at
		FROM market.deal_forum_topic WHERE deal_id = @deal_id`,
		pgx.NamedArgs{"deal_id": dealID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealForumTopicRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return model.DealForumTopicRowToEntity(row), nil
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
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.DealForumTopicRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", err
	}
	t := model.DealForumTopicRowToEntity(row)
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
