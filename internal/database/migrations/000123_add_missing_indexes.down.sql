DROP INDEX IF EXISTS idx_ban_target_deleted_valid;

DROP INDEX IF EXISTS idx_report_message_report_id;

DROP INDEX IF EXISTS idx_net_asn_ip_range;

DROP INDEX IF EXISTS idx_net_proxy_ip_range;

DROP INDEX IF EXISTS idx_net_location_ip_range;

DROP INDEX IF EXISTS idx_person_connections_steam_id;

DROP INDEX IF EXISTS idx_person_notification_steam_id;
ALTER TABLE person_notification DROP CONSTRAINT IF EXISTS person_notification_pkey;

DROP INDEX IF EXISTS idx_person_messages_match_id;

DROP INDEX IF EXISTS idx_match_round_player_variants_steam_id;

DROP INDEX IF EXISTS idx_match_round_player_steam_id;

DROP INDEX IF EXISTS idx_match_demo_id;
DROP INDEX IF EXISTS idx_match_map_id;
DROP INDEX IF EXISTS idx_match_server_id;

DROP INDEX IF EXISTS idx_match_round_match_id;
