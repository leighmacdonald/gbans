
CREATE TABLE IF NOT EXISTS mgemod_stats (stats_id serial, rating INTEGER, steamid BIGINT, name TEXT, wins INTEGER, losses INTEGER, lastplayed INTEGER, hitblip INTEGER);
CREATE TABLE IF NOT EXISTS mgemod_duels (duel_id serial, winner BIGINT, loser BIGINT, winnerscore INTEGER, loserscore INTEGER, winlimit INTEGER, gametime INTEGER, mapname TEXT, arenaname TEXT);
CREATE TABLE IF NOT EXISTS mgemod_duels_2v2 (duel2_id serial, winner BIGINT, winner2 BIGINT, loser BIGINT, loser2 BIGINT, winnerscore INTEGER, loserscore INTEGER, winlimit INTEGER, gametime INTEGER, mapname TEXT, arenaname TEXT);
