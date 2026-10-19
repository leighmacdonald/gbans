package frontend

import "errors"

var ErrContentRoot = errors.New("failed to open content root")

// jsRoutes defines the SPA routes served index.html for crawler bots and
// browsers hard refreshing a deep link. The patterns use the same wildcard
// syntax as the frontend client routes, ex. /profile/{steam_id}.
//
// nolint:gochecknoglobals
var jsRoutes = []string{
	"/admin/anticheat", "/admin/appeals", "/admin/bans", "/admin/contests", "/admin/filters",
	"/admin/network", "/admin/network/cidrblocks", "/admin/network/ipInfo", "/admin/network/playersbyip",
	"/admin/news", "/admin/people", "/admin/reports", "/admin/roles", "/admin/serverlogs",
	"/admin/servers", "/admin/settings", "/admin/votes", "/ban/{ban_id}", "/changelog",
	"/chatlogs", "/forums", "/forums/{forum_id}", "/forums/thread/{forum_thread_id}", "/login",
	"/login/success", "/logout", "/match/{match_id}", "/matches", "/matches/{steam_id}", "/mge",
	"/mge/1v1", "/mge/2v2", "/notifications", "/page-not-found", "/patreon", "/permission",
	"/privacy-policy", "/profile/{steam_id}", "/report", "/report/{report_id}", "/servers",
	"/settings", "/stats", "/stv", "/wiki", "/wiki/{slug...}",
}
