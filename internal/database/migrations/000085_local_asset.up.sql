begin ;
truncate contest_entry_vote;

truncate  contest_entry cascade ;

truncate  asset cascade ;

truncate demo;

alter table asset
    add column hash bytea not null,
    add column author_id bigint not null default 0 references person (steam_id),
    add column is_private bool not null default false,
    add column updated_on timestamp,
    add column created_on timestamp,
    drop column if exists old_id,
    drop column if exists path;

drop table if exists media;

commit;