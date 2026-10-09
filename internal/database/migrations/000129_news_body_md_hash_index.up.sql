begin;

drop index if exists news_body_md_uindex;
create unique index if not exists news_body_md_uindex on news (md5(body_md));

commit;
