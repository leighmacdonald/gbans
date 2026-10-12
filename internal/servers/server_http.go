package servers

import (
	"net/http"
	"strconv"
)

// RegisterConnectHandler serves game-connect redirects. Discord strips
// steam:// links, so alerts link to https://<host>/connect/<server id> and
// the browser bounces to the game client from here.
//
// The pattern is more specific than the "/connect/" Connect-RPC mount, so
// single-segment GETs land here while RPC procedure calls are unaffected.
func RegisterConnectHandler(mux *http.ServeMux, servers *Servers) {
	mux.HandleFunc("GET /connect/{serverID}", func(res http.ResponseWriter, req *http.Request) {
		id, errParse := strconv.ParseInt(req.PathValue("serverID"), 10, 32)
		if errParse != nil || id <= 0 {
			http.NotFound(res, req)

			return
		}

		server, errServer := servers.Server(req.Context(), int32(id))
		if errServer != nil || server.Deleted {
			http.NotFound(res, req)

			return
		}

		http.Redirect(res, req, server.SteamLink(), http.StatusFound)
	})
}
