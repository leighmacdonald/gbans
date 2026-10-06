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

insert into role_permissions (role_id, permission, created_on, updated_on)
select
  r.role_id,
  v.permission,
  NOW(),
  NOW()
from
  (
    select
      role_id
    from
      roles
    where
      role_name = 'admin'
  )
  as r
  inner join
    (
      values
        (cast('PERMISSION_ANTICHEAT_READ' as permission)),
        (cast('PERMISSION_APPEAL_READ' as permission)),
        (cast('PERMISSION_APPEAL_WRITE' as permission)),
        (cast('PERMISSION_ASSET_CREATE' as permission)),
        (cast('PERMISSION_ASSET_DELETE' as permission)),
        (cast('PERMISSION_BAN_READ' as permission)),
        (cast('PERMISSION_BAN_WRITE' as permission)),
        (cast('PERMISSION_BAN_CREATE' as permission)),
        (cast('PERMISSION_BLOCKLIST_READ' as permission)),
        (cast('PERMISSION_BLOCKLIST_WRITE' as permission)),
        (
          cast('PERMISSION_BLOCKLIST_DELETE' as permission)
        ),
        (cast('PERMISSION_CHATLOG_READ' as permission)),
        (cast('PERMISSION_CONFIG_READ' as permission)),
        (cast('PERMISSION_CONFIG_WRITE' as permission)),
        (cast('PERMISSION_CONTEST_READ' as permission)),
        (cast('PERMISSION_CONTEST_WRITE' as permission)),
        (cast('PERMISSION_CONTEST_DELETE' as permission)),
        (cast('PERMISSION_CURRENT_PROFILE' as permission)),
        (
          cast('PERMISSION_CURRENT_SETTINGS' as permission)
        ),
        (cast('PERMISSION_DEMO_READ' as permission)),
        (cast('PERMISSION_DEMO_ADMIN' as permission)),
        (cast('PERMISSION_DISCORD' as permission)),
        (cast('PERMISSION_FORUM_READ' as permission)),
        (cast('PERMISSION_FORUM_WRITE' as permission)),
        (cast('PERMISSION_FORUM_EDIT' as permission)),
        (cast('PERMISSION_GAMEADMIN_READ' as permission)),
        (cast('PERMISSION_GAMEADMIN_WRITE' as permission)),
        (cast('PERMISSION_NETWORK_READ' as permission)),
        (cast('PERMISSION_NETWORK_ADMIN' as permission)),
        (cast('PERMISSION_NEWS_READ' as permission)),
        (cast('PERMISSION_NEWS_WRITE' as permission)),
        (cast('PERMISSION_NEWS_DELETE' as permission)),
        (cast('PERMISSION_NOTIFICATIONS' as permission)),
        (cast('PERMISSION_PERSON_READ' as permission)),
        (cast('PERMISSION_PERSON_WRITE' as permission)),
        (cast('PERMISSION_PLAYER_PROFILE' as permission)),
        (cast('PERMISSION_REPORT_READ' as permission)),
        (cast('PERMISSION_REPORT_WRITE' as permission)),
        (cast('PERMISSION_REPORT_CREATE' as permission)),
        (cast('PERMISSION_REPORT_ADMIN' as permission)),
        (cast('PERMISSION_ROLE_READ' as permission)),
        (cast('PERMISSION_ROLE_WRITE' as permission)),
        (cast('PERMISSION_SERVER_READ' as permission)),
        (cast('PERMISSION_SERVER_WRITE' as permission)),
        (cast('PERMISSION_SPEEDRUN_READ' as permission)),
        (cast('PERMISSION_SPEEDRUN_WRITE' as permission)),
        (cast('PERMISSION_STATS_READ' as permission)),
        (cast('PERMISSION_STEAMID_RESOLVE' as permission)),
        (cast('PERMISSION_WIKI_READ' as permission)),
        (cast('PERMISSION_WIKI_EDIT' as permission)),
        (cast('PERMISSION_VOTE_READ' as permission)),
        (cast('PERMISSION_WORDFILTER_READ' as permission)),
        (
          cast('PERMISSION_WORDFILTER_WRITE' as permission)
        ),
        (
          cast('PERMISSION_WORDFILTER_DELETE' as permission)
        )
    )
    as v (permission)
  on true
