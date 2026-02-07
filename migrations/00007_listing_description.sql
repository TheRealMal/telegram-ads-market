-- +goose Up
ALTER TABLE market.listing ADD COLUMN IF NOT EXISTS description TEXT;

-- +goose Down
ALTER TABLE market.listing DROP COLUMN IF EXISTS description;
