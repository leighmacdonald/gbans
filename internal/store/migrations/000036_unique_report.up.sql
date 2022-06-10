begin;

create unique index report_author_id_reported_id_uindex
    on report (author_id, reported_id);

commit;
