do
$do$
BEGIN
    CREATE TYPE permission AS ENUM (
        'PERMISSION_ANTICHEAT_READ',
        'PERMISSION_APPEAL_READ',
        'PERMISSION_APPEAL_WRITE',
        'PERMISSION_ASSET_WRITE',
        'PERMISSION_ASSET_DELETE',
        'PERMISSION_BAN_READ',
        'PERMISSION_BAN_WRITE',
        'PERMISSION_BAN_CREATE',
        'PERMISSION_BLOCKLIST_READ',
        'PERMISSION_BLOCKLIST_WRITE',
        'PERMISSION_BLOCKLIST_DELETE',
        'PERMISSION_CONFIG_READ',
        'PERMISSION_CONFIG_WRITE',
        'PERMISSION_CONTEST_READ',
        'PERMISSION_CONTEST_WRITE',
        'PERMISSION_CONTEST_DELETE',
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
        'PERMISSION_PERSON_READ',
        'PERMISSION_PERSON_WRITE',
        'PERMISSION_REPORT_READ',
        'PERMISSION_REPORT_WRITE',
        'PERMISSION_REPORT_CREATE',
        'PERMISSION_SERVER_READ',
        'PERMISSION_SERVER_WRITE',
        'PERMISSION_SPEEDRUN_READ',
        'PERMISSION_SPEEDRUN_WRITE',
        'PERMISSION_STATS_READ',
        'PERMISSION_WIKI_READ',
        'PERMISSION_WIKI_EDIT',
        'PERMISSION_VOTE_READ',
        'PERMISSION_WORDFILTER_READ',
        'PERMISSION_WORDFILTER_WRITE',
        'PERMISSION_WORDFILTER_DELETE',
        'PERMISSION_ROLE_READ',
        'PERMISSION_ROLE_WRITE'
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
insert into roles (role_name) values ('steamer') on conflict do nothing;
insert into roles (role_name) values ('banned') on conflict do nothing;
insert into roles (role_name) values ('user') on conflict do nothing;


INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'admin') r
CROSS JOIN (VALUES
    ('PERMISSION_ANTICHEAT_READ'::permission),
    ('PERMISSION_APPEAL_READ'::permission),
    ('PERMISSION_APPEAL_WRITE'::permission),
    ('PERMISSION_ASSET_WRITE'::permission),
    ('PERMISSION_ASSET_DELETE'::permission),
    ('PERMISSION_BAN_READ'::permission),
    ('PERMISSION_BAN_WRITE'::permission),
    ('PERMISSION_BAN_CREATE'::permission),
    ('PERMISSION_BLOCKLIST_READ'::permission),
    ('PERMISSION_BLOCKLIST_WRITE'::permission),
    ('PERMISSION_BLOCKLIST_DELETE'::permission),
    ('PERMISSION_CONFIG_READ'::permission),
    ('PERMISSION_CONFIG_WRITE'::permission),
    ('PERMISSION_CONTEST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_CONTEST_DELETE'::permission),
    ('PERMISSION_FORUM_READ'::permission),
    ('PERMISSION_FORUM_WRITE'::permission),
    ('PERMISSION_FORUM_EDIT'::permission),
    ('PERMISSION_GAMEADMIN_READ'::permission),
    ('PERMISSION_GAMEADMIN_WRITE'::permission),
    ('PERMISSION_NETWORK_READ'::permission),
    ('PERMISSION_NETWORK_ADMIN'::permission),
    ('PERMISSION_NEWS_READ'::permission),
    ('PERMISSION_NEWS_WRITE'::permission),
    ('PERMISSION_NEWS_DELETE'::permission),
    ('PERMISSION_PERSON_READ'::permission),
    ('PERMISSION_PERSON_WRITE'::permission),
    ('PERMISSION_REPORT_READ'::permission),
    ('PERMISSION_REPORT_WRITE'::permission),
    ('PERMISSION_REPORT_CREATE'::permission),
    ('PERMISSION_SERVER_READ'::permission),
    ('PERMISSION_SERVER_WRITE'::permission),
    ('PERMISSION_SPEEDRUN_READ'::permission),
    ('PERMISSION_SPEEDRUN_WRITE'::permission),
    ('PERMISSION_STATS_READ'::permission),
    ('PERMISSION_WIKI_READ'::permission),
    ('PERMISSION_WIKI_EDIT'::permission),
    ('PERMISSION_VOTE_READ'::permission),
    ('PERMISSION_WORDFILTER_READ'::permission),
    ('PERMISSION_WORDFILTER_WRITE'::permission),
    ('PERMISSION_WORDFILTER_DELETE'::permission),
    ('PERMISSION_ROLE_READ'::permission),
    ('PERMISSION_ROLE_WRITE'::permission)
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
    ('PERMISSION_BLOCKLIST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_CONTEST_DELETE'::permission),
    ('PERMISSION_FORUM_EDIT'::permission),
    ('PERMISSION_GAMEADMIN_READ'::permission),
    ('PERMISSION_GAMEADMIN_WRITE'::permission),
    ('PERMISSION_NETWORK_READ'::permission),
    ('PERMISSION_NETWORK_ADMIN'::permission),
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
    ('PERMISSION_BAN_WRITE'::permission)
) v(permission);

INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'user') r
CROSS JOIN (VALUES
    ('PERMISSION_ASSET_WRITE'::permission),
    ('PERMISSION_ASSET_DELETE'::permission),
    ('PERMISSION_BAN_CREATE'::permission),
    ('PERMISSION_CONTEST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_FORUM_READ'::permission),
    ('PERMISSION_FORUM_WRITE'::permission),
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
