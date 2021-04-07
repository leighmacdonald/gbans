create table if not exists person_names
(
    personaname_id bigserial primary key,
    steam_id       int8,
    personaname    text      not null,
    created_on     timestamp not null,
    constraint fk_steam_id foreign key (steam_id) references person (steam_id)
);