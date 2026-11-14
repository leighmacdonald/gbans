begin;

alter table if exists config
    drop column if exists stats_refresh_schedule;

commit;
