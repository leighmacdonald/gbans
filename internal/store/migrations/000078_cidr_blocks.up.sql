BEGIN;

CREATE TABLE cidr_block_source (
    cidr_block_source_id serial primary key,
    name text not null unique,
    url text not null unique,
    enabled bool not null default true,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

CREATE TABLE cidr_block_whitelist (
    cidr_block_whitelist_id serial primary key,
    address cidr not null unique,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

COMMIT;
