begin;

drop index if exists person_messages_steam_id_uindex;

alter table if exists person add column if not exists muted bool default false;

alter table if exists ban add column if not exists appeal_state int default 0;
alter table if exists ban_group add column if not exists appeal_state int default 0;
alter table if exists ban_net add column if not exists appeal_state int default 0;
alter table if exists ban_asn add column if not exists appeal_state int default 0;

drop index if exists wiki_media_name_uindex;

alter table person alter column permission_level type int using permission_level::int;
alter table person alter column permission_level set default 10;
UPDATE person set permission_level = 10 where permission_level <= 10;

commit;
