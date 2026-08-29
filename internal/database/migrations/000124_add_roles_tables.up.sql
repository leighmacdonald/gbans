do
$do$
BEGIN
    CREATE TYPE permission AS ENUM (
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
        'PERMISSION_LOGIN',
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
        'PERMISSION_WIKI_READ',
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
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$do$;

create table if not exists roles (
  role_id serial primary key,
  role_name text not null,
  created_on timestamp with time zone
  not null
  default NOW(),
  updated_on timestamp with time zone
  not null
  default NOW()
);

create unique index if not exists "roles_role_name_uidx" on roles using btree (role_name);

create table if not exists role_permissions (
  role_id int
  not null
  references roles (role_id) on DELETE cascade,
  permission permission not null,
  created_on timestamp with time zone
  not null
  default NOW(),
  updated_on timestamp with time zone
  not null
  default NOW(),
  primary key (role_id, permission)
);

create table if not exists role_assignments (
  steam_id bigint
  not null
  references person (steam_id) on DELETE cascade,
  role_id int
  not null
  references roles (role_id) on DELETE cascade,
  created_on timestamp with time zone
  not null
  default NOW(),
  primary key (steam_id, role_id)
);

create index
if not exists "role_assignments_steam_id_idx"
on role_assignments
using btree
(
  steam_id
);

insert into roles (role_name) values ('admin') on conflict do nothing;
insert into roles (role_name) values ('moderator') on conflict do nothing;
insert into roles (role_name) values ('streamer') on conflict do nothing;
insert into roles (role_name) values ('banned') on conflict do nothing;
insert into roles (role_name) values ('user') on conflict do nothing;

INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'admin') r
CROSS JOIN (VALUES
    ('PERMISSION_ANTICHEAT_READ'::permission),
    ('PERMISSION_APPEAL_READ'::permission),
    ('PERMISSION_APPEAL_WRITE'::permission),
    ('PERMISSION_ASSET_CREATE'::permission),
    ('PERMISSION_ASSET_DELETE'::permission),
    ('PERMISSION_BAN_READ'::permission),
    ('PERMISSION_BAN_WRITE'::permission),
    ('PERMISSION_BAN_CREATE'::permission),
    ('PERMISSION_BLOCKLIST_READ'::permission),
    ('PERMISSION_BLOCKLIST_WRITE'::permission),
    ('PERMISSION_BLOCKLIST_DELETE'::permission),
    ('PERMISSION_CHATLOG_READ'::permission),
    ('PERMISSION_CONFIG_READ'::permission),
    ('PERMISSION_CONFIG_WRITE'::permission),
    ('PERMISSION_CONTEST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_CONTEST_DELETE'::permission),
    ('PERMISSION_CURRENT_PROFILE'::permission),
    ('PERMISSION_CURRENT_SETTINGS'::permission),
    ('PERMISSION_DEMO_READ'::permission),
    ('PERMISSION_DEMO_ADMIN'::permission),
    ('PERMISSION_DISCORD'::permission),
    ('PERMISSION_FORUM_READ'::permission),
    ('PERMISSION_FORUM_WRITE'::permission),
    ('PERMISSION_FORUM_EDIT'::permission),
    ('PERMISSION_GAMEADMIN_READ'::permission),
    ('PERMISSION_GAMEADMIN_WRITE'::permission),
    ('PERMISSION_LOGIN'::permission),
    ('PERMISSION_NETWORK_READ'::permission),
    ('PERMISSION_NETWORK_ADMIN'::permission),
    ('PERMISSION_NEWS_READ'::permission),
    ('PERMISSION_NEWS_WRITE'::permission),
    ('PERMISSION_NEWS_DELETE'::permission),
    ('PERMISSION_NOTIFICATIONS'::permission),
    ('PERMISSION_PERSON_READ'::permission),
    ('PERMISSION_PERSON_WRITE'::permission),
    ('PERMISSION_PLAYER_PROFILE'::permission),
    ('PERMISSION_REPORT_READ'::permission),
    ('PERMISSION_REPORT_WRITE'::permission),
    ('PERMISSION_REPORT_CREATE'::permission),
    ('PERMISSION_ROLE_READ'::permission),
    ('PERMISSION_ROLE_WRITE'::permission),
    ('PERMISSION_SERVER_READ'::permission),
    ('PERMISSION_SERVER_WRITE'::permission),
    ('PERMISSION_SPEEDRUN_READ'::permission),
    ('PERMISSION_SPEEDRUN_WRITE'::permission),
    ('PERMISSION_STATS_READ'::permission),
    ('PERMISSION_STEAMID_RESOLVE'::permission),
    ('PERMISSION_WIKI_READ'::permission),
    ('PERMISSION_WIKI_EDIT'::permission),
    ('PERMISSION_VOTE_READ'::permission),
    ('PERMISSION_WORDFILTER_READ'::permission),
    ('PERMISSION_WORDFILTER_WRITE'::permission),
    ('PERMISSION_WORDFILTER_DELETE'::permission)

) v(permission);

INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'moderator') r
CROSS JOIN (VALUES
    ('PERMISSION_ANTICHEAT_READ'::permission),
    ('PERMISSION_APPEAL_READ'::permission),
    ('PERMISSION_APPEAL_WRITE'::permission),
    ('PERMISSION_BAN_READ'::permission),
    ('PERMISSION_BAN_WRITE'::permission),
    ('PERMISSION_BAN_CREATE'::permission),
    ('PERMISSION_BLOCKLIST_READ'::permission),
    ('PERMISSION_BLOCKLIST_WRITE'::permission),
    ('PERMISSION_BLOCKLIST_DELETE'::permission),
    ('PERMISSION_CHATLOG_READ'::permission),
    ('PERMISSION_CONFIG_READ'::permission),
    ('PERMISSION_CONFIG_WRITE'::permission),
    ('PERMISSION_CONTEST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_CONTEST_DELETE'::permission),
    ('PERMISSION_DEMO_READ'::permission),
    ('PERMISSION_FORUM_EDIT'::permission),
    ('PERMISSION_LOGIN'::permission),
    ('PERMISSION_NETWORK_READ'::permission),
    ('PERMISSION_NEWS_READ'::permission),
    ('PERMISSION_NEWS_WRITE'::permission),
    ('PERMISSION_NEWS_DELETE'::permission),
    ('PERMISSION_PERSON_READ'::permission),
    ('PERMISSION_REPORT_READ'::permission),
    ('PERMISSION_REPORT_WRITE'::permission),
    ('PERMISSION_SPEEDRUN_WRITE'::permission),
    ('PERMISSION_WIKI_EDIT'::permission),
    ('PERMISSION_VOTE_READ'::permission),
    ('PERMISSION_WORDFILTER_READ'::permission),
    ('PERMISSION_WORDFILTER_WRITE'::permission),
    ('PERMISSION_WORDFILTER_DELETE'::permission)
) v(permission);


INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'streamer') r
CROSS JOIN (VALUES
    ('PERMISSION_LOGIN'::permission),
    ('PERMISSION_BAN_WRITE'::permission)
) v(permission);

INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'user') r
CROSS JOIN (VALUES
    ('PERMISSION_ASSET_CREATE'::permission),
    ('PERMISSION_ASSET_DELETE'::permission),
    ('PERMISSION_BAN_CREATE'::permission),
    ('PERMISSION_CHATLOG_READ'::permission),
    ('PERMISSION_CONTEST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_DEMO_READ'::permission),
    ('PERMISSION_FORUM_READ'::permission),
    ('PERMISSION_FORUM_WRITE'::permission),
    ('PERMISSION_LOGIN'::permission),
    ('PERMISSION_NEWS_READ'::permission),
    ('PERMISSION_REPORT_CREATE'::permission),
    ('PERMISSION_SPEEDRUN_READ'::permission),
    ('PERMISSION_STATS_READ'::permission),
    ('PERMISSION_WIKI_READ'::permission)
) v(permission);

-- Bootstrap the admin role for existing administrators so role-based access
-- is granted to them without requiring manual role assignment.
insert into role_assignments (steam_id, role_id, created_on)
select
  p.steam_id,
  r.role_id,
  NOW()
from
  person as p
  inner join
    roles as r
  on r.role_name = 'admin'
where
  p.permission_level = 100
on conflict do nothing;

insert into role_assignments (steam_id, role_id, created_on)
select
  p.steam_id,
  r.role_id,
  NOW()
from
  person as p
  inner join
    roles as r
  on r.role_name = 'moderator'
where
  p.permission_level = 50
on conflict do nothing;

-- Replace the legacy forum.permission_level, wiki.permission_level and
-- contest.min_permission_level integer columns with a granular required_permission
-- enum column backed by the roles.v1.Permission enum values.

alter table forum
    add column if not exists required_permission permission not null default 'PERMISSION_FORUM_READ';

