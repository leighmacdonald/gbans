begin;

drop table if exists report_media;

alter table if exists report drop column if exists title;

alter table if exists wiki drop column if exists title;

alter table if exists wiki_media rename column wiki_media_id to media_id;

alter table if exists wiki_media rename to media;

commit;
