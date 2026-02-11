-- +goose Up

-- escrow_amount = price + transaction_gas (configurable) + commission (configurable % of price). Backfilled with 0.1 TON gas + 2% commission.
ALTER TABLE market.deal ADD COLUMN IF NOT EXISTS escrow_amount BIGINT NOT NULL DEFAULT 0;

UPDATE market.deal
SET escrow_amount = price + 100000000 + (price * 2 / 100)
WHERE escrow_amount = 0;

-- Locks for escrow release/refund and for posting message: avoid double-send and allow recovery after crash (expire_at).
CREATE TABLE IF NOT EXISTS market.deal_action_lock (
    id          UUID      NOT NULL DEFAULT gen_random_uuid(),
    deal_id     BIGINT    NOT NULL,
    action_type TEXT      NOT NULL,
    status      TEXT      NOT NULL DEFAULT 'locked',
    expire_at   TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id),
    FOREIGN KEY (deal_id) REFERENCES market.deal(id)
);

CREATE INDEX IF NOT EXISTS idx_deal_action_lock_deal_action_active
ON market.deal_action_lock (deal_id, action_type)
WHERE status = 'locked';

-- +goose Down
DROP INDEX IF EXISTS market.idx_deal_action_lock_deal_action_active;
DROP TABLE IF EXISTS market.deal_action_lock;
ALTER TABLE market.deal DROP COLUMN IF EXISTS escrow_amount;
