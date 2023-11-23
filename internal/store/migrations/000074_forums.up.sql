BEGIN;

CREATE TABLE forum_category
(
    category_id serial primary key,
    title       text        not null unique,
    description text        not null default '',
    ordering    int         not null default 0,
    created_on  timestamptz not null,
    updated_on  timestamptz not null
);

CREATE TABLE forum
(
    forum_id       serial primary key,
    category_id    int         not null references forum_category (category_id),
    last_thread_id bigint references forum_thread (thread_id),
    title          text        not null unique,
    description    text        not null default '',
    ordering       int         not null default 0,
    count_threads  bigint      not null default 0,
    count_messages bigint      not null default 0,
    created_on     timestamptz not null,
    updated_on     timestamptz not null
);

CREATE UNIQUE INDEX ON forum (category_id, title);

CREATE TABLE forum_thread
(
    thread_id  bigserial primary key,
    forum_id   int         not null references forum (forum_id),
    source_id  bigserial   not null references person (steam_id),
    title          text        not null unique,
    sticky     bool        not null default false,
    locked     bool        not null default false,
    views      bigint      not null default 0,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

CREATE UNIQUE INDEX ON forum_thread (forum_id, title);

CREATE TABLE forum_message
(
    message_id bigserial primary key,
    thread_id  bigserial references forum_thread (thread_id),
    source_id  bigserial   not null references person (steam_id),
    body_md    text        not null,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

CREATE TABLE forum_message_vote
(
    message_id bigserial primary key,
    source_id  bigserial   not null references person (steam_id),
    vote       integer     not null CHECK ( vote == 1 OR vote == -1 ),
    created_on timestamptz not null,
    updated_on timestamptz not null
);

COMMIT;
