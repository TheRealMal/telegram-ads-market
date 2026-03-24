-- +goose Up

-- Restructure admin_rights from flat userbot-only JSON to nested {bot, userbot} structure.
-- Existing data becomes the "userbot" key; "bot" is set to full rights for active admin channels.
UPDATE market.channel
SET admin_rights = jsonb_build_object(
    'bot', CASE WHEN bot_member_status = 'administrator' THEN
        '{"can_post_messages":true,"can_edit_messages":true,"can_delete_messages":true,"can_post_stories":true,"can_edit_stories":true,"can_delete_stories":true,"can_promote_members":true}'::jsonb
    ELSE '{}'::jsonb END,
    'userbot', admin_rights
)
WHERE admin_rights != '{}' OR bot_member_status = 'administrator';

-- Remaining rows (empty admin_rights, non-admin bot): set default nested structure.
UPDATE market.channel
SET admin_rights = '{"bot":{},"userbot":{}}'
WHERE admin_rights->>'bot' IS NULL;

-- +goose Down

-- Revert to flat userbot-only JSON.
UPDATE market.channel
SET admin_rights = COALESCE(admin_rights->'userbot', '{}');
