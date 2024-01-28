begin;

create table wiki
(
    slug       text      not null,
    title      text      not null,
    body_md    text      not null,
    revision   int       not null,
    created_on timestamp not null,
    updated_on timestamp not null
);

create unique index wiki_slug_revision_uindex
    on wiki (slug, revision);

commit;
