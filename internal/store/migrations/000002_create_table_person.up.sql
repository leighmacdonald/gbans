begin;

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
create index if not exists idx_personaname_lower ON person (LOWER(personaname));

commit;