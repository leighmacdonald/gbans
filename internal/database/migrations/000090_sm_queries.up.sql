BEGIN;

-- Add temp tables
CREATE TABLE cidr_block_entries
(
    cidr_block_entries_id bigserial primary key,
    cidr_block_source_id  int       NOT NULL REFERENCES cidr_block_source (cidr_block_source_id) ON DELETE CASCADE,
    net_block             ip4r      not null,
    created_on            timestamp not null
);

CREATE INDEX ON cidr_block_entries using gist (net_block);

CREATE TABLE steam_group_members
(
    steam_id   bigint    not null REFERENCES person (steam_id) ON DELETE CASCADE,
    group_id   bigint    not null, -- Cannot use as FK as ban_group.group_id is not unique constrained
    created_on timestamp not null,
    PRIMARY KEY (steam_id, group_id)
);

CREATE INDEX on steam_group_members (steam_id);

CREATE TABLE steam_friends
(
    steam_id   bigint    not null REFERENCES person (steam_id) ON DELETE CASCADE,
    friend_id  bigint    not null REFERENCES person (steam_id) ON DELETE CASCADE,
    created_on timestamp not null,
    PRIMARY KEY (steam_id, friend_id)
);

CREATE INDEX on steam_friends (friend_id);

-- Migrate more column types to use ip4r extension
ALTER TABLE net_asn
    ALTER COLUMN ip_range TYPE ip4r;
ALTER TABLE net_location
    ALTER COLUMN ip_range TYPE ip4r;
ALTER TABLE ban_net
    ALTER COLUMN cidr TYPE ip4r;
ALTER TABLE cidr_block_whitelist
    ALTER COLUMN address TYPE ip4r;

-- select steam_to_steam64('STEAM_0:1:583502767'); -- -> 76561199127271263
-- select steam_to_steam64('76561199127271263'); -- -> 76561199127271263

-- Perform ban lookups for both the players steamid and IP. Accepts Steam and Steam64 string
-- inputs. Include some ability to also support ignoring whitelisted matches. Currently missing support
-- for the evade_ok exceptions for bans.
--
-- Seems more than fast enough @ ~10ms per execution on old i7-6700 CPU & Samsung 850 Pro SSD using
-- full mirror of ut dataset.
CREATE OR REPLACE FUNCTION check_ban(steam text, ip text,
                                     OUT out_ban_source text,
                                     OUT out_ban_id int,
                                     OUT out_reason int,
                                     OUT out_evade_ok bool,
                                     OUT out_valid_until timestamp,
                                     OUT out_ban_type int) AS
$$
DECLARE
    in_steam_id bigint ;
BEGIN
    in_steam_id := steam_to_steam64(steam);

    -- These are executed in *roughly* the order of least expensive to most
    SELECT 'ban_steam', ban_id, ban_type, reason, evade_ok, valid_until
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban
    WHERE deleted = false
      AND target_id = in_steam_id
      AND valid_until > now();

    if out_ban_id > 0 then
        return;
    end if;

    SELECT 'ban_steam_friend', 1, 2, 15, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM steam_friends
    WHERE friend_id = in_steam_id;

    if out_ban_id > 0 then
        return;
    end if;

    SELECT 'steam_group', 1, 2, 16, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM steam_group_members
    WHERE steam_id = in_steam_id;

    if out_ban_id > 0 then
        return;
    end if;

    SELECT 'ban_net', net_id, 2, reason, false, valid_until
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban_net
    WHERE deleted = false
      AND (ip::ip4 <<= cidr OR target_id = in_steam_id)
      AND valid_until > now();

    if out_ban_id > 0 then
        return;
    end if;

    SELECT 'cidr_block', 1, 2, 14, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM cidr_block_entries
    WHERE ip::ip4 <<= net_block
      AND NOT ip::ip4 IN (SELECT address FROM cidr_block_whitelist);

    if out_ban_id > 0 then
        return;
    end if;

    SELECT 'ban_asn', 1, 2, 17, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban_asn
             LEFT JOIN net_asn na on ban_asn.as_num = na.as_num
    WHERE ip::ip4 <<= na.ip_range
      AND NOT ip::ip4 IN (SELECT address FROM cidr_block_whitelist);

    if out_ban_id > 0 then
        return;
    end if;

END
$$
    LANGUAGE plpgsql;


-- SELECT * from check_ban('STEAM_1:1:566689572', '1.2.3.4');
-- SELECT 'ban_steam', *
-- from check_ban('76561198820293485', '1.2.3.4') -- ban_steam bigint
-- UNION
-- SELECT 'ban_steam2', *
-- from check_ban('STEAM_0:1:430013878', '1.2.3.4') -- ban_steam STEAM_0:1:430013878
-- UNION
-- SELECT 'ban_steam_friend', *
-- from check_ban('STEAM_0:0:58744148', '1.2.3.4') -- ban_steam_friend
-- UNION
-- SELECT 'ban_net', *
-- from check_ban('STEAM_1:1:566689574', '162.222.198.2') -- ban_net
-- UNION
-- SELECT 'cidr_block_whitelist', *
-- from check_ban('STEAM_1:1:566689573', '1.12.36.4') -- cidr_block_whitelist
-- UNION
-- SELECT 'cidr_block', *
-- from check_ban('STEAM_1:1:566689573', '2.57.68.6') -- cidr_block
-- UNION
-- SELECT 'steam_group', *
-- FROM check_ban('76561198011576839', '7.7.7.7') -- steam_group
-- UNION
-- SELECT 'ban_asn', *
-- FROM check_ban('76561198820293489', '1.1.8.2');
-- ban_asn
-- 8 rows retrieved starting from 1 in 89 ms (execution: 74 ms, fetching: 15 ms)

COMMIT;
