do
$do$
BEGIN
    CREATE TYPE permission AS ENUM (
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
        'PERMISSION_WORDFILTER_DELETE'
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
