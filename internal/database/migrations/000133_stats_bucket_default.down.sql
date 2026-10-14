begin;

alter table if exists match
    alter column stats_bucket_id set default 1;

drop index if exists stats_bucket_one_default_idx;

alter table if exists stats_bucket
    drop column if exists is_default;

commit;
