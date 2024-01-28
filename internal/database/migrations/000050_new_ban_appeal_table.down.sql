begin;

drop table if exists ban_appeal;

create table ban_appeal
(
    appeal_id    serial
        primary key,
    ban_id       bigint            not null
        references ban
        constraint fk_ban_id
            references ban,
    appeal_text  text              not null,
    appeal_state integer default 0 not null,
    email        text              not null,
    created_on   timestamp         not null,
    updated_on   timestamp         not null
);

commit;
