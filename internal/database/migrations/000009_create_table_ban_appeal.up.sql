begin;
create table if not exists ban_appeal
(
    appeal_id    serial primary key,
    ban_id       int8          not null references ban,
    appeal_text  text          not null,
    appeal_state int default 0 not null,
    email        text          not null,
    created_on   timestamp     not null,
    updated_on   timestamp     not null,
    CONSTRAINT fk_ban_id FOREIGN KEY (ban_id) REFERENCES ban (ban_id)
);

create unique index if not exists ban_appeal_ban_id_uindex
    on ban_appeal (ban_id);

commit;