BEGIN;

ALTER TABLE config DROP COLUMN default_route;
ALTER TABLE config DROP COLUMN news_enabled;
ALTER TABLE config DROP COLUMN forums_enabled;
ALTER TABLE config DROP COLUMN contests_enabled;
ALTER TABLE config DROP COLUMN wiki_enabled;
ALTER TABLE config DROP COLUMN stats_enabled;
ALTER TABLE config DROP COLUMN servers_enabled;
ALTER TABLE config DROP COLUMN reports_enabled;
ALTER TABLE config DROP COLUMN chatlogs_enabled;
ALTER TABLE config DROP COLUMN demos_enabled;

ALTER TABLE config DROP COLUMN vote_log_channel_id;
ALTER TABLE config DROP COLUMN appeal_log_channel_id;
ALTER TABLE config DROP COLUMN ban_log_channel_id;
ALTER TABLE config DROP COLUMN forum_log_channel_id;
ALTER TABLE config DROP COLUMN word_filter_log_channel_id;

ALTER TABLE asset ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE asset ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE auth_discord ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE auth_discord ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE auth_patreon ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE auth_patreon ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE cidr_block_entries ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE discord_user ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE discord_user ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE filtered_word ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE filtered_word ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE host_key ALTER COLUMN created_on TYPE timestamp;
ALTER TABLE person ALTER COLUMN updated_on_steam TYPE timestamp;

ALTER TABLE person_whitelist ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE person_whitelist ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE sm_admins ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE sm_admins ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE sm_admins_groups ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE sm_admins_groups ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE sm_admins_groups ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE sm_admins_groups ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE sm_group_immunity ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE sm_group_overrides ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE sm_group_overrides ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE sm_groups ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE sm_groups ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE sm_overrides ALTER COLUMN updated_on TYPE timestamp;
ALTER TABLE sm_overrides ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE steam_friends ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE steam_group_members ALTER COLUMN created_on TYPE timestamp;

ALTER TABLE vote_result ALTER COLUMN created_on TYPE timestamp;

COMMIT;