on conflict do nothing;

insert into role_permissions (role_id, permission, created_on, updated_on)
select
  r.role_id,
  v.permission,
  NOW(),
  NOW()
from
  (
    select
      role_id
    from
      roles
    where
      role_name = 'moderator'
  )
  as r
  inner join
    (
      values
        (cast('PERMISSION_ANTICHEAT_READ' as permission)),
        (cast('PERMISSION_APPEAL_READ' as permission)),
        (cast('PERMISSION_APPEAL_WRITE' as permission)),
        (cast('PERMISSION_BAN_READ' as permission)),
        (cast('PERMISSION_BAN_WRITE' as permission)),
        (cast('PERMISSION_BAN_CREATE' as permission)),
        (cast('PERMISSION_BLOCKLIST_READ' as permission)),
        (cast('PERMISSION_BLOCKLIST_WRITE' as permission)),
        (
          cast('PERMISSION_BLOCKLIST_DELETE' as permission)
        ),
        (cast('PERMISSION_CHATLOG_READ' as permission)),
        (cast('PERMISSION_CONTEST_READ' as permission)),
        (cast('PERMISSION_CONTEST_WRITE' as permission)),
        (cast('PERMISSION_CONTEST_DELETE' as permission)),
        (cast('PERMISSION_DEMO_READ' as permission)),
        (cast('PERMISSION_FORUM_EDIT' as permission)),
        (cast('PERMISSION_NETWORK_READ' as permission)),
        (cast('PERMISSION_NEWS_READ' as permission)),
        (cast('PERMISSION_NEWS_WRITE' as permission)),
        (cast('PERMISSION_NEWS_DELETE' as permission)),
        (cast('PERMISSION_NOTIFICATIONS' as permission)),
        (cast('PERMISSION_PERSON_READ' as permission)),
        (cast('PERMISSION_REPORT_READ' as permission)),
        (cast('PERMISSION_REPORT_WRITE' as permission)),
        (cast('PERMISSION_REPORT_ADMIN' as permission)),
        (cast('PERMISSION_SPEEDRUN_WRITE' as permission)),
        (cast('PERMISSION_STATS_READ' as permission)),
        (cast('PERMISSION_WIKI_EDIT' as permission)),
        (cast('PERMISSION_VOTE_READ' as permission)),
        (cast('PERMISSION_WORDFILTER_READ' as permission)),
        (
          cast('PERMISSION_WORDFILTER_WRITE' as permission)
        ),
        (
          cast('PERMISSION_WORDFILTER_DELETE' as permission)
        )
    )
    as v (permission)
  on true
on conflict do nothing;


insert into role_permissions (role_id, permission, created_on, updated_on)
select
  r.role_id,
  v.permission,
  NOW(),
  NOW()
from
  (
    select
      role_id
    from
      roles
    where
      role_name = 'streamer'
  )
  as r
  inner join
    (
      values
        (cast('PERMISSION_BAN_WRITE' as permission))
    )
    as v (permission)
  on true
on conflict do nothing;

insert into role_permissions (role_id, permission, created_on, updated_on)
select
  r.role_id,
  v.permission,
  NOW(),
  NOW()
