-- +goose Up
-- Fix any channel IDs stored in bare MTProto format (positive) to Bot API format (negative -100 prefix).
-- See https://core.telegram.org/api/bots/ids
DO $$
DECLARE
    r RECORD;
    new_id BIGINT;
BEGIN
    FOR r IN SELECT id FROM market.channel WHERE id > 0 LOOP
        new_id := -1000000000000 - r.id;
        IF NOT EXISTS (SELECT 1 FROM market.channel WHERE id = new_id) THEN
            UPDATE market.channel_stats SET channel_id = new_id WHERE channel_id = r.id;
            UPDATE market.channel_admin SET channel_id = new_id WHERE channel_id = r.id;
            UPDATE market.listing SET channel_id = new_id WHERE channel_id = r.id;
            UPDATE market.deal SET channel_id = new_id WHERE channel_id = r.id;
            UPDATE market.deal_post_message SET channel_id = new_id WHERE channel_id = r.id;
            UPDATE market.channel SET id = new_id WHERE id = r.id;
        ELSE
            -- Bot API version already exists: merge by deleting the bare duplicate
            DELETE FROM market.channel_stats WHERE channel_id = r.id;
            DELETE FROM market.channel_admin WHERE channel_id = r.id;
            UPDATE market.listing SET channel_id = new_id WHERE channel_id = r.id;
            UPDATE market.deal SET channel_id = new_id WHERE channel_id = r.id;
            UPDATE market.deal_post_message SET channel_id = new_id WHERE channel_id = r.id;
            DELETE FROM market.channel WHERE id = r.id;
        END IF;
    END LOOP;
END $$;

-- +goose Down
-- Data fix, no reverse migration
