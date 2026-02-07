-- +goose Up

CREATE SCHEMA IF NOT EXISTS market;

CREATE TABLE IF NOT EXISTS market.user (
    id          BIGINT      NOT NULL,
    username    TEXT        NOT NULL,
    photo       TEXT        NOT NULL,
    first_name  TEXT        NOT NULL DEFAULT '',
    last_name   TEXT        NOT NULL DEFAULT '',
    locale      TEXT        NOT NULL DEFAULT '',
    referrer_id BIGINT      NOT NULL DEFAULT 0,
    allows_pm   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS market.channel (
    id              BIGINT      NOT NULL,
    access_hash     BIGINT      NOT NULL,
    admin_rights    JSONB       NOT NULL DEFAULT '{}',
    title           TEXT        NOT NULL DEFAULT '',
    username        TEXT        NOT NULL DEFAULT '',
    photo           TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

-- TODO: Add indexes for stats
CREATE TABLE IF NOT EXISTS market.channel_stats (
    channel_id      BIGINT      NOT NULL,
    stats           JSONB       NOT NULL DEFAULT '{}',
    updated_at      TIMESTAMP   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (channel_id),
    FOREIGN KEY (channel_id) REFERENCES market.channel(id) ON DELETE CASCADE
);

CREATE TYPE market.role AS ENUM ('owner', 'admin');

CREATE TABLE IF NOT EXISTS market.channel_admin (
    user_id     BIGINT      NOT NULL,
    channel_id  BIGINT      NOT NULL,
    role        market.role NOT NULL,
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id),
    FOREIGN KEY (user_id) REFERENCES market.user(id),
    FOREIGN KEY (channel_id) REFERENCES market.channel(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS market.channel_admin;
DROP TYPE IF EXISTS market.role;
DROP TABLE IF EXISTS market.channel_stats;
DROP TABLE IF EXISTS market.channel;
DROP TABLE IF EXISTS market.user;
DROP SCHEMA IF EXISTS market;
