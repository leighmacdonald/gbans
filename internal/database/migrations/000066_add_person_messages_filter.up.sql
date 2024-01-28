BEGIN;

CREATE TABLE IF NOT EXISTS person_messages_filter
(
    person_message_filter_id bigserial primary key,
    person_message_id        bigint
        constraint person_message_filter_message_id_fk
            references person_messages
            on update cascade on delete cascade,
    filter_id                bigint
        constraint person_message_filter_filter_id_fk
            references filtered_word
            on update cascade on delete cascade
);

create unique index person_messages_match_uindex
    on person_messages_filter (person_message_id, filter_id);

ALTER TABLE report
    ADD column person_message_id bigint
        constraint report_person_message_id_fk
            references person_messages
            on update cascade on delete cascade;

COMMIT;
