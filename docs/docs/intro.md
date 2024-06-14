---
sidebar_position: 1
---

# Intro

gbans is a system for handling Team Fortress 2 server administration in a centralized manner. It is primarily written 
in golang, typescript and sourcepawn and consists of 3 major components:

- Website / Frontend
- Backend services
- Sourcemod plugin

## High Level Features

- Several types of bans (steam, ip/cidr, group membership, friends) with whitelisting for ips/steamids.
- User reports and appeals
- Server browser
- Forums
- Wiki
- Chat logs
- SourceTV downloads
- Player Stats
- Patreon integration (OAuth)
- Discord integration (Bot + OAuth)
- Flagging and automatic action for flagged words
- Sourcemod SQL Admins compatibility

Most of these features have some level of being toggled on/off so you can choose what features suits your setup best.

## Installation Guide

To get started with installation, please see the [installation](./install) guide.

## Development Guide

If you want to hack on gbans, please see the [develelopment](./devel) guide to get started.