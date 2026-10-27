# Reports and Appeals

## Player reports

Players can report another player from **Report** (`/report`). A report captures:

- The reported **SteamID** (pre-filled when reporting directly from the chat logs).
- A **reason** (cheating, language, custom, …). Choosing a custom reason unlocks the
  free-text field.
- An optional **demo ID** and **demo tick** pointing at the evidence.
- A free-text **description**.

Moderators triage reports from **Admin → Reports**, where they can act on them (for
example issue a ban) or close them.

## Ban appeals

A banned player can appeal their ban. The **Appeal** item (or the ban page
`/ban/<id>`) shows the ban details alongside an appeal message thread. The player starts
an appeal by replying on the thread; staff respond in the same thread and can change the
ban's state.

Appeals are managed from **Admin → Ban Appeals**. Only Steam-based bans can be appealed;
IP/CIDR, ASN and group bans are treated as permanent.

:::note
When [anonymized moderators](moderation.md) is enabled, staff replies on reports and
appeals are shown to players under a fake identity.
:::
