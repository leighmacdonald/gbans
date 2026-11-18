begin;

alter table if exists config
    drop column if exists stats_balance_algorithm;

commit;
