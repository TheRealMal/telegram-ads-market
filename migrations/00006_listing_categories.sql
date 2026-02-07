-- +goose Up
ALTER TABLE market.listing ADD COLUMN IF NOT EXISTS categories JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE market.listing DROP COLUMN IF EXISTS categories;
