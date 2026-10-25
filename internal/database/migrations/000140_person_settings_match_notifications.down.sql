begin;

alter table if exists person_settings
    drop column if exists match_notifications;

commit;
