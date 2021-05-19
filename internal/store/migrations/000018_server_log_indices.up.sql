begin;

CREATE INDEX IF NOT EXISTS server_log_source_id_idx  on server_log (source_id);
CREATE INDEX IF NOT EXISTS server_log_target_id_idx  on server_log (target_id);

commit;
