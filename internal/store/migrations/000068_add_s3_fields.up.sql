BEGIN;

-- ALTER TABLE demo
--     DROP COLUMN IF EXISTS raw_data;
-- ALTER TABLE demo
--     DROP COLUMN IF EXISTS size;
-- ALTER TABLE demo
--     DROP COLUMN IF EXISTS downloads;
--
-- ALTER TABLE media
--     DROP COLUMN IF EXISTS mime_type;
-- ALTER TABLE media
--     DROP COLUMN IF EXISTS size;
-- ALTER TABLE media
--     DROP COLUMN IF EXISTS contents;


CREATE TABLE asset
(
    asset_id  uuid primary key default gen_random_uuid(),
    bucket    text                        not null,
    path      text                        not null,
    mime_type text                        not null,
    size      bigint                      not null,
    name      text             default '' not null,
    old_id    bigint           default 0
);

ALTER TABLE demo
    ADD COLUMN asset_id uuid
        CONSTRAINT demo_asset_fk
            REFERENCES asset ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE media
    ADD COLUMN asset_id uuid
        CONSTRAINT media_asset_fk
            REFERENCES asset ON UPDATE CASCADE ON DELETE CASCADE;

-- ALTER TABLE demo
--     ADD CONSTRAINT demo_s3_or_local_check
--         CHECK ((raw_data IS NOT NULL OR asset_id IS NOT NULL) AND NOT (raw_data IS NOT NULL AND asset_id IS NOT NULL));
--
-- ALTER TABLE media
--     ADD CONSTRAINT media_s3_or_local_check
--         CHECK ((contents IS NOT NULL OR asset_id IS NOT NULL) AND NOT (contents IS NOT NULL AND asset_id IS NOT NULL));


ALTER TABLE media
    ALTER COLUMN contents drop not null;

COMMIT;
