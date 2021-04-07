begin;
create table if not exists ban
(
    ban_id      bigserial primary key,
    steam_id    int8            not null,
    author_id   int8 default 0  not null,
    ban_type    int             not null,
    reason      int             not null,
    reason_text text default '' not null,
    note        text default '' not null,
    valid_until timestamp       not null,
    created_on  timestamp       not null,
    updated_on  timestamp       not null,
    ban_source  int  default 0  not null,
    CONSTRAINT fk_person FOREIGN KEY (steam_id) REFERENCES person (steam_id)
);

create unique index if not exists ban_steam_id_uindex
    on ban (steam_id);

commit;