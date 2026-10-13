begin;

alter table if exists config
    drop column if exists filters_warning_limit;

commit;
