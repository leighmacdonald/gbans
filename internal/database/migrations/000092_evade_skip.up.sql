BEGIN;

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
    is_whitelist bigint;
    is_whitelist_addr bool;
BEGIN
    in_steam_id := steam_to_steam64(steam);

    -- These are executed in *roughly* the order of least expensive to most
    SELECT 'ban_steam', ban_id, ban_type, reason, evade_ok, valid_until
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban
    WHERE deleted = false
      AND valid_until > now()
      AND (target_id = in_steam_id OR ( evade_ok = false AND last_ip = ip::inet ));

    if out_ban_id > 0 and out_evade_ok then
        SELECT true INTO is_whitelist_addr FROM cidr_block_whitelist WHERE ip::ip4 <<= address LIMIT 1;

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


    SELECT steam_id INTO is_whitelist FROM person_whitelist where steam_id = in_steam_id;
    if is_whitelist > 0 then
        return;
    end if;

    SELECT true INTO is_whitelist_addr FROM cidr_block_whitelist WHERE ip::ip4 <<= address LIMIT 1;
    if is_whitelist_addr then
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
    WHERE ip::ip4 <<= net_block;

    if out_ban_id > 0 then
        return;
    end if;

    SELECT 'ban_asn', 1, 2, 17, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban_asn
             LEFT JOIN net_asn na on ban_asn.as_num = na.as_num
    WHERE ip::ip4 <<= na.ip_range;

    if out_ban_id > 0 then
        return;
    end if;

END
$$
    LANGUAGE plpgsql;
COMMIT;

SELECT 'ban_steam', *
from check_ban('76561198084134025', '192.168.0.57') -- ban_steam bigint