from
  (
    select role_id from roles where role_name = 'user'
  )
  as r
  inner join
    (
      values
        (cast('PERMISSION_ASSET_CREATE' as permission)),
        (cast('PERMISSION_ASSET_DELETE' as permission)),
        (cast('PERMISSION_BAN_CREATE' as permission)),
        (cast('PERMISSION_CHATLOG_READ' as permission)),
        (cast('PERMISSION_CONTEST_READ' as permission)),
        (cast('PERMISSION_CONTEST_WRITE' as permission)),
        (cast('PERMISSION_DEMO_READ' as permission)),
        (cast('PERMISSION_FORUM_READ' as permission)),
        (cast('PERMISSION_FORUM_WRITE' as permission)),
        (cast('PERMISSION_NEWS_READ' as permission)),
        (cast('PERMISSION_REPORT_CREATE' as permission)),
        (cast('PERMISSION_REPORT_READ' as permission)),
        (cast('PERMISSION_WIKI_READ' as permission)),
        (cast('PERMISSION_NOTIFICATIONS' as permission))
    )
    as v (permission)
  on true
on conflict do nothing;

-- Bootstrap the admin role for existing administrators so role-based access
-- is granted to them without requiring manual role assignment. Guarded so the
-- migration can re-run after person.permission_level has already been dropped.
do
$do$
begin
  if exists (
    select
      1
    from
      information_schema.columns
    where
      table_schema = 'public'
      and table_name = 'person'
      and column_name = 'permission_level'
  ) then
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
  end if;
end
$do$;

-- Replace the legacy forum.permission_level, wiki.permission_level and
-- contest.min_permission_level integer columns with a granular required_permission
-- enum column backed by the roles.v1.Permission enum values.

alter table forum
  add column if not exists required_permission permission
    not null
    default 'PERMISSION_FORUM_READ';

alter table wiki
  add column if not exists required_permission permission
    not null
    default 'PERMISSION_WIKI_READ';

alter table contest
  add column if not exists required_permission permission
    not null
    default 'PERMISSION_CONTEST_READ';

-- Backfill any rows that previously required an elevated privilege (>=moderator)
-- with the moderator-level staff permission. Everything else retains the default
-- read permission assigned above. Guarded so the migration can re-run after the
-- legacy integer columns have already been dropped.
do
$do$
begin
  if exists (
    select
      1
    from
      information_schema.columns
    where
      table_schema = 'public'
      and table_name = 'forum'
      and column_name = 'permission_level'
  ) then
    update forum set required_permission = 'PERMISSION_FORUM_EDIT' where permission_level >= 50;
  end if;

  if exists (
    select
      1
    from
      information_schema.columns
    where
      table_schema = 'public'
      and table_name = 'wiki'
      and column_name = 'permission_level'
  ) then
    update wiki set required_permission = 'PERMISSION_WIKI_EDIT' where permission_level >= 50;
  end if;

  if exists (
    select
      1
    from
      information_schema.columns
    where
      table_schema = 'public'
      and table_name = 'contest'
      and column_name = 'min_permission_level'
  ) then
    update contest
    set required_permission = 'PERMISSION_CONTEST_ADMIN'
    where
      min_permission_level >= 50;
  end if;
end
$do$;

alter table forum
  drop column if exists permission_level;

alter table wiki
  drop column if exists permission_level;

alter table contest
  drop column if exists min_permission_level;

-- The legacy person.permission_level integer column is fully replaced by the
-- role-based permission system; drop it along with its index.
drop index if exists idx_person_permission;

alter table person
  drop column if exists permission_level;

-- Migrate legacy sourcemod users: one role per sm_groups row, granted the
-- PERMISSION_SOURCEMOD_* set implied by the group's flags, assigned to every
-- member. Flag characters follow canonical SourceMod semantics.
insert into roles (role_name) select 'sm-' || g.name from sm_groups as g on conflict do nothing;

insert into role_permissions (role_id, permission, created_on, updated_on)
select
  r.role_id,
  v.permission,
  NOW(),
  NOW()
