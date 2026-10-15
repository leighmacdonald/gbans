begin;

-- Part 1: config NULL backfill.
-- Columns added to config without NOT NULL accepted NULL rows (via manual
-- edits, restores, or older writes), but Config.Read scans them into plain
-- string/bool fields, so any NULL aborts startup and every CLI command with:
-- "can't scan into dest (col: general_sentry_dsn): cannot scan NULL into *string".
-- Backfill NULLs and enforce NOT NULL going forward.
update config set general_site_description = '' where general_site_description is null;
update config set general_favicon = '' where general_favicon is null;
update config set general_sentry_dsn = '' where general_sentry_dsn is null;
update config set general_sentry_dsn_web = '' where general_sentry_dsn_web is null;
update config set general_banner_image = '' where general_banner_image is null;
update config set demo_parser_url = 'http://localhost:8811/' where demo_parser_url is null;
update config set general_mge_enabled = false where general_mge_enabled is null;

alter table if exists config alter column general_site_description set default '';
alter table if exists config alter column general_site_description set not null;
alter table if exists config alter column general_favicon set default '';
alter table if exists config alter column general_favicon set not null;
alter table if exists config alter column general_sentry_dsn set default '';
alter table if exists config alter column general_sentry_dsn set not null;
alter table if exists config alter column general_sentry_dsn_web set default '';
alter table if exists config alter column general_sentry_dsn_web set not null;
alter table if exists config alter column general_banner_image set default '';
alter table if exists config alter column general_banner_image set not null;
alter table if exists config alter column demo_parser_url set default 'http://localhost:8811/';
alter table if exists config alter column demo_parser_url set not null;
alter table if exists config alter column general_mge_enabled set default false;
alter table if exists config alter column general_mge_enabled set not null;

-- Part 2: remove PERMISSION_WIKI_READ, wiki reading is public and no longer
-- gated. Pages previously requiring it become public
-- (PERMISSION_UNSPECIFIED), matching the default for newly created pages.
delete from role_permissions where permission = 'PERMISSION_WIKI_READ';
delete from command_override_permissions where permission = 'PERMISSION_WIKI_READ';

update wiki set required_permission = 'PERMISSION_UNSPECIFIED' where required_permission = 'PERMISSION_WIKI_READ';

-- Defaults must be dropped before the type swap: postgres cannot coerce a
-- default expression of the old enum type to the rebuilt type.
alter table forum alter column required_permission drop default;
alter table wiki alter column required_permission drop default;
alter table contest alter column required_permission drop default;

-- Postgres cannot drop an enum label, so rebuild the type without the label.
alter type permission rename to permission_old;

create type permission as enum (
    'PERMISSION_UNSPECIFIED',
    'PERMISSION_ANTICHEAT_READ',
    'PERMISSION_APPEAL_READ',
    'PERMISSION_APPEAL_WRITE',
    'PERMISSION_APPEAL_ADMIN',
    'PERMISSION_ASSET_CREATE',
    'PERMISSION_ASSET_DELETE',
    'PERMISSION_BAN_READ',
    'PERMISSION_BAN_WRITE',
    'PERMISSION_BAN_CREATE',
    'PERMISSION_BLOCKLIST_READ',
    'PERMISSION_BLOCKLIST_WRITE',
    'PERMISSION_BLOCKLIST_DELETE',
    'PERMISSION_CHATLOG_READ',
    'PERMISSION_CONFIG_READ',
    'PERMISSION_CONFIG_WRITE',
    'PERMISSION_CONTEST_READ',
    'PERMISSION_CONTEST_WRITE',
    'PERMISSION_CONTEST_DELETE',
    'PERMISSION_CONTEST_ADMIN',
    'PERMISSION_CURRENT_PROFILE',
    'PERMISSION_CURRENT_SETTINGS',
    'PERMISSION_DEMO_READ',
    'PERMISSION_DEMO_ADMIN',
    'PERMISSION_DISCORD',
    'PERMISSION_FORUM_READ',
    'PERMISSION_FORUM_WRITE',
    'PERMISSION_FORUM_EDIT',
    'PERMISSION_GAMEADMIN_READ',
    'PERMISSION_GAMEADMIN_WRITE',
    'PERMISSION_NETWORK_READ',
    'PERMISSION_NETWORK_ADMIN',
    'PERMISSION_NEWS_READ',
    'PERMISSION_NEWS_WRITE',
    'PERMISSION_NEWS_DELETE',
    'PERMISSION_NOTIFICATIONS',
    'PERMISSION_PERSON_READ',
    'PERMISSION_PERSON_WRITE',
    'PERMISSION_PLAYER_PROFILE',
    'PERMISSION_REPORT_READ',
    'PERMISSION_REPORT_WRITE',
    'PERMISSION_REPORT_CREATE',
    'PERMISSION_REPORT_ADMIN',
    'PERMISSION_ROLE_READ',
    'PERMISSION_ROLE_WRITE',
    'PERMISSION_SERVER_READ',
    'PERMISSION_SERVER_WRITE',
    'PERMISSION_SPEEDRUN_READ',
    'PERMISSION_SPEEDRUN_WRITE',
    'PERMISSION_STATS_READ',
    'PERMISSION_STEAMID_RESOLVE',
    'PERMISSION_WIKI_EDIT',
    'PERMISSION_VOTE_READ',
    'PERMISSION_WORDFILTER_READ',
    'PERMISSION_WORDFILTER_WRITE',
    'PERMISSION_WORDFILTER_DELETE',
    'PERMISSION_SOURCEMOD_RESERVED',
    'PERMISSION_SOURCEMOD_GENERIC',
    'PERMISSION_SOURCEMOD_KICK',
    'PERMISSION_SOURCEMOD_BAN',
    'PERMISSION_SOURCEMOD_UNBAN',
    'PERMISSION_SOURCEMOD_SLAY',
    'PERMISSION_SOURCEMOD_CHANGEMAP',
    'PERMISSION_SOURCEMOD_CVAR',
    'PERMISSION_SOURCEMOD_CFG',
    'PERMISSION_SOURCEMOD_CHAT',
    'PERMISSION_SOURCEMOD_VOTE',
    'PERMISSION_SOURCEMOD_PASSWORD',
    'PERMISSION_SOURCEMOD_RCON',
    'PERMISSION_SOURCEMOD_CHEATS',
    'PERMISSION_SOURCEMOD_ROOT',
    'PERMISSION_SOURCEMOD_CUSTOM_1',
    'PERMISSION_SOURCEMOD_CUSTOM_2',
    'PERMISSION_SOURCEMOD_CUSTOM_3',
    'PERMISSION_SOURCEMOD_CUSTOM_4',
    'PERMISSION_SOURCEMOD_CUSTOM_5',
    'PERMISSION_SOURCEMOD_CUSTOM_6'
);

alter table role_permissions alter column permission type permission using permission::text::permission;
alter table command_override_permissions alter column permission type permission using permission::text::permission;
alter table forum alter column required_permission type permission using required_permission::text::permission;
alter table wiki alter column required_permission type permission using required_permission::text::permission;
alter table contest alter column required_permission type permission using required_permission::text::permission;

drop type permission_old;

alter table forum alter column required_permission set default 'PERMISSION_FORUM_READ';
alter table wiki alter column required_permission set default 'PERMISSION_UNSPECIFIED';
alter table contest alter column required_permission set default 'PERMISSION_CONTEST_READ';

commit;
