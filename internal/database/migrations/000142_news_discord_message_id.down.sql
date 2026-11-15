begin;

alter table news
    drop column discord_message_id;

commit;
