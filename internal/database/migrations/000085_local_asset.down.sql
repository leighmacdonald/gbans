BEGIN;

alter table asset
    drop column hash,
    drop column author_id,
    drop column is_private,
    add column old_id int;

COMMIT;
