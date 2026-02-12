-- +goose Up
ALTER TABLE market.deal
    ADD COLUMN IF NOT EXISTS channel_id BIGINT NULL REFERENCES market.channel(id);

-- +goose Down
ALTER TABLE market.deal DROP COLUMN IF EXISTS channel_id;