from
  sm_groups as g
  inner join
    (
      values
        (
          'a',
          cast('PERMISSION_SOURCEMOD_GENERIC' as permission)
        ),
        (
          'a',
          cast('PERMISSION_SOURCEMOD_RESERVED'
          as permission)
        ),
        (
          'b',
          cast('PERMISSION_SOURCEMOD_KICK' as permission)
        ),
        (
          'c',
          cast('PERMISSION_SOURCEMOD_BAN' as permission)
        ),
        (
          'd',
          cast('PERMISSION_SOURCEMOD_UNBAN' as permission)
        ),
        (
          'e',
          cast('PERMISSION_SOURCEMOD_SLAY' as permission)
        ),
        (
          'f',
          cast('PERMISSION_SOURCEMOD_CHANGEMAP'
          as permission)
        ),
        (
          'g',
          cast('PERMISSION_SOURCEMOD_PASSWORD'
          as permission)
        ),
        (
          'h',
          cast('PERMISSION_SOURCEMOD_CVAR' as permission)
        ),
        (
          'i',
          cast('PERMISSION_SOURCEMOD_CFG' as permission)
        ),
        (
          'j',
          cast('PERMISSION_SOURCEMOD_CHAT' as permission)
        ),
        (
          'k',
          cast('PERMISSION_SOURCEMOD_VOTE' as permission)
        ),
        (
          'l',
          cast('PERMISSION_SOURCEMOD_RCON' as permission)
        ),
        (
          'm',
          cast('PERMISSION_SOURCEMOD_RCON' as permission)
        ),
        (
          'n',
          cast('PERMISSION_SOURCEMOD_CHEATS' as permission)
        ),
        (
          'p',
          cast('PERMISSION_SOURCEMOD_CUSTOM_1'
          as permission)
        ),
        (
          'q',
          cast('PERMISSION_SOURCEMOD_CUSTOM_2'
          as permission)
        ),
        (
          'r',
          cast('PERMISSION_SOURCEMOD_CUSTOM_3'
          as permission)
        ),
        (
          's',
          cast('PERMISSION_SOURCEMOD_CUSTOM_4'
          as permission)
        ),
        (
          't',
          cast('PERMISSION_SOURCEMOD_CUSTOM_5'
          as permission)
        ),
        (
          'u',
          cast('PERMISSION_SOURCEMOD_CUSTOM_6'
          as permission)
        ),
        (
          'z',
          cast('PERMISSION_SOURCEMOD_ROOT' as permission)
        )
    )
    as v (flag, permission)
  on true
  inner join
    roles as r
  on r.role_name = 'sm-' || g.name
where
  v.flag = any (regexp_split_to_array(g.flags, ''))
group by r.role_id,
  v.permission
on conflict do nothing;

insert into role_assignments (steam_id, role_id, created_on)
select
  a.steam_id,
  r.role_id,
  NOW()
from
  sm_admins as a
  inner join
    sm_admins_groups as agg
  on agg.admin_id = a.id
  inner join
    sm_groups as g
  on g.id = agg.group_id
  inner join
    roles as r
  on r.role_name = 'sm-' || g.name
on conflict do nothing;

-- The personal role carries an admin's direct permissions and display name
-- (stored on the sm_admins row in the legacy model). It is tracked explicitly
-- so a group that happens to have a single member is never mistaken for it.

create table if not exists sm_personal_roles (
  steam_id bigint
  not null
  references person (steam_id) on DELETE cascade,
  role_id int
  not null
  references roles (role_id) on DELETE cascade,
  created_on timestamp with time zone
  not null
  default NOW(),
  updated_on timestamp with time zone
  not null
  default NOW(),
  primary key (steam_id)
);

create index
if not exists "sm_personal_roles_role_id_idx"
on sm_personal_roles
using btree
(
  role_id
);

-- Migrate the flags granted directly to each legacy admin into a personal
-- role, named after the admin and falling back to the steam id when the name
-- is empty or already taken by a group with members.

do
$do$
declare
  rec record;
  v_role_name text;
  v_role_id int;
  member_count int;
