begin;

alter table ban add report_id int;

alter table ban add constraint ban_report_report_id_fk  foreign key (report_id) references report;

commit;
