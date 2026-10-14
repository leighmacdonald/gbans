begin;

-- A default stats bucket so that servers and matches without an explicit
-- bucket are filed under a known group (casual) instead of a hardcoded id.
-- At most one bucket may be the default.
alter table if exists stats_bucket
    add column if not exists is_default boolean not null default false;

update stats_bucket
set is_default = true
where bucket_name = 'casual';

create unique index if not exists stats_bucket_one_default_idx
    on stats_bucket ((true))
    where is_default;

-- Matches always carry an explicit bucket (resolved from the server or the
-- default bucket at import time), so the hardcoded column default is removed.
alter table if exists match
    alter column stats_bucket_id drop default;

commit;
