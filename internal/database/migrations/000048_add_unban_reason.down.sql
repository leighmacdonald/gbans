begin;

alter table if exists ban drop column if exists unban_reason_text;

alter table ban_net drop column if exists reason;
alter table ban_asn drop column if exists reason;

alter table ban_net rename column reason_text to reason;
alter table ban_asn rename column reason_text to reason;



commit;
