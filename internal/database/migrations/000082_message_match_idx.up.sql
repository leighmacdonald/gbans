BEGIN;

create index person_messages_match_id_index
    on person_messages (match_id);

ALTER TABLE match ADD COLUMN demo_name TEXT NOT NULL default '';

COMMIT;