begin
  for rec in
    select
      distinct on (steam_id)
      steam_id,
      name,
      flags,
      created_on,
      updated_on
    from
      sm_admins
    where
      steam_id is not null
    order by
      steam_id,
      created_on,
      id
  loop
    -- Skip admins already migrated by a previous (partial) run.
    if exists (select 1 from sm_personal_roles as spr where spr.steam_id = rec.steam_id) then
      continue;
    end if;

    if coalesce(trim(rec.name), '') <> '' then
      v_role_name := 'sm-' || rec.name;
    else
      v_role_name := 'sm-' || rec.steam_id;
    end if;

    select
      r.role_id
    into
      v_role_id
    from
      roles as r
    where
      r.role_name = v_role_name;

    if not found then
      insert into roles (role_name, created_on, updated_on)
      values (v_role_name, rec.created_on, rec.updated_on)
      returning role_id into v_role_id;
    else
      select
        count(*)
      into
        member_count
      from
        role_assignments as ra
      where
        ra.role_id = v_role_id;

      if member_count > 0 then
        -- The alias is taken by a group; fall back to the steam id.
        v_role_name := 'sm-' || rec.steam_id;

        select
          r.role_id
        into
          v_role_id
        from
          roles as r
        where
          r.role_name = v_role_name;

        if found then
          select
            count(*)
          into
            member_count
          from
            role_assignments as ra
          where
            ra.role_id = v_role_id;
        end if;

        -- Both names are taken by groups with members; leave the admin with
        -- their group roles only.
        if found and member_count > 0 then
          continue;
        end if;

        if not found then
          insert into roles (role_name, created_on, updated_on)
          values (v_role_name, rec.created_on, rec.updated_on)
          returning role_id into v_role_id;
        end if;
      end if;
    end if;

    insert into role_permissions (role_id, permission, created_on, updated_on)
    select
      v_role_id,
      v.permission,
      rec.created_on,
      rec.updated_on
    from
      (values
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
      ) v (flag, permission)
    where
      v.flag = any(regexp_split_to_array(rec.flags, ''))
    on conflict do nothing;

    insert into role_assignments (steam_id, role_id, created_on)
    values (rec.steam_id, v_role_id, rec.created_on)
    on conflict do nothing;

    insert into sm_personal_roles (steam_id, role_id, created_on, updated_on)
    values (rec.steam_id, v_role_id, rec.created_on, rec.updated_on)
    on conflict (steam_id) do nothing;
  end loop;
end
$do$;

-- Group immunities and command/group overrides, previously stored in the sm_*
-- tables, re-pointed at the roles system so the sourcemod package no longer
-- touches the legacy tables.
create table if not exists role_immunity (
  role_immunity_id serial primary key,
  role_id int
  not null
  references roles (role_id) on DELETE cascade,
  other_id int
  not null
  references roles (role_id) on DELETE cascade,
  created_on timestamp with time zone
  not null
  default NOW()
);

create unique
index
if not exists "role_immunity_role_other_uidx"
on role_immunity
using btree
(
  role_id,
  other_id
);

create table if not exists role_overrides (
  override_id serial primary key,
  role_id int
  not null
  references roles (role_id) on DELETE cascade,
  type text
  not null
  check (type in ('command', 'group')),
  name text not null,
  access text
  not null
  check (access in ('allow', 'deny')),
  created_on timestamp with time zone
  not null
  default NOW(),
  updated_on timestamp with time zone
  not null
  default NOW()
);

create unique
index
if not exists "role_overrides_role_type_name_uidx"
on role_overrides
using btree
(
  role_id,
  type,
  name
);

create table if not exists command_overrides (
  override_id serial primary key,
  type text
  not null
  check (type in ('command', 'group')),
  name text not null,
  created_on timestamp with time zone
  not null
  default NOW(),
  updated_on timestamp with time zone
  not null
  default NOW()
);

create unique
index
if not exists "command_overrides_type_name_uidx"
on command_overrides
using btree
(
  type,
  name
);

