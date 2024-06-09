BEGIN;
CREATE OR REPLACE FUNCTION steam_to_steam64(steam_id text) RETURNS bigint
    LANGUAGE plpgsql
as
$func$
DECLARE
    parts text[];
BEGIN
    if starts_with(steam_id, '76561') then return cast(steam_id as bigint); end if;

    parts := regexp_matches(steam_id, '^STEAM_([0-5]):([0-1]):([0-9]+)$');
    return (cast(parts[3] as bigint) * 2) + 76561197960265728 + cast(parts[2] as bigint);
END ;
$func$;

CREATE OR REPLACE FUNCTION check_ban(steam text, ip text,
                                     OUT out_ban_source text,
                                     OUT out_ban_id int,
                                     OUT out_reason int,
                                     OUT out_evade_ok bool,
                                     OUT out_valid_until timestamp,
                                     OUT out_ban_type int) AS
$func$
DECLARE
    in_steam_id       bigint ;
    is_whitelist_sid  bool;
    is_whitelist_addr bool;
BEGIN
    in_steam_id := steam_to_steam64(steam);

    SELECT true INTO is_whitelist_addr FROM cidr_block_whitelist WHERE ip::ip4 <<= address LIMIT 1;
    SELECT true INTO is_whitelist_sid FROM person_whitelist where steam_id = in_steam_id;

    is_whitelist_addr = coalesce(is_whitelist_addr, false);
    is_whitelist_sid = coalesce(is_whitelist_sid, false);

    -- These are executed in *roughly* the order of least expensive to most
    SELECT 'ban_steam', ban_id, ban_type, reason, evade_ok, valid_until
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban
    WHERE deleted = false
        AND valid_until > now()
        AND target_id = in_steam_id;

    IF out_ban_id > 0 THEN
        return;
    END IF;

    SELECT 'ban_steam', ban_id, ban_type, reason, evade_ok, valid_until
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban
    WHERE deleted = false
      AND valid_until > now()
      AND last_ip IS NOT NULL
      AND last_ip::inet <<= ip::inet;

    IF out_ban_id > 0 THEN
        return;
    END IF;

    SELECT 'ban_steam_friend', 1, 2, 15, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM steam_friends
    WHERE friend_id = in_steam_id;

    if out_ban_id > 0 AND NOT is_whitelist_sid then
        return;
    else
        out_ban_id = null;
    end if;

    SELECT 'steam_group', 1, 2, 16, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM steam_group_members
    WHERE steam_id = in_steam_id;

    if out_ban_id > 0 AND NOT is_whitelist_sid then
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

    if out_ban_id > 0 AND NOT (is_whitelist_addr OR is_whitelist_sid) then
        return;
    end if;

    SELECT 'cidr_block', 1, 2, 14, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM cidr_block_entries
    WHERE ip::ip4 <<= net_block;

    if out_ban_id > 0 AND NOT (is_whitelist_addr OR is_whitelist_sid) then
        return;
    end if;

    SELECT 'ban_asn', 1, 2, 17, false, NOW() + (INTERVAL '10 years')
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban_asn
             LEFT JOIN net_asn na on ban_asn.as_num = na.as_num
    WHERE ip::ip4 <<= na.ip_range;

    if out_ban_id > 0 AND NOT (is_whitelist_addr OR is_whitelist_sid) then
        return;
    end if;

END
$func$ LANGUAGE plpgsql;

-- update ban set evade_ok = true where target_id = 76561199533858043;
--
select check_ban('76561199587429942', '1.1.1.6'); -- ip ok, diff ip
select check_ban('76561197963621597', '1.12.36.4'); -- id diff, ip same
select check_ban('76561199093644873', '1.12.36.4'); -- id diff, ip same

COMMIT;
