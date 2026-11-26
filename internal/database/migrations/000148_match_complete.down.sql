begin;

alter table if exists match
    drop column if exists is_complete;

commit;
