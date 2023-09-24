BEGIN;

CREATE TABLE contest
(
    contest_id      uuid primary key     default gen_random_uuid(),
    title           text        not null unique,
    public          bool        not null default true,
    description     text        not null,
    date_start      timestamptz not null,
    date_end        timestamptz not null,
    max_submissions int         not null default 1,
    media_types     text        not null default '',
    deleted         bool        not null default false,
    created_on      timestamptz not null,
    updated_on      timestamptz not null
);

CREATE TABLE contest_entry
(
    contest_entry_id uuid primary key     default gen_random_uuid(),
    contest_id       uuid        not null references contest,
    steam_id         bigint      not null references person,
    asset_id         uuid        not null references asset,
    description      text        not null default '',
    placement        int         not null default 0,
    deleted          bool        not null default false,
    created_on       timestamptz not null,
    updated_on       timestamptz not null
);

COMMIT;
