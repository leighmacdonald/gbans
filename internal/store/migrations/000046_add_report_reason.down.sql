begin;

alter table if exists report drop column if exists reason;
alter table if exists report drop column if exists reason_text;

commit;
