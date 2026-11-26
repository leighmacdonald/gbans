begin;

-- Matches without a game_over event ended before the map concluded (aborted
-- recording, map change or restart). They stay available for stats browsing
-- but are excluded from rating/skill calculations.
alter table if exists match
    add column if not exists is_complete bool not null default true;

commit;
