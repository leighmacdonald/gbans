package frontend

import "errors"

var ErrContentRoot = errors.New("failed to open content root")

// nolint:gochecknoglobals
var jsRoutes = []string{
	"/servers", "/profile/:steam_id", "/bans", "/appeal", "/settings", "/report",
	"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/bans", "/admin/reports", "/admin/news", "/admin/import", "/admin/filters",
	"/404", "/logout", "/login/success", "/report/:report_id", "/wiki", "/wiki/*slug", "/log/:match_id",
	"/logs/:steam_id", "/logs", "/ban/:ban_id", "/chatlogs", "/admin/appeals", "/login", "/pug", "/quickplay",
	"/global_stats", "/stv", "/login/discord", "/notifications", "/admin/network", "/stats",
	"/stats/weapon/:weapon_id", "/stats/player/:steam_id", "/privacy-policy", "/admin/contests",
	"/contests", "/contests/:contest_id", "/forums", "/forums/:forum_id", "/forums/thread/:forum_thread_id",
	"/admin/network/ip_hist", "/admin/network/players_by_ip", "/admin/network/ip_info", "/admin/network/cidr_blocks",
	"/admin/votes",
}
