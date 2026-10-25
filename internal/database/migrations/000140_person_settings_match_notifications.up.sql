begin;

-- Opt-in preference for in-game notifications when a match the player took part
-- in has been imported. Disabled by default.
alter table if exists person_settings
    add column if not exists match_notifications boolean not null default false;

commit;
