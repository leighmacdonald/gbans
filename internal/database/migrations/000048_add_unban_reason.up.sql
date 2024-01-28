begin;

alter table ban add column if not exists unban_reason_text text default '' not null;

alter table ban_net rename column reason to reason_text;
alter table ban_asn rename column reason to reason_text;

alter table ban_net add reason int default 1 not null;
alter table ban_asn add reason int default 1 not null;

commit;
