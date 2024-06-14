# Stats Overview

Stats are generated for many game events. They are summarized into atomic matches and stores in the database. The
design goal is to have a cross between [logs.tf](https://logs.tf)
and [hlstatsx:ce](https://github.com/A1mDev/hlstatsx-community-edition).

## Compared with hlstatsx:ce

- Simpler deployments, single monolithic binary.
- Considerably better scaling performance
- Matches are committed to the database in a single transaction instead of immediately upon incoming events
- Long term tracking of who killed who is not available currently

## General Info

- Ignores showing stats from players in the match < 60 seconds
- Matches that dont have a minimum amount of players are discarded
- Matches are

## How stats are generated (new, WIP)

1. gbans checks for new demos.
2. If a demo exists, download it locally or stop.
3. Demo is uploaded to the [parsing service](https://github.com/leighmacdonald/tf2_demostats) (avail soon).
4. Results are inserted into the database.