BEGIN;

CREATE TABLE person_settings (
    person_settings_id bigserial primary key,
    steam_id bigint not null primary key,
    forum_signature text not null default '',
    forum_profile_messages bool not null default true,
    stats_hidden bool not null default false

);

COMMIT;
