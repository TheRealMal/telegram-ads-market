-- +goose Up
ALTER TABLE market.deal ADD COLUMN message TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE market.deal DROP COLUMN message;
