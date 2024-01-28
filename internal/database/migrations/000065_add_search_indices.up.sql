BEGIN;

ALTER TABLE IF EXISTS person_messages
    ADD name_search tsvector GENERATED ALWAYS AS (to_tsvector('simple', persona_name)) STORED;

CREATE INDEX idx_name_search ON person_messages USING GIN (name_search);

ALTER TABLE IF EXISTS person_messages
    add message_search tsvector GENERATED ALWAYS AS (to_tsvector('simple', body)) STORED;

CREATE INDEX idx_message_search ON person_messages USING GIN (message_search);
CREATE INDEX idx_message_steam_id ON person_messages (steam_id);
CREATE INDEX idx_created ON person_messages USING brin (created_on);

COMMIT;
