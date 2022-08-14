begin;

drop table if exists ban_group;

alter table if exists ban
    rename column target_id to steam_id;
alter table if exists ban
    rename column source_id to author_id;
alter table if exists ban
    rename column origin to source;

alter table if exists ban
    drop column if exists is_enabled;

alter table if exists ban_asn
    drop column if exists  is_enabled;

alter table if exists ban_net
    drop column if exists note;
alter table if exists ban_net
    drop column if exists unban_reason_text;
alter table if exists ban_net
    drop column if exists  is_enabled;
alter table if exists ban_net
    drop column if exists  author_id;
alter table if exists ban_net
    drop column if exists  target_id;

alter table if exists ban_net
    rename column origin to source;

alter table if exists ban_asn
    rename column  source_id to author_id;

alter table if exists ban_asn
    drop column if exists unban_reason_text;

commit;
