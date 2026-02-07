-- +goose Up

CREATE TABLE IF NOT EXISTS market.deal_chat (
    deal_id             BIGINT              NOT NULL,
    reply_to_chat_id    BIGINT              NOT NULL,
    reply_to_message_id BIGINT              NOT NULL,
    replied_message     TEXT                NULL,
    created_at          TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP           NOT NULL DEFAULT NOW(),
    PRIMARY KEY (deal_id, reply_to_chat_id, reply_to_message_id),
    FOREIGN KEY (deal_id) REFERENCES market.deal(id),
    FOREIGN KEY (reply_to_chat_id) REFERENCES market.user(id)
);

-- +goose Down
DROP TABLE IF EXISTS market.deal_chat;
