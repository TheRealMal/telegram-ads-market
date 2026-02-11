-- +goose Up

CREATE TYPE market.deal_post_message_status AS ENUM(
    'exists',   -- message was sent and is expected to exist
    'deleted',  -- message was deleted before until
    'passed',   -- we passed the until time and last check was successful
    'completed' -- deal was moved to waiting_escrow_release
);

CREATE TABLE IF NOT EXISTS market.deal_post_message (
    id          BIGSERIAL                           NOT NULL,
    deal_id     BIGINT                              NOT NULL,
    channel_id  BIGINT                              NOT NULL,
    message_id  BIGINT                              NOT NULL,
    post_message TEXT                              NOT NULL,
    status      market.deal_post_message_status     NOT NULL DEFAULT 'exists',
    next_check  TIMESTAMP                           NOT NULL,
    until_ts    TIMESTAMP                           NOT NULL,
    created_at  TIMESTAMP                           NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP                           NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id),
    FOREIGN KEY (deal_id) REFERENCES market.deal(id),
    UNIQUE (deal_id)
);

CREATE INDEX IF NOT EXISTS idx_deal_post_message_status_next_check
    ON market.deal_post_message (status, next_check)
    WHERE status = 'exists';

CREATE INDEX IF NOT EXISTS idx_deal_post_message_status_passed
    ON market.deal_post_message (id)
    WHERE status = 'passed';

-- +goose Down
DROP TABLE IF EXISTS market.deal_post_message;
DROP TYPE IF EXISTS market.deal_post_message_status;
