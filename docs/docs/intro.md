---
sidebar_position: 1
---

# Intro

gbans is a system for handling Team Fortress 2 server administration in a centralized manner. It is primarily written
in golang, TypeScript and sourcepawn and consists of several major components:

:::warning

Keep in mind, while we do use this software in production, it's still undergoing pretty heavy development, with a
number of unstable parts and not entirely complete features sets. This means there is effectively no stability
guarentee yet. Version 1.0 is our target for a very complete and more seamless user experience.

:::

- Website / Frontend
- Backend / API
- Sourcemod plugin

## High Level Features

- Several types of [bans](./guide/bans) (steam, ip/cidr, asn, group membership, friends) with whitelisting for ips/steamids.
- [User reports and appeals](./guide/reports)
- [Server browser](./guide/servers)
- [Forums](./guide/forums), [wiki](./guide/wiki) and [news](./guide/news)
- [Chat logs](./guide/chatlogs) with word filtering and automatic action
- [SourceTV](./guide/sourcetv) downloads
- [Player stats](./guide/stats) and [map statistics](./guide/maps), built on
  [tf2_demostats](https://github.com/leighmacdonald/tf2_demostats)
- [MGE](./guide/mge) duel standings
- [Patreon](./guide/patreon) integration (OAuth)
- [Discord](./guide/discord) integration (Bot + OAuth)
- [Notifications](./guide/notifications) (in-app center and in-game match results)
- [Anticheat](./guide/anticheat) detection and automatic action
- [Roles and permissions](./guide/roles) web UI for managing gbans and sourcemod users,
  groups, permissions and overrides.
- [Sourcemod SQL Admins](https://wiki.alliedmods.net/SQL_Admins_(SourceMod)) compatibility. We use the same database
  schema as the base sourcemod sql admins, so you can also use the built in sql-admins* plugins to edits these in game.

Most of these features have some level of being toggled on/off so you can choose what features suits your setup best.

## Installation Guide

To get started with the installation, please see the [installation](./install) guide.

## Development Guide

If you want to hack on gbans, please see the [develelopment](./devel) guide to get started.
