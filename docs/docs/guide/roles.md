# Roles and Permissions

gbans has two related permission systems:

1. **gbans roles** — web-app roles that control what a player can do on the website (view
   bans, moderate reports, manage servers, and so on).
2. **SourceMod roles** — the standard SourceMod groups/admins/overrides, stored in the
   SQL admins schema and synced to your servers.

## gbans roles

**Admin → Roles** ("Roles & Permissions") lets you create roles, edit the set of
permissions each role grants, assign roles to players, and see who has each role.
Permissions gate both the navigation (which pages a user sees) and the API (which actions
a user is allowed to perform).

## SourceMod overrides

The same page surfaces SourceMod **Command Overrides** and **Group Immunity**:

- **Command Overrides** grant or deny specific SourceMod commands to players or groups.
- **Group Immunity** controls which groups can administer over other groups.

SourceMod **groups** and **admins** live in the standard SQL admins schema (see the
[SourceMod plugin](sourcemod.md) page for the schema and the in-game `gb_admins` command),
so they can also be edited with the built-in `sql-admins` plugin or any other tool that
speaks the same schema.

## People

**Admin → People** is a searchable index of every player gbans has seen, with their
SteamID, visibility state, VAC and community ban status, and first-seen date. You can
filter by VAC bans, game bans, or community-ban status.
