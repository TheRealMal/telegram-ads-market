-- +goose Up

CREATE TYPE market.listing_type AS ENUM(
    'lessor',   -- When user lists his channel
    'lessee'    -- When user lists his ads
);

CREATE TYPE market.listing_status AS ENUM(
    'active',
    'inactive'
);

CREATE TABLE IF NOT EXISTS market.listing (
    id          BIGSERIAL                   NOT NULL,
    status      market.listing_status       NOT NULL DEFAULT 'inactive',
    user_id     BIGINT                      NOT NULL,
    channel_id  BIGINT                      NULL,
    type        market.listing_type         NOT NULL,
    prices      JSONB                       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMP                   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP                   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES market.user(id),
    FOREIGN KEY (channel_id) REFERENCES market.channel(id)
);

CREATE TYPE market.deal_status AS ENUM(
    'draft',
    'approved',
    'waiting_escrow_deposit',
    'escrow_deposit_confirmed',
    'in_progress',
    'waiting_escrow_release',
    'escrow_release_confirmed',
    'completed',
    'expired',
    'rejected'
);

CREATE TABLE IF NOT EXISTS market.deal (
    id                  BIGSERIAL           NOT NULL,
    listing_id          BIGINT              NOT NULL,
    lessor_id           BIGINT              NOT NULL,
    lessee_id           BIGINT              NOT NULL,
    type                TEXT                NOT NULL,
    duration            BIGINT              NOT NULL,
    price               BIGINT              NOT NULL,
    details             JSONB               NOT NULL DEFAULT '{}',

    lessor_signature    TEXT                NULL,
    lessee_signature    TEXT                NULL,

    status              market.deal_status  NOT NULL DEFAULT 'draft',
    escrow_address      TEXT                NULL,
    escrow_private_key  TEXT                NULL,
    created_at          TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP           NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id),
    FOREIGN KEY (listing_id) REFERENCES market.listing(id),
    FOREIGN KEY (lessor_id) REFERENCES market.user(id),
    FOREIGN KEY (lessee_id) REFERENCES market.user(id)
);

-- +goose Down
DROP TABLE IF EXISTS market.deal;
DROP TYPE IF EXISTS market.deal_status;
DROP TABLE IF EXISTS market.listing;
DROP TYPE IF EXISTS market.listing_status;
DROP TYPE IF EXISTS market.listing_type;
