BEGIN;

create table patreon_auth
(
    creator_access_token text not null,
    creator_refresh_token text not null
);

insert into patreon_auth values ('', '');

COMMIT;
