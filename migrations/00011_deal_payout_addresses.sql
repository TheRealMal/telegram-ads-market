-- +goose Up
ALTER TABLE market.deal ADD COLUMN IF NOT EXISTS lessor_payout_address TEXT NULL;
ALTER TABLE market.deal ADD COLUMN IF NOT EXISTS lessee_payout_address TEXT NULL;
-- +goose Down
ALTER TABLE market.deal DROP COLUMN IF EXISTS lessor_payout_address;
ALTER TABLE market.deal DROP COLUMN IF EXISTS lessee_payout_address;
