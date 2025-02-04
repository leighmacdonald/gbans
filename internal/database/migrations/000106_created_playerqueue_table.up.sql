BEGIN;

CREATE TABLE IF NOT EXISTS playerqueue_messages
(
    message_id  uuid primary key,
    steam_id    bigint      not null references person (steam_id) ON DELETE CASCADE,
    personaname text        not null,
    avatarhash  text        not null,
    created_on  timestamptz not null,
    deleted     bool        not null default false,
    body_md     text        not null check ( length(body_md) > 0 )
);

CREATE INDEX playerqueue_messages_created_on_idx ON playerqueue_messages (created_on);

ALTER TABLE config
    ADD COLUMN IF NOT EXISTS general_playerqueue_enabled bool not null default false;

BEGIN;

DO
$$
    BEGIN
        CREATE TYPE chat_status AS ENUM (
            'readwrite',
            'readonly',
            'noaccess'
            );
    EXCEPTION
        WHEN duplicate_object THEN null;
    END
$$;


ALTER TABLE person
    ADD COLUMN IF NOT EXISTS playerqueue_chat_status chat_status DEFAULT 'readwrite';

COMMIT;
