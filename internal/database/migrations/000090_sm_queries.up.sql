CREATE TABLE cidr_block_entries (
    cidr_block_entries_id bigserial primary key,
    cidr_block_source_id int NOT NULL REFERENCES cidr_block_source (cidr_block_source_id) ON DELETE CASCADE,
    net_block cidr not null,
    created_on timestamp not null
);

BEGIN;
-- select steam_to_steam64('STEAM_0:1:583502767'); -- -> 76561199127271263
CREATE OR REPLACE FUNCTION steam_to_steam64(steam2 text) RETURNS bigint as
$$
DECLARE
    parts text[];
BEGIN
    parts := regexp_matches(steam2, '^STEAM_([0-5]):([0-1]):([0-9]+)$');
    return (cast(parts[3] as bigint) * 2) + 76561197960265728 + cast(parts[2] as bigint);
END;
$$
    LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION check_ban(steam text, ip text,
                                     OUT out_ban_source text,
                                     OUT out_ban_id int,
                                     OUT out_reason int,
                                     OUT out_evade_ok bool,
                                     OUT out_valid_until timestamp,
                                     OUT out_ban_type int) AS
$$
BEGIN
    SELECT 'steam', ban_id, ban_type, reason, evade_ok, valid_until
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban
    WHERE deleted = false
      AND target_id = steam_to_steam64(steam)
      AND valid_until > now();

    if out_ban_id > 0 then
        return;
    end if;

    SELECT 'ip', net_id, 2, reason, false, valid_until
    INTO out_ban_source, out_ban_id, out_ban_type, out_reason, out_evade_ok, out_valid_until
    FROM ban_net
    WHERE deleted = false
      AND ip::cidr <<= cidr
      AND valid_until > now();

END
$$
    LANGUAGE plpgsql;

-- SELECT * from check_ban('STEAM_1:1:566689572', '1.2.3.4');

COMMIT;
