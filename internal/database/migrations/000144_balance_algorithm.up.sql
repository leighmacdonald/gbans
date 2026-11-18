begin;

-- Team balancing implementation selected from the admin Stats settings. The
-- default keeps the simple alternating split for existing installations.
alter table if exists config
    add column if not exists stats_balance_algorithm text not null default 'alternating';

commit;
