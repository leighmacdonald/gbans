BEGIN;

DROP TABLE match_team;

TRUNCATE TABLE match cascade;

ALTER TABLE match ADD column match_raw jsonb not null;
ALTER TABLE match ADD column winner int not null;


ALTER TABLE match_medic DROP COLUMN charges;
ALTER TABLE match_medic ADD column charge_uber int not null default 0;
ALTER TABLE match_medic ADD column charge_kritz int not null default 0;
ALTER TABLE match_medic ADD column charge_vacc int not null default 0;
ALTER TABLE match_medic ADD column charge_quickfix int not null default 0;


create table if not exists weapon (
    weapon_id serial primary key,
    key text unique,
    name text unique
);

COMMIT;
