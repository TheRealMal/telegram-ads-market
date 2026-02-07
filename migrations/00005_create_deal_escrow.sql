-- +goose Up

CREATE SCHEMA IF NOT EXISTS payments;

CREATE TABLE IF NOT EXISTS payments.transactions_incoming (
    id                  BIGSERIAL           NOT NULL,
    address             TEXT                NOT NULL,
    currency            TEXT                NOT NULL,
    amount              BIGINT              NOT NULL,
    tx_hash             TEXT                NOT NULL,
    created_at          TIMESTAMP           NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS payments.transactions_outgoing (
    id                  BIGSERIAL           NOT NULL,
    source_address      TEXT                NOT NULL,
    destination_address TEXT                NOT NULL,
    currency            TEXT                NOT NULL,
    amount              BIGINT              NOT NULL,
    comment             TEXT                NULL,
    tx_hash             TEXT                NOT NULL,
    created_at          TIMESTAMP           NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE IF EXISTS payments.transactions_incoming;
DROP TABLE IF EXISTS payments.transactions_outgoing;
DROP SCHEMA IF EXISTS payments;
