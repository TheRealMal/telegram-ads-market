-- +goose Up
ALTER TABLE market.deal DROP COLUMN IF EXISTS escrow_private_key;
-- +goose Down
ALTER TABLE market.deal ADD COLUMN escrow_private_key TEXT NULL;
