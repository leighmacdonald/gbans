BEGIN;

CREATE TABLE IF NOT EXISTS playerqueue_messages
(
    message_id  uuid primary key,
    steam_id    bigint      not null references person (steam_id) ON DELETE CASCADE,
    personaname text        not null,
    avatarhash  text        not null,
    created_on  timestamptz not null,
    body_md     text        not null check ( length(body_md) > 0 )
);

CREATE INDEX playerqueue_messages_created_on_idx ON playerqueue_messages (created_on);

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS general_playerqueue_enabled bool not null default false;

COMMIT;
