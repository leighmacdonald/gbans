BEGIN;

DROP TABLE patreon_auth;

CREATE TABLE patreon_auth
(
    steam_id bigint primary key not null references person (steam_id),
    patreon_id text not null,
    access_token  text not null,
    expires_in    int  not null,
    scope         text not null,
    token_type    text not null,
    version       text not null
);

COMMIT;
