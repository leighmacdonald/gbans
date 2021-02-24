create table if not exists person
(
    steam_id                 int8
        constraint player_pk primary key,
    created_on               timestamp       not null,
    updated_on               timestamp       not null,
    ip_addr                  text default '' not null,
    communityvisibilitystate int  default 0  not null,
    profilestate             int             not null,
    personaname              text            not null,
    profileurl               text            not null,
    avatar                   text            not null,
    avatarmedium             text            not null,
    avatarfull               text            not null,
    avatarhash               text            not null,
    personastate             int             not null,
    realname                 text            not null,
    timecreated              int             not null,
    loccountrycode           text            not null,
    locstatecode             text            not null,
    loccityid                int             not null
);

CREATE INDEX if not exists idx_personaname_lower ON person (LOWER(personaname));

-- GDPR violation?
CREATE TABLE IF NOT EXISTS person_names
(
    personaname_id BIGSERIAL PRIMARY KEY,
    steam_id       int8,
    personaname    text      not null,
    created_on     timestamp not null,
    CONSTRAINT fk_steam_id FOREIGN KEY (steam_id) REFERENCES person (steam_id)
);

create table if not exists ban
(
    ban_id      BIGSERIAL PRIMARY KEY,
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

create table if not exists ban_net
(
    net_id      BIGSERIAL PRIMARY KEY,
    cidr        cidr            not null,
    source      int  default 0  not null,
    created_on  timestamp       not null,
    updated_on  timestamp       not null,
    reason      text default '' not null,
    valid_until timestamp       not null
);

create unique index if not exists ban_net_cidr_uindex
    on ban_net (cidr);

create table if not exists server
(
    server_id        SERIAL PRIMARY KEY,
    short_name       varchar(32)            not null,
    token            varchar(40) default '' not null,
    address          varchar(128)           not null,
    port             int                    not null,
    rcon             varchar(128)           not null,
    token_created_on timestamp,
    reserved_slots   smallint               not null,
    created_on       timestamp              not null,
    updated_on       timestamp              not null,
    password         varchar(20)            not null
);

create unique index if not exists server_name_uindex
    on server (short_name);

create table if not exists filtered_word
(
    word_id SERIAL PRIMARY KEY,
    word    varchar not null
);

create unique index if not exists filtered_word_word_uindex
    on filtered_word (word);

create table if not exists ban_appeal
(
    appeal_id    SERIAL PRIMARY KEY,
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

CREATE TABLE IF NOT EXISTS filtered_word
(
    word_id    BIGSERIAL PRIMARY KEY,
    word       text,
    created_on timestamp not null
);
