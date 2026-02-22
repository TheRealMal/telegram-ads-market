-- +goose Up

CREATE SCHEMA IF NOT EXISTS analytic;

-- Hourly snapshot: listings count, deals count, deals by status (pie), deal amounts by status in TON (pie), commission earned (completed only), users count, avg listings per user.
CREATE TABLE IF NOT EXISTS analytic.snapshot (
    id                          BIGSERIAL       NOT NULL,
    recorded_at                  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    listings_count               BIGINT          NOT NULL DEFAULT 0,
    deals_count                  BIGINT          NOT NULL DEFAULT 0,
    deals_by_status              JSONB           NOT NULL DEFAULT '{}',
    deal_amounts_by_status_ton   JSONB           NOT NULL DEFAULT '{}',
    commission_earned_nanoton    BIGINT          NOT NULL DEFAULT 0,
    users_count                  BIGINT          NOT NULL DEFAULT 0,
    avg_listings_per_user        NUMERIC(20, 6)  NOT NULL DEFAULT 0,
    PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_analytic_snapshot_recorded_at ON analytic.snapshot (recorded_at DESC);

-- +goose Down

DROP INDEX IF EXISTS analytic.idx_analytic_snapshot_recorded_at;
DROP TABLE IF EXISTS analytic.snapshot;
DROP SCHEMA IF EXISTS analytic;
