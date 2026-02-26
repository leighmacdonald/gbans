ALTER TABLE config
    ADD COLUMN general_site_description text CHECK ( length(general_site_description) <= 155) default '';
