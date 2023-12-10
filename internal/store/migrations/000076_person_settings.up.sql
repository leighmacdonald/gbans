BEGIN;

CREATE TABLE person_settings (
    person_settings_id bigserial primary key,
    steam_id bigint not null,
    forum_signature text not null default '',
    forum_profile_messages bool not null default true,
    stats_hidden bool not null default false,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

ALTER TABLE person_settings
    ADD CONSTRAINT fk_steam_id_settings FOREIGN KEY (steam_id) REFERENCES person (steam_id) ON DELETE CASCADE   ;

COMMIT;
