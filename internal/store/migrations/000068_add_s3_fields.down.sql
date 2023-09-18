BEGIN;

alter table demo
    drop column if exists asset_id;

alter table media
    drop column if exists asset_id;

drop table if exists asset;

COMMIT;