alter table wiki
    add column if not exists required_permission permission not null default 'PERMISSION_WIKI_READ';

alter table contest
    add column if not exists required_permission permission not null default 'PERMISSION_CONTEST_READ';

-- Backfill any rows that previously required an elevated privilege (>=moderator)
-- with the moderator-level staff permission. Everything else retains the default
-- read permission assigned above.
update forum set required_permission = 'PERMISSION_FORUM_EDIT' where permission_level >= 50;
update wiki set required_permission = 'PERMISSION_WIKI_EDIT' where permission_level >= 50;
update contest
set required_permission = 'PERMISSION_CONTEST_ADMIN'
where
  min_permission_level >= 50;

alter table forum
  drop column if exists permission_level;

alter table wiki
  drop column if exists permission_level;

alter table contest
  drop column if exists min_permission_level;

-- Migrate legacy sourcemod users: one role per sm_groups row, granted the
-- PERMISSION_SOURCEMOD_* set implied by the group's flags, assigned to every
-- member. Flag characters follow canonical SourceMod semantics.
insert into roles (role_name)
select
  'sm-' || g.name
from
  sm_groups as g
on conflict do nothing;

insert into role_permissions (role_id, permission, created_on, updated_on)
select
  r.role_id,
  v.permission,
  NOW(),
  NOW()
from
  sm_groups as g
  cross join (values
    ('a', 'PERMISSION_SOURCEMOD_GENERIC'::permission),
    ('a', 'PERMISSION_SOURCEMOD_RESERVED'::permission),
    ('b', 'PERMISSION_SOURCEMOD_KICK'::permission),
    ('c', 'PERMISSION_SOURCEMOD_BAN'::permission),
    ('d', 'PERMISSION_SOURCEMOD_UNBAN'::permission),
    ('e', 'PERMISSION_SOURCEMOD_SLAY'::permission),
    ('f', 'PERMISSION_SOURCEMOD_CHANGEMAP'::permission),
    ('g', 'PERMISSION_SOURCEMOD_PASSWORD'::permission),
    ('h', 'PERMISSION_SOURCEMOD_CVAR'::permission),
    ('i', 'PERMISSION_SOURCEMOD_CFG'::permission),
    ('j', 'PERMISSION_SOURCEMOD_CHAT'::permission),
    ('k', 'PERMISSION_SOURCEMOD_VOTE'::permission),
    ('l', 'PERMISSION_SOURCEMOD_RCON'::permission),
    ('m', 'PERMISSION_SOURCEMOD_RCON'::permission),
    ('n', 'PERMISSION_SOURCEMOD_CHEATS'::permission),
    ('p', 'PERMISSION_SOURCEMOD_CUSTOM_1'::permission),
    ('q', 'PERMISSION_SOURCEMOD_CUSTOM_2'::permission),
    ('r', 'PERMISSION_SOURCEMOD_CUSTOM_3'::permission),
    ('s', 'PERMISSION_SOURCEMOD_CUSTOM_4'::permission),
    ('t', 'PERMISSION_SOURCEMOD_CUSTOM_5'::permission),
    ('u', 'PERMISSION_SOURCEMOD_CUSTOM_6'::permission),
    ('z', 'PERMISSION_SOURCEMOD_ROOT'::permission)
  ) v (flag, permission)
  inner join roles as r
  on r.role_name = 'sm-' || g.name
where
  v.flag = any(string_to_array(g.flags, ''))
group by
  r.role_id,
  v.permission
on conflict do nothing;

insert into role_assignments (steam_id, role_id, created_on)
select
  a.steam_id,
  r.role_id,
  NOW()
from
  sm_admins as a
  inner join sm_admins_groups as agg
  on agg.admin_id = a.id
  inner join sm_groups as g
  on g.id = agg.group_id
  inner join roles as r
  on r.role_name = 'sm-' || g.name
on conflict do nothing;

-- Group immunities and command/group overrides, previously stored in the sm_*
-- tables, re-pointed at the roles system so the sourcemod package no longer
-- touches the legacy tables.
create table if not exists role_immunity (
  role_immunity_id serial primary key,
  role_id int not null references roles (role_id) on DELETE cascade,
  other_id int not null references roles (role_id) on DELETE cascade,
  created_on timestamp with time zone not null default NOW()
);

create unique index if not exists "role_immunity_role_other_uidx"
on role_immunity
using btree
(
  role_id,
  other_id
);

