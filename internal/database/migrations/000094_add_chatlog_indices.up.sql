BEGIN;

CREATE INDEX person_messages_server_id_idx ON person_messages (server_id);

COMMIT;
