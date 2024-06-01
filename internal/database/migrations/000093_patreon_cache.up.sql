BEGIN;

-- missing from older
create or replace function steam_to_steam64(steam_id text) returns bigint
    language plpgsql
as
$$
DECLARE
    parts text[];
BEGIN
    if starts_with(steam_id, '76561') then return cast(steam_id as bigint); end if;

    parts := regexp_matches(steam_id, '^STEAM_([0-5]):([0-1]):([0-9]+)$');
    return (cast(parts[3] as bigint) * 2) + 76561197960265728 + cast(parts[2] as bigint);
END ;
$$;

DROP TABLE patreon_auth;

CREATE TABLE auth_patreon
(
    steam_id bigint primary key not null references person (steam_id),
    patreon_id text not null,
    access_token  text not null,
    refresh_token text not null,
    expires_in    int  not null,
    scope         text not null,
    token_type    text not null,
    version       text not null,
    created_on timestamp not null,
    updated_on timestamp not null
);

CREATE TABLE discord_user
(
    discord_id text primary key,
    steam_id bigint not null references person (steam_id),
    username text not null default '',
    avatar text not null default '',
    publicFlags int not null default 0,
    mfa_enabled bool not null default false,
    premium_type int not null default 0,
    created_on timestamp not null,
    updated_on timestamp not null
);

CREATE TABLE auth_discord
(
    steam_id bigint primary key not null references person (steam_id),
    discord_id text not null references discord_user (discord_id) ON DELETE CASCADE,
    access_token  text not null,
    refresh_token text not null,
    expires_in    int  not null,
    scope         text not null,
    token_type    text not null,
    created_on timestamp not null,
    updated_on timestamp not null
);


COMMIT;
