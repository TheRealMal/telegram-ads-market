-- +goose Up
ALTER TABLE market.user ADD COLUMN IF NOT EXISTS wallet_address TEXT NULL;
-- +goose Down
ALTER TABLE market.user DROP COLUMN IF EXISTS wallet_address;
