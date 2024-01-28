begin;

alter table if exists person drop column if exists muted;

alter table if exists ban drop column if exists appeal_state;
alter table if exists ban_group drop column if exists appeal_state;
alter table if exists ban_net drop column if exists appeal_state;
alter table if exists ban_asn drop column if exists appeal_state;

commit;
