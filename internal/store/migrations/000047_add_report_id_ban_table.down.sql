begin;
alter table if exists ban drop column if exists report_id;

alter table if exists ban drop constraint if exists ban_report_report_id_fk;

commit;
