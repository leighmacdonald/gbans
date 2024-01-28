BEGIN;

alter table if exists demo add column if not exists map_name text not null ;
alter table if exists demo add column if not exists created_on timestamptz not null ;
alter table if exists demo add column if not exists archive bool not null default false;
alter table if exists demo add column if not exists stats jsonb not null default '{}';

CREATE INDEX demo_players_idx ON demo USING GIN ((stats));

COMMIT;
