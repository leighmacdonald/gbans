ALTER TABLE ban
ADD COLUMN IF NOT EXISTS asn_num BIGINT NOT NULL DEFAULT 0;

ALTER TABLE ban
ADD COLUMN IF NOT EXISTS cidr ip4r;

ALTER TABLE ban
ADD COLUMN IF NOT EXISTS name text not null default '';

INSERT INTO
  ban (
    target_id,
    source_id,
    ban_type,
    reason,
    valid_until,
    created_on,
    updated_on,
    cidr,
    deleted,
    is_enabled,
    appeal_state
  )
SELECT
  n.target_id,
  n.source_id,
  2,
  n.reason,
  n.valid_until,
  n.created_on,
  n.updated_on,
  cidr,
  deleted,
  is_enabled,
  appeal_state
FROM
  ban_net n;

INSERT INTO
  ban (
    target_id,
    source_id,
    ban_type,
    reason,
    valid_until,
    created_on,
    updated_on,
    asn_num,
    deleted,
    is_enabled,
    appeal_state
  )
SELECT
  n.target_id,
  n.source_id,
  2,
  n.reason,
  n.valid_until,
  n.created_on,
  n.updated_on,
  as_num,
  deleted,
  is_enabled,
  appeal_state
FROM
  ban_asn n;

INSERT INTO
  ban (
    target_id,
    source_id,
    ban_type,
    valid_until,
    created_on,
    updated_on,
    reason,
    deleted,
    is_enabled,
    appeal_state,
    name
  )
SELECT
  n.target_id,
  n.source_id,
  2, -- ban
  n.valid_until,
  n.created_on,
  n.updated_on,
  1, -- custom
  n.deleted,
  n.is_enabled,
  n.appeal_state,
  n.group_name
FROM
  ban_group n;

DROP TABLE ban_group;

DROP TABLE ban_asn;

DROP TABLE ban_net;

ALTER TABLE IF EXISTS ban
DROP COLUMN IF EXISTS include_friends;

ALTER TABLE IF EXISTS ban
DROP COLUMN IF EXISTS asn_num;

CREATE TABLE IF NOT EXISTS asn_ban (
  as_num INTEGER NOT NULL PRIMARY KEY,
  reason TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_on timestamp with time zone not null,
  updated_on timestamp with time zone not null
);

DROP TABLE IF EXISTS river_migration;

DROP TABLE IF EXISTS river_job;

DROP TABLE IF EXISTS river_queue;

DROP TABLE IF EXISTS river_client_queue;

DROP TABLE IF EXISTS river_client;

DROP TABLE IF EXISTS river_leader;
