-- +goose Up

ALTER TYPE market.deal_status ADD VALUE 'waiting_escrow_refund';
ALTER TYPE market.deal_status ADD VALUE 'escrow_refund_confirmed';

ALTER TYPE market.deal_post_message_status ADD VALUE 'failed';

-- +goose Down
-- PostgreSQL does not support removing enum values; leave as-is.
