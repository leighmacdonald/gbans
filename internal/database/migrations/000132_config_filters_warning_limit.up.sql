begin;

-- The filters_warning_limit column was missed when the config table was
-- created, so every config write failed with "column does not exist" and no
-- admin settings change could be saved. The default matches the filters form
-- ("Maximum number of warnings allowed").
alter table if exists config
    add column if not exists filters_warning_limit int not null default 3;

commit;
