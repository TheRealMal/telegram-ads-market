-- +goose Up
ALTER TABLE market.channel ADD COLUMN IF NOT EXISTS bot_member_status TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE market.channel DROP COLUMN IF EXISTS bot_member_status;
