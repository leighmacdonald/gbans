ALTER TABLE person_notification
  ADD CONSTRAINT person_notification_pkey PRIMARY KEY (person_notification_id);

CREATE INDEX IF NOT EXISTS idx_person_notification_steam_id
  ON person_notification (steam_id, deleted, read, person_notification_id DESC);

CREATE INDEX IF NOT EXISTS idx_person_connections_steam_id
  ON person_connections (steam_id, created_on DESC);

CREATE INDEX IF NOT EXISTS idx_net_location_ip_range
  ON net_location USING GIST (ip_range);

CREATE INDEX IF NOT EXISTS idx_net_proxy_ip_range
  ON net_proxy USING GIST (ip_range);

CREATE INDEX IF NOT EXISTS idx_net_asn_ip_range
  ON net_asn USING GIST (ip_range);

CREATE INDEX IF NOT EXISTS idx_report_message_report_id
  ON report_message (report_id, created_on);

CREATE INDEX IF NOT EXISTS idx_ban_target_deleted_valid
  ON ban (target_id, deleted, valid_until);

CREATE INDEX IF NOT EXISTS idx_match_round_match_id
  ON match_round (match_id);

CREATE INDEX IF NOT EXISTS idx_match_server_id
  ON match (server_id);

CREATE INDEX IF NOT EXISTS idx_match_map_id
  ON match (map_id);

CREATE INDEX IF NOT EXISTS idx_match_demo_id
  ON match (demo_id);

CREATE INDEX IF NOT EXISTS idx_match_round_player_steam_id
  ON match_round_player (steam_id);

CREATE INDEX IF NOT EXISTS idx_match_round_player_variants_steam_id
  ON match_round_player_variants (steam_id);

CREATE INDEX IF NOT EXISTS idx_person_messages_match_id
  ON person_messages (match_id, person_message_id);