create table if not exists command_override_permissions (
  override_id int
  not null
  references command_overrides (override_id)
  on DELETE cascade,
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
  inner join
    sm_groups as g
  on g.id = gi.group_id
  inner join
    roles as r
  on r.role_name = 'sm-' || g.name
  inner join
    sm_groups as og
  on og.id = gi.other_id
  inner join
    roles as o
  on o.role_name = 'sm-' || og.name
on conflict do nothing;

insert into role_overrides (
  role_id,
  type,
  name,
  access,
  created_on,
  updated_on
)
select
  r.role_id,
  go.type,
  go.name,
  go.access,
  go.created_on,
  go.updated_on
from
  sm_group_overrides as go
  inner join
    sm_groups as g
  on g.id = go.group_id
  inner join
    roles as r
  on r.role_name = 'sm-' || g.name
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
  inner join
    sm_overrides as so
  on so.type = co.type and so.name = co.name
  inner join
    lateral regexp_split_to_table(so.flags, '')
    as f (ch)
  on true
  inner join
    (
      values
        (
          'a',
          cast('PERMISSION_SOURCEMOD_RESERVED'
          as permission)
        ),
        (
          'a',
          cast('PERMISSION_SOURCEMOD_GENERIC' as permission)
        ),
        (
          'b',
          cast('PERMISSION_SOURCEMOD_KICK' as permission)
        ),
        (
          'c',
          cast('PERMISSION_SOURCEMOD_BAN' as permission)
        ),
        (
          'd',
          cast('PERMISSION_SOURCEMOD_UNBAN' as permission)
        ),
        (
          'e',
          cast('PERMISSION_SOURCEMOD_SLAY' as permission)
        ),
        (
          'f',
          cast('PERMISSION_SOURCEMOD_CHANGEMAP'
          as permission)
        ),
        (
          'g',
          cast('PERMISSION_SOURCEMOD_PASSWORD'
          as permission)
        ),
        (
          'h',
          cast('PERMISSION_SOURCEMOD_CVAR' as permission)
        ),
        (
          'i',
          cast('PERMISSION_SOURCEMOD_CFG' as permission)
        ),
        (
          'j',
          cast('PERMISSION_SOURCEMOD_CHAT' as permission)
        ),
        (
          'k',
          cast('PERMISSION_SOURCEMOD_VOTE' as permission)
        ),
        (
          'l',
          cast('PERMISSION_SOURCEMOD_RCON' as permission)
        ),
        (
          'm',
          cast('PERMISSION_SOURCEMOD_RCON' as permission)
        ),
        (
          'n',
          cast('PERMISSION_SOURCEMOD_CHEATS' as permission)
        ),
        (
          'p',
          cast('PERMISSION_SOURCEMOD_CUSTOM_1'
          as permission)
        ),
        (
          'q',
          cast('PERMISSION_SOURCEMOD_CUSTOM_2'
          as permission)
        ),
        (
          'r',
          cast('PERMISSION_SOURCEMOD_CUSTOM_3'
          as permission)
        ),
        (
          's',
          cast('PERMISSION_SOURCEMOD_CUSTOM_4'
          as permission)
        ),
        (
          't',
          cast('PERMISSION_SOURCEMOD_CUSTOM_5'
          as permission)
        ),
        (
          'u',
          cast('PERMISSION_SOURCEMOD_CUSTOM_6'
          as permission)
        ),
        (
          'z',
          cast('PERMISSION_SOURCEMOD_ROOT' as permission)
        )
    )
    as m (ch, permission)
  on m.ch = f.ch
on conflict (override_id, permission) do nothing;

-- Ensure no duplicate assets with the same content hash are created.
-- Partial index: only non-deleted assets must have unique hashes; deleted
-- assets can coexist so that Restore() can re-activate them.
create unique
index
if not exists "asset_hash_unique"
on asset
using btree
(
  hash
)
where
  not deleted;
