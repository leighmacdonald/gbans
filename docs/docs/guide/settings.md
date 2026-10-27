# Player Settings

The **Settings** page (`/settings`) is where you manage your account. It is split into
tabs; which tabs appear depends on the features your server has enabled and your
permissions.

## General

Core account options that don't fit into a sub-category:

- **Hide personal stats on profile** — hides your stats from your public profile. You
  can still view your own stats.
- **Notify me in-game when my match results are ready** — sends you an in-game private
  message with a link to your match once it has been imported. See
  [match notifications](notifications.md#match-notifications).
- **Desktop notifications** (where your browser supports them) — request browser
  permission so gbans can raise desktop alerts for your notifications. You can revoke
  this at any time through your browser's site settings.

## Gameplay

The Gameplay tab lists your in-game
[clientprefs](../install/config.md#clientprefs) — the per-player settings your server
exposes through the `sm_cookies` table.

- **Boolean** prefs are three-state: **on**, **off**, or **default** (which removes your
  override and leaves the choice to the server).
- **Number** prefs must be a whole number. Where the server defines a range, the field is
  limited to it (for example `fov_override` is restricted to 20–130).
- **Text** prefs accept free-form values up to 100 characters.
- **Protected** prefs are shown read-only, and **private** prefs are never shown.

Leave a field empty to fall back to the server default. Saved values are written to your
SourceMod cookie cache and applied when you connect.

## Forums

Available when you have the `forum_edit` permission.

- **Your forum signature** — the signature shown alongside your forum posts.
- **Enable people to sign your profile** — lets other players leave a signature/message on
  your profile.

## Connections

Available when Discord and/or Patreon integration is enabled.

- **Discord** — link or unlink your Discord account. Linking maps your Steam identity to
  Discord, which powers bot features such as role management and match summaries (see
  [Discord Reference](discord.md)).
- **Patreon** — connect your Patreon account to be recognised as a supporter (see
  [Patreon](patreon.md)).

Changes are saved per-tab with the **Save** button.
