-- +goose Up
-- Add prepared_post JSONB nullable column to listing table (extensible for future fields)
-- Initial structure: {"message": "post text here"}
ALTER TABLE market.listing ADD COLUMN prepared_post JSONB;

-- Migrate existing prices from [["24hr", price]] to [["post", "24hr", price]]
UPDATE market.listing
SET prices = (
    SELECT jsonb_agg(jsonb_build_array('post', elem->0, elem->1))
    FROM jsonb_array_elements(prices) AS elem
)
WHERE prices IS NOT NULL AND prices != '{}'::jsonb AND prices != '[]'::jsonb;

-- Migrate existing deal type from "24hr" to "post"
UPDATE market.deal SET type = 'post' WHERE type ~ '^\d+hr$';

-- +goose Down
ALTER TABLE market.listing DROP COLUMN IF EXISTS prepared_post;
