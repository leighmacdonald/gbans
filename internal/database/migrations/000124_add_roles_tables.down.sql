alter table contest
    add column if not exists min_permission_level integer not null default 10;

alter table wiki
    add column if not exists permission_level integer not null default 1;

alter table forum
    add column if not exists permission_level integer not null default 1;

update forum set permission_level = 50 where required_permission = 'PERMISSION_FORUM_EDIT';

update wiki set permission_level = 50 where required_permission = 'PERMISSION_WIKI_EDIT';

update contest set min_permission_level = 50 where required_permission = 'PERMISSION_CONTEST_ADMIN';

alter table forum
  drop column if exists required_permission;

alter table wiki
  drop column if exists required_permission;

alter table contest
  drop column if exists required_permission;


drop table if exists role_assignments;

drop table if exists role_permissions;

drop table if exists roles;

drop type if exists permission;