create table if not exists role_overrides (
  override_id serial primary key,
  role_id int not null references roles (role_id) on DELETE cascade,
  type text not null check (type in ('command', 'group')),
  name text not null,
  access text not null check (access in ('allow', 'deny')),
  created_on timestamp with time zone not null default NOW(),
  updated_on timestamp with time zone not null default NOW()
);

create unique index if not exists "role_overrides_role_type_name_uidx"
on role_overrides
using btree
(
  role_id,
  type,
  name
);

create table if not exists command_overrides (
  override_id serial primary key,
  type text not null check (type in ('command', 'group')),
  name text not null,
  created_on timestamp with time zone not null default NOW(),
  updated_on timestamp with time zone not null default NOW()
);

create unique index if not exists "command_overrides_type_name_uidx"
on command_overrides
using btree
(
  type,
  name
);

create table if not exists command_override_permissions (
  override_id int
  not null
  references command_overrides (override_id) on DELETE cascade,
  permission permission not null,
  created_on timestamp with time zone
  not null
  default NOW(),
  updated_on timestamp with time zone
  not null
  default NOW(),
  primary key (override_id, permission)
);

insert into role_immunity (role_id, other_id, created_on)
select
  r.role_id,
  o.role_id,
  gi.created_on
from
  sm_group_immunity as gi
  inner join sm_groups as g on g.id = gi.group_id
  inner join roles as r on r.role_name = 'sm-' || g.name
  inner join sm_groups as og on og.id = gi.other_id
  inner join roles as o on o.role_name = 'sm-' || og.name
on conflict do nothing;

insert into role_overrides (role_id, type, name, access, created_on, updated_on)
select
  r.role_id,
  go.type,
  go.name,
  go.access,
  go.created_on,
  go.updated_on
from
  sm_group_overrides as go
  inner join sm_groups as g on g.id = go.group_id
  inner join roles as r on r.role_name = 'sm-' || g.name
on conflict do nothing;

insert into command_overrides (type, name, created_on, updated_on)
select
  type,
  name,
  created_on,
  updated_on
from
  sm_overrides
on conflict do nothing;

-- Translate the legacy override flag strings into permissions using the same
-- flag mapping as the sm_groups migration above.
insert into command_override_permissions (override_id, permission, created_on, updated_on)
select distinct
  co.override_id,
  m.permission,
  NOW(),
  NOW()
from
  command_overrides as co
  inner join sm_overrides as so
  on so.type = co.type and so.name = co.name
  cross join lateral regexp_split_to_table(so.flags, '') as f(ch)
  join (values
    ('a', 'PERMISSION_SOURCEMOD_RESERVED'::permission),
    ('a', 'PERMISSION_SOURCEMOD_GENERIC'::permission),
    ('b', 'PERMISSION_SOURCEMOD_KICK'::permission),
    ('c', 'PERMISSION_SOURCEMOD_BAN'::permission),
    ('d', 'PERMISSION_SOURCEMOD_UNBAN'::permission),
    ('e', 'PERMISSION_SOURCEMOD_SLAY'::permission),
    ('f', 'PERMISSION_SOURCEMOD_CHANGEMAP'::permission),
    ('g', 'PERMISSION_SOURCEMOD_PASSWORD'::permission),
    ('h', 'PERMISSION_SOURCEMOD_CVAR'::permission),
    ('i', 'PERMISSION_SOURCEMOD_CFG'::permission),
    ('j', 'PERMISSION_SOURCEMOD_CHAT'::permission),
    ('k', 'PERMISSION_SOURCEMOD_VOTE'::permission),
    ('l', 'PERMISSION_SOURCEMOD_RCON'::permission),
    ('m', 'PERMISSION_SOURCEMOD_RCON'::permission),
    ('n', 'PERMISSION_SOURCEMOD_CHEATS'::permission),
    ('p', 'PERMISSION_SOURCEMOD_CUSTOM_1'::permission),
    ('q', 'PERMISSION_SOURCEMOD_CUSTOM_2'::permission),
    ('r', 'PERMISSION_SOURCEMOD_CUSTOM_3'::permission),
    ('s', 'PERMISSION_SOURCEMOD_CUSTOM_4'::permission),
    ('t', 'PERMISSION_SOURCEMOD_CUSTOM_5'::permission),
    ('u', 'PERMISSION_SOURCEMOD_CUSTOM_6'::permission),
    ('z', 'PERMISSION_SOURCEMOD_ROOT'::permission)
  ) m(ch, permission) on m.ch = f.ch
on conflict (override_id, permission) do nothing;
