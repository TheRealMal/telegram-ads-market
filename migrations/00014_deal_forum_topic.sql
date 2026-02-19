-- +goose Up
-- Deal chat: one row per deal; one topic per side in each user's chat (lessor_chat_id = lessor Telegram user id, lessee_chat_id = lessee). Messages mirrored via copyMessage. Topics deleted via deleteForumTopic on finalize.
CREATE TABLE IF NOT EXISTS market.deal_forum_topic (
    deal_id                      BIGINT    NOT NULL,
    lessor_chat_id               BIGINT    NOT NULL,
    lessee_chat_id               BIGINT    NOT NULL,
    lessor_message_thread_id     INT       NOT NULL,
    lessee_message_thread_id     INT       NOT NULL,
    created_at                   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (deal_id),
    FOREIGN KEY (deal_id) REFERENCES market.deal(id)
);

-- Remove old deal chat (invite + reply history).
DROP TABLE IF EXISTS market.deal_chat;

-- +goose Down
CREATE TABLE IF EXISTS market.deal_chat (
    deal_id             BIGINT    NOT NULL,
    reply_to_chat_id    BIGINT    NOT NULL,
    reply_to_message_id BIGINT    NOT NULL,
    replied_message     TEXT      NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (deal_id, reply_to_chat_id, reply_to_message_id),
    FOREIGN KEY (deal_id) REFERENCES market.deal(id)
);

DROP TABLE IF EXISTS market.deal_forum_topic;
