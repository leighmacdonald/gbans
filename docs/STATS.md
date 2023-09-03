# Stats Overview

Stats are generated for many game events. They are summarized into atomic matches and stores in the database. The
design goal is to have a cross between [logs.tf](https://logs.tf)
and [hlstatsx:ce](https://github.com/A1mDev/hlstatsx-community-edition).

## Compared with hlstatsx:ce

- Simpler deployments, single monolithic binary.
- Considerably better scaling performance:
    - Stats are summarized in real time in memory into a `Match` instance
    - Matches are committed to the database in a single transaction instead of immediately upon incoming events
- Long term tracking of who killed who is not available currently

## General Info

- Ignores showing stats from players in the match < 60 seconds
- Matches that dont have a minimum amount of players are discarded
- Matches are

## Incoming Events Flow

1. gbans udp log listener starts
2. Incoming logs get parsed with the [logparse.LogParser](pkg/logparse/log_parser.go) pkg.
3. If a match does not exist, create a new [logparse.Match](pkg/logparse/match.go) to feed parsed events into for
   summarizing
4. Get game over event, save match to database.
5. Broadcast match result summary to discord
