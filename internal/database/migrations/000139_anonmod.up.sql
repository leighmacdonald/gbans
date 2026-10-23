begin;

-- Toggle for the anonymized moderator (anonmod) feature. When enabled,
-- moderator replies to reports and ban appeals are shown to non-moderators
-- under a randomly assigned fake name/avatar, unique per issue.
alter table if exists config
    add column if not exists general_anonmod_enabled boolean not null default false;

-- Stable per-issue anonymous persona for each moderator author. Each
-- (issue_type, issue_id, author_id) triple maps to exactly one random
-- name/avatar, so a moderator stays consistent across all their replies
-- within a single report or ban appeal.
create table if not exists anonmod_identity
(
    issue_type       text                         not null,
    issue_id         integer                      not null,
    author_id        bigint                       not null,
    anon_name        text                         not null,
    anon_avatar_hash text                         not null,
    created_on       timestamp with time zone     not null default now(),
    constraint anonmod_identity_pkey primary key (issue_type, issue_id, author_id)
);

commit;
