BEGIN;

create table host_key
(
    address text primary key,
    key text not null,
    created_on timestamp
);

COMMIT;
