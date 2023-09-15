# TODO

This is just a living document on things that could/will get implemented.

If you are looking for things to try implementing, this is a good place to look.

- Give players N time to reconnect before losing slots

# Core

## Moderation/Security

- Generate random user info for appeals messages to conceal moderator identity for preventing harassment
- Alert for N player connecting from the same IP
- Alert for connecting from an IP which a banned player uses (alt)
- Save stac logs to database, tied to matches/demos.
    - Show in reports/appeals if they exist.
- (Maybe) Rate limiting. It's currently largely handled by frontend proxies though.
- Finish networking query utilities in the admin area
- Make inteface to query sourcebans data (using [bd-api](https://github.com/leighmacdonald/bd-api))
- Add player search for mods (maybe users?)
-

## Demos

- Attach the demo to the match_id, probably using a match uuid returned from match start call.
- Add S3 compatible (minio) storage backend
- Permanently save demos which trigger AC or are attached to reports

## Stats

Stats are currently calculated either close to real time (post match) or real time depending on the api (player
profile).
Long term this will likely become more problematic performance wise and we will want to start to investigate storing
daily summaries to calculate from.

- Make sure spec class stats are no longer possible
- When score is tied and map is payload (and maybe ad?) use the stopwatch rules for determining winner
- Make win rate only count when player was in game for > N minutes & > N players
- Trigger match start / end from source mod instead of the current guesswork
- Add more specific filters for things such as
    - Per Server
    - Per Region
    - Temporal (daily/weekly/monthly/yearly/alltime)
- (Maybe) Awards similar to hlstatx
- Nice looking weapon/kill icons (over 250...)
- Nice graphs
- Calculate and compare player stat changes / trends over days/weeks/etc.
- Killstreak stats
- User messages stats

## Forums

We find that while discord is nice for real time communication, it leaves a lot to be desired as far as
giving out information to users in a more long term fashion. Discord threads have quite poor ux/discoverability
meaning there is far too many repeat questions or counter to that, a difficulty for mods to actually engage in the
ones that are more important. The real time nature of it can also encourage people to be more confrontational about
answering questions. eg: "Why do you disable crits? i like them" Which can turn into less than desirable
discourse depending on the community.

- Threads
- Categories
- Search

# Frontend

## General

- Customizable favicons
- Customizable country flags (or include them all)
- (Maybe) Support ESBuild ?
- Migrate remaining DataTables to `LazyTable`
- More components to ensure better ui consistency.

## System Alerts

- Use LazyTable
- Add basic management functionality (delete/sort/search)

# Long term possibilities

- Support for more srcds based games 
