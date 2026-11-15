begin;

-- Tracks the discord message posted for a published article so edits can
-- update the channel message in place instead of posting a duplicate.
alter table news
    add column discord_message_id bigint;

commit;
