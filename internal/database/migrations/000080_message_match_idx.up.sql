BEGIN;

create index person_messages_match_id_index
    on person_messages (match_id);

COMMIT;
