begin;

-- Cron expression controlling when the stats materialized views are refreshed.
-- The default preserves the previous behaviour: daily at 00:05 server time.
alter table if exists config
    add column if not exists stats_refresh_schedule text not null default '5 0 * * *';

commit;
