-- +goose Up

CREATE TYPE market.user_role AS ENUM (
    'user',
    'admin'
);

ALTER TABLE market.user
ADD COLUMN IF NOT EXISTS role market.user_role NOT NULL DEFAULT 'user';

-- +goose Down

ALTER TABLE market.user DROP COLUMN IF EXISTS role;
DROP TYPE IF EXISTS market.user_role;
