BEGIN;
-- Basically taken from https://github.com/alliedmodders/sourcemod/blob/master/configs/sql-init-scripts/pgsql/create_admins.sql
-- A few extra constraints and fk's added

CREATE TABLE sm_admins
(
    id         serial,
    steam_id   bigint REFERENCES person (steam_id) ON DELETE CASCADE,
    authtype   varchar(6)  NOT NULL CHECK ( authtype in ('steam', 'name', 'ip') ),
    identity   varchar(65) NOT NULL,
    password   varchar(65),
    flags      varchar(30) NOT NULL,
    name       varchar(65) NOT NULL,
    immunity   int         NOT NULL,
    created_on timestamp   NOT NULL,
    updated_on timestamp   NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE sm_groups
(
    id             serial,
    flags          varchar(30)  NOT NULL,
    name           varchar(120) NOT NULL UNIQUE,
    immunity_level int          NOT NULL,
    created_on     timestamp    NOT NULL,
    updated_on     timestamp    NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE sm_group_immunity
(
    group_immunity_id serial primary key,
    group_id   int       NOT NULL,
    other_id   int       NOT NULL,
    created_on timestamp NOT NULL,
    FOREIGN KEY (group_id) REFERENCES sm_groups (id) ON DELETE CASCADE,
    FOREIGN KEY (other_id) REFERENCES sm_groups (id) ON DELETE CASCADE
);

CREATE INDEX ON sm_group_immunity (group_id, other_id);

CREATE TABLE sm_group_overrides
(
    group_override_id serial primary key,
    group_id   int         NOT NULL,
    FOREIGN KEY (group_id) REFERENCES sm_groups (id) ON DELETE CASCADE,
    type       varchar(10) NOT NULL CHECK (type in ('command', 'group')),
    name       varchar(32) NOT NULL,
    access     varchar(5)  NOT NULL CHECK (access in ('allow', 'deny')),
    created_on timestamp   NOT NULL,
    updated_on timestamp   NOT NULL
);

CREATE INDEX ON sm_group_overrides (group_id, type, name);

CREATE TABLE sm_overrides
(
    override_id serial primary key,
    type       varchar(10) NOT NULL CHECK (type in ('command', 'group')),
    name       varchar(32) NOT NULL,
    flags      varchar(30) NOT NULL,
    created_on timestamp   NOT NULL,
    updated_on timestamp   NOT NULL
);

CREATE INDEX ON sm_overrides (type, name);

CREATE TABLE sm_admins_groups
(
    admin_id      int       NOT NULL REFERENCES sm_admins (id) ON DELETE CASCADE,
    group_id      int       NOT NULL REFERENCES sm_groups (id) ON DELETE CASCADE,
    inherit_order int       NOT NULL,
    created_on    timestamp NOT NULL,
    updated_on    timestamp NOT NULL,
    PRIMARY KEY (admin_id, group_id)
);

CREATE TABLE sm_config
(
    cfg_key   varchar(32)  NOT NULL,
    cfg_value varchar(255) NOT NULL,
    PRIMARY KEY (cfg_key)
);

-- Probably not necessary, but satisfy the current schema version, so it does not try
-- and auto update the schema.
-- https://github.com/alliedmodders/sourcemod/blob/master/plugins/sql-admin-manager.sp#L40
INSERT INTO sm_config (cfg_key, cfg_value) VALUES ('admin_version', '1.0.0.1409');

COMMIT;
