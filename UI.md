# UI Issues — localhost:6007 Crawl

Crawled 2026-10-12 via orca browser (logged in as admin test user, steamId `76561198084134025`).
Covered: all guest pages, all auth pages, all `/admin/*` pages, `/ban/:id`, `/match/:id`.
Method: accessibility snapshots + DOM eval + console/network logs per page (artifacts in `.tmp/ui/`).
Result: **0 console errors** on every page; no failed HTTP requests except the Patreon 404 below.
Nothing fixed — findings only.

## Bugs / visible defects

### 1. `/patreon` renders the word "undefined" everywhere + broken link
Backend `PatreonService` is not registered (`POST /connect/patreon.v1.PatreonService/PatronCampaigns` → 404) while config has `patreonEnabled: true`. The page then renders:
- Title: `Patreon Campaign: undefined`
- Image `src="undefined"`, summary `undefined`, thanks `undefined`, patron count `undefined`
- "VIEW MEMBERSHIP TIERS" button href = `undefined/membership`
- No error/empty state is shown at all.

`frontend/src/routes/_guest.patreon.tsx:65,92,97,102,142,158` — `String(data?.campaigns[0].attributes?.x)` short-circuits to the string `"undefined"`.
Fix direction: handle the failed/empty query with an error state, and/or register the patreon service in `internal/cmd/app.go`.

### 2. Raw internal error strings shown to end users (systemic)
Submitting `/report` with the required SteamID empty shows a flash banner containing:

```
ConnectError[invalid_argument] target_id: value is required [required], and 1 more violation
```

`sendError` in `frontend/src/routes/__root.tsx:104-117` passes `error.detail` (raw protobuf/connect validation text, incl. proto field names) straight into the flash. Affects every form that reports via `sendError` (report, ban create/edit, wiki, forum, …). Additionally the report form has **no client-side validation** — the empty field only fails after a server round trip.

### 3. Epoch-zero timestamps displayed as real dates
- `/admin/people` → Created column shows `Thu, 01 Jan 1970 00:00:00 GMT` for zero timestamps. `frontend/src/routes/_mod.admin.people.tsx:189-199` — the `if (!value)` guard misses epoch values.
- `/ban/22028` → `Account Age: 1969-12-31 17:00` (zero account-creation time rendered as an age).

### 4. "Permanent" ban shows a concrete expiry
`/ban/22028` is a Permanent ban (list column says `Permanent`) yet the detail page shows `Expires At: 2036-09-28` and `in almost 10 years`. Inconsistent rendering of permanent bans.

### 5. Empty link text in `/admin/bans` player column
`frontend/src/routes/_mod.admin.bans.tsx:253`:
```tsx
{row.original.targetPersonaName ?? row.original.targetId}
```
An **empty-string** persona name renders a link with no text (unnamed link in the a11y tree). `??` doesn't catch `""`. The correct pattern already exists in `frontend/src/component/PersonCell.tsx:210` (`emptyOrNullString`).

### 6. Copy-paste bug: persona-name fallback is `serverId`
`frontend/src/routes/_mod.admin.anticheat.tsx:109`:
```tsx
{row.original.personaName ?? row.original.serverId}
```
Falls back to the server ID instead of the steam ID.

### 7. Mojibake in external ban history
`/ban/22028` → "Reporter External Ban History" Name cell shows `Â·Æ'Â· Captain Blades` — double-encoded UTF-8 (bytes decoded as Latin-1) from external source `skial`.

### 8. Wiki pages render raw HTML as literal text
`/wiki/Settings` and `/wiki/Events` display literal `<b><u>…</b></u>` / `<br />` tags. `frontend/src/component/MarkdownRenderer.tsx:52` sets `disableParsingRawHTML: true` while stored wiki content contains HTML.

## Accessibility

Icon-only controls without accessible names (confirmed via a11y-tree sweep of all 47 page snapshots):

| Location | Elements |
|---|---|
| Top bar — **every page** | Notifications icon link, no `aria-label` — `frontend/src/component/TopBar.tsx:452` (`IconButton component={RouterLink}` + `MailIcon` only; the BLU/RED toggle next to it has a Tooltip, this one doesn't) |
| Top bar — admin, **every page** | Admin dropdown trigger `label: <SettingsIcon />` — `TopBar.tsx:321` (icon-only, no accessible name) |
| `/admin/news` | Add + per-row Delete/Edit `IconButton`s — `_mod.admin.news.tsx:252,214,223` |
| `/admin/network/*` | Add + per-row Delete/Edit — `component/table/NetworkBlocklist.tsx:162,134,143` |
| `/admin/network/ip-whitelist` | Add + per-row Edit/Delete — `component/table/IPWhitelistTable.tsx:176,148,157` |
| `/admin/network/steam-whitelist` | Delete + Add — `component/table/SteamWhitelistTable.tsx:119,138` |
| `/admin/bans` | 2 unnamed row-action buttons per row |
| `/admin/roles`, `/admin/filters`, `/admin/contests` | Unnamed per-row action buttons |
| `/report` | Unnamed icon link per report row |
| `/report`, `/ban/:id` | Quill editor toolbar buttons expose as `radio` role (minor) |

Fix direction: add `aria-label` (and/or `Tooltip`) to every icon-only `IconButton`/link — many are one-line changes.

## Copy / minor

- Typo: "Enabled/disable the forums functionality." — `frontend/src/routes/_admin.admin.settings.tsx:451`.
- PersonCell avatar fallback is a hardcoded `"P"` — `frontend/src/component/PersonCell.tsx:194` — while the topbar avatar falls back to the first letter of the name. Inconsistent fallback.

## Verified OK

- Theme toggle (BLU/RED) works and persists; news pagination works; all nav/footer hrefs correct.
- 404 page, empty states (match detail "No records to display", tables) render properly.
- No horizontal page overflow at 1183px (tables scroll inside their containers).
- `/forums` correctly redirects home when disabled.

## Environment-only (not code bugs)

- No internet in this env → steamstatic.com avatars fail; all MUI avatar fallbacks show (topbar "u", PersonCell "P").
- Vite dev 504s during first dependency optimization are transient/self-healing.
- `/servers` shows identical `9810km` distances for all servers (user location unknown fallback).
- `/match/:id` shows all-zero stats for a match with demo data (stats not processed — data issue, UI renders correctly).

## Limitations

- Orca CLI has no viewport/device emulation → **mobile/responsive breakpoints could not be verified**.
- Only the logged-in (admin) session was tested; logged-out flows (Steam login round-trip, guest topbar) were not exercised.
