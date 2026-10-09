-- Replace the kill-only match_kill table with a generic match_event table
-- storing every MatchEvent exported by tf2_demostats v0.3.3, not just kills.
-- Events carrying positioning information keep geometry(PointZ) columns in
-- game-local Cartesian coordinates, matching the previous kill table.
BEGIN;

DROP TABLE IF EXISTS match_kill;

CREATE TABLE IF NOT EXISTS match_event (
  match_event_id bigserial PRIMARY KEY,
  match_id uuid NOT NULL REFERENCES match (match_id) ON UPDATE CASCADE ON DELETE CASCADE,
  round_id integer REFERENCES match_round (round_id) ON UPDATE CASCADE ON DELETE CASCADE,
  tick integer NOT NULL CHECK (tick >= 0),
  event_type text NOT NULL CHECK (event_type IN (
    'kill', 'capture_started', 'capture', 'capture_blocked', 'capture_broken',
    'building_built', 'building_destroyed', 'building_upgraded', 'building_carried',
    'building_dropped', 'building_removed', 'building_detonated', 'sapper_placed',
    'round_started', 'round_won', 'stalemate', 'game_over',
    'sudden_death_begin', 'sudden_death_end', 'overtime_begin', 'overtime_end',
    'setup_finished', 'uber_dropped', 'uber_deployed', 'flag_event',
    'flag_captured', 'killstreak_ended'
  )),
  actor_steam_id bigint REFERENCES person (steam_id),
  target_steam_id bigint REFERENCES person (steam_id),
  assister_steam_id bigint REFERENCES person (steam_id),
  weapon text,
  building text CHECK (building IN ('sentry', 'dispenser', 'teleporter', 'sapper')),
  killer_position geometry(PointZ),
  victim_position geometry(PointZ),
  event_position geometry(PointZ),
  details jsonb NOT NULL DEFAULT '{}'
);

COMMENT ON COLUMN match_event.round_id IS 'Owning round when the event tick falls within a parsed round interval; NULL when unknown.';
COMMENT ON COLUMN match_event.actor_steam_id IS 'Primary actor (killer, blocker, spy, medic, engineer); NULL when absent or unresolvable.';
COMMENT ON COLUMN match_event.target_steam_id IS 'Primary target (victim, uber target, flag carrier); NULL when absent or unresolvable.';
COMMENT ON COLUMN match_event.assister_steam_id IS 'Assisting player, currently only populated by building_destroyed events.';
COMMENT ON COLUMN match_event.killer_position IS 'Game-local Cartesian coordinates in X, Y, Z order for kill events; NULL otherwise.';
COMMENT ON COLUMN match_event.victim_position IS 'Game-local Cartesian coordinates in X, Y, Z order for kill events; NULL otherwise.';
COMMENT ON COLUMN match_event.event_position IS 'Game-local Cartesian coordinates in X, Y, Z order for positioned non-kill events (building built/destroyed); NULL otherwise.';
COMMENT ON COLUMN match_event.details IS 'Full variant-specific payload (control points, cappers, flags, angles, streaks, etc.) as JSON.';

CREATE INDEX IF NOT EXISTS idx_match_event_match_id ON match_event (match_id, tick);
CREATE INDEX IF NOT EXISTS idx_match_event_match_type ON match_event (match_id, event_type);
CREATE INDEX IF NOT EXISTS idx_match_event_round_id ON match_event (round_id);
CREATE INDEX IF NOT EXISTS idx_match_event_killer_position ON match_event USING GIST (killer_position);
CREATE INDEX IF NOT EXISTS idx_match_event_victim_position ON match_event USING GIST (victim_position);
CREATE INDEX IF NOT EXISTS idx_match_event_event_position ON match_event USING GIST (event_position);

COMMIT;
