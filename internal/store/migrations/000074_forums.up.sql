BEGIN;

CREATE TABLE forum_category
(
    forum_category_id serial primary key,
    title             text        not null unique,
    description       text        not null default '',
    ordering          int         not null default 0,
    created_on        timestamptz not null,
    updated_on        timestamptz not null
);

CREATE TABLE forum
(
    forum_id          serial primary key,
    forum_category_id int         not null references forum_category (forum_category_id),
    last_thread_id    bigint,
    title             text        not null unique,
    description       text        not null default '',
    ordering          int         not null default 0,
    count_threads     bigint      not null default 0,
    count_messages    bigint      not null default 0,
    permission_level  int         not null default 1,
    created_on        timestamptz not null,
    updated_on        timestamptz not null
);

CREATE UNIQUE INDEX ON forum (forum_category_id, title);

CREATE TABLE forum_thread
(
    forum_thread_id       bigserial primary key,
    forum_id              int         not null references forum (forum_id),
    source_id             bigserial   not null references person (steam_id),
    title                 text        not null unique,
    sticky                bool        not null default false,
    locked                bool        not null default false,
    views                 bigint      not null default 0,
    last_forum_message_id bigint,
    created_on            timestamptz not null,
    updated_on            timestamptz not null
);


CREATE UNIQUE INDEX ON forum_thread (forum_id, title);

ALTER TABLE forum
    ADD CONSTRAINT fk_last_thread_id
        FOREIGN KEY (last_thread_id) REFERENCES forum_thread (forum_thread_id) ON DELETE SET NULL;

CREATE TABLE forum_message
(
    forum_message_id bigserial primary key,
    forum_thread_id  bigserial references forum_thread (forum_thread_id),
    source_id        bigserial   not null references person (steam_id),
    body_md          text        not null,
    created_on       timestamptz not null,
    updated_on       timestamptz not null
);

ALTER TABLE forum_thread
    ADD CONSTRAINT fk_last_thread_id
        FOREIGN KEY (last_forum_message_id) REFERENCES forum_message (forum_message_id) ON DELETE CASCADE;

CREATE TABLE forum_message_vote
(
    forum_message_vote_id bigserial primary key,
    forum_message_id      bigserial references forum_message (forum_message_id) ON DELETE CASCADE,
    source_id             bigserial   not null references person (steam_id),
    vote                  integer     not null CHECK ( vote = 1 OR vote = -1 ),
    created_on            timestamptz not null,
    updated_on            timestamptz not null
);

CREATE UNIQUE INDEX ON forum_message_vote (forum_message_id, source_id);

COMMIT;
