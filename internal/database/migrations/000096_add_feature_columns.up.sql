BEGIN;

ALTER TABLE config ADD COLUMN general_default_route text not null default '/';
ALTER TABLE config ADD COLUMN general_news_enabled bool not null default true;
ALTER TABLE config ADD COLUMN general_forums_enabled bool not null default false;
ALTER TABLE config ADD COLUMN general_contests_enabled bool not null default false;
ALTER TABLE config ADD COLUMN general_wiki_enabled bool not null default false;
ALTER TABLE config ADD COLUMN general_stats_enabled bool not null default false;
ALTER TABLE config ADD COLUMN general_servers_enabled bool not null default true;
ALTER TABLE config ADD COLUMN general_reports_enabled bool not null default true;
ALTER TABLE config ADD COLUMN general_chatlogs_enabled bool not null default false;
ALTER TABLE config ADD COLUMN general_demos_enabled bool not null default true;

ALTER TABLE config ADD COLUMN discord_vote_log_channel_id text not null default '';
ALTER TABLE config ADD COLUMN discord_appeal_log_channel_id text not null default '';
ALTER TABLE config ADD COLUMN discord_ban_log_channel_id text not null default '';
ALTER TABLE config ADD COLUMN discord_forum_log_channel_id text not null default '';
ALTER TABLE config ADD COLUMN discord_word_filter_log_channel_id text not null default '';

ALTER TABLE asset ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE asset ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE auth_discord ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE auth_discord ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE auth_patreon ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE auth_patreon ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE cidr_block_entries ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE discord_user ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE discord_user ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE filtered_word ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE filtered_word ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE host_key ALTER COLUMN created_on TYPE timestamptz;
ALTER TABLE person ALTER COLUMN updated_on_steam TYPE timestamptz;

ALTER TABLE person_whitelist ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE person_whitelist ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE sm_admins ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE sm_admins ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE sm_admins_groups ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE sm_admins_groups ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE sm_admins_groups ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE sm_admins_groups ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE sm_group_immunity ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE sm_group_overrides ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE sm_group_overrides ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE sm_groups ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE sm_groups ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE sm_overrides ALTER COLUMN updated_on TYPE timestamptz;
ALTER TABLE sm_overrides ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE steam_friends ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE steam_group_members ALTER COLUMN created_on TYPE timestamptz;

ALTER TABLE vote_result ALTER COLUMN created_on TYPE timestamptz;

COMMIT;
