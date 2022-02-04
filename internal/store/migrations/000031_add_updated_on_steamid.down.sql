begin;

alter table if exists person drop column if exists updated_on_steam;

commit;