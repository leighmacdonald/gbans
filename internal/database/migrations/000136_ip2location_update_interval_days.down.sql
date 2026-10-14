begin;

alter table if exists config
    drop column if exists ip2location_update_interval_days;

commit;
