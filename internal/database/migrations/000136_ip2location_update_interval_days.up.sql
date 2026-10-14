begin;

-- How often (in days) the built in tracker re-downloads and imports the
-- ip2location databases. A value of 0 disables automatic updates, leaving
-- `gbans refresh location` as the only refresh path.
alter table if exists config
    add column if not exists ip2location_update_interval_days int not null default 30;

commit;
