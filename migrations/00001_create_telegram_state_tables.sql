-- +goose Up

CREATE SCHEMA IF NOT EXISTS userbot;

CREATE TABLE IF NOT EXISTS userbot.telegram_state (
    user_id BIGINT NOT NULL,
    pts INTEGER NOT NULL DEFAULT 0,
    qts INTEGER NOT NULL DEFAULT 0,
    date INTEGER NOT NULL DEFAULT 0,
    seq INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS userbot.telegram_channel_state (
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    pts INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_telegram_channel_state_user_id
ON userbot.telegram_channel_state(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_telegram_channel_state_user_id;
DROP TABLE IF EXISTS userbot.telegram_channel_state;
DROP TABLE IF EXISTS userbot.telegram_state;
DROP SCHEMA IF EXISTS userbot;