package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testResponse(t *testing.T, unit httpTestUnit, f func(w *httptest.ResponseRecorder) bool) {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, unit.r)
	if !f(w) {
		t.Fail()
	}
}

func newTestReq(method string, route string, body interface{}, token string) *http.Request {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, route, bytes.NewReader(b))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return req
}

type httpTestResult struct {
	Code int
	Body interface{}
}

type httpTestUnit struct {
	r *http.Request
	e httpTestResult
	m string
}

func createToken(sid steamid.SID64, pr model.Privilege) string {
	p, _ := GetOrCreatePersonBySteamID(sid)
	p.PermissionLevel = pr
	_ = SavePerson(p)
	token, _ := newJWT(p.SteamID)
	return token
}

func TestOnAPIPostBan(t *testing.T) {
	type req struct {
		// TODO replace string with SID64 when steam package gets fixed
		SteamID    string        `json:"steam_id"`
		Duration   string        `json:"duration"`
		BanType    model.BanType `json:"ban_type"`
		Reason     model.Reason  `json:"reason"`
		ReasonText string        `json:"reason_text"`
		Network    string        `json:"network"`
	}
	token := createToken(76561198084134025, model.PAdmin)
	s1 := fmt.Sprintf("%d", 76561197960265728+rand.Int63n(100000000))
	units := []httpTestUnit{
		{newTestReq("POST", "/api/ban", req{
			SteamID:    s1,
			Duration:   "1d",
			BanType:    model.Banned,
			Reason:     0,
			ReasonText: "test",
			Network:    "",
		}, token),
			httpTestResult{Code: http.StatusCreated},
			"Failed to successfully create steam ban"},
		{newTestReq("POST", "/api/ban", req{
			SteamID:    s1,
			Duration:   "1d",
			BanType:    model.Banned,
			Reason:     0,
			ReasonText: "test",
			Network:    "",
		}, token),
			httpTestResult{Code: http.StatusConflict},
			"Failed to successfully handle duplicate ban creation"},
	}
	testUnits(t, units)
}

func TestAPIGetServers(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/servers", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		if w.Code != http.StatusOK {
			return false
		}
		var r apiResponse
		b, err := ioutil.ReadAll(w.Body)
		require.NoError(t, err, "Failed to read body")
		require.NoError(t, json.Unmarshal(b, &r), "Failed to unmarshall body")
		return true
	})
}

func TestSteamWebAPI(t *testing.T) {
	if config.General.SteamKey == "" {
		t.Skip("No steamkey set")
		return
	}
	friends, err := fetchFriends(76561197961279983)
	require.NoError(t, err)
	require.True(t, len(friends) > 100)
	summaries, err := fetchSummaries(friends)
	require.NoError(t, err)
	require.Equal(t, len(friends), len(summaries))
}

func TestOnPostLogMessage(t *testing.T) {
	const exampleLog = `L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")
L 02/21/2021 - 06:22:23: server_cvar: "sm_nextmap" "pl_frontier_final"
L 02/21/2021 - 06:22:24: rcon from "23.239.22.163:42004": command "status"
L 02/21/2021 - 06:22:31: "Hacksaw<12><[U:1:68745073]><>" entered the game
L 02/21/2021 - 06:22:35: "Hacksaw<12><[U:1:68745073]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:22:36: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"
L 02/21/2021 - 06:23:04: "Dzefersons14<8><[U:1:1080653073]><Blue>" committed suicide with "world" (attacker_position "-1189 2513 -423")
L 02/21/2021 - 06:23:11: World triggered "Round_Start"
L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "0")
L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")
L 02/21/2021 - 06:23:44: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-476 154 -254") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")
L 02/21/2021 - 06:24:14: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313") 
L 02/21/2021 - 06:24:22: "amogus gaming<13><[U:1:1089803558]><>" connected, address "139.47.95.130:47949"
L 02/21/2021 - 06:24:23: "amogus gaming<13><[U:1:1089803558]><>" STEAM USERID validated
L 02/21/2021 - 06:26:33: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "2 -579 -255")
L 02/21/2021 - 06:30:45: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1074 -2279 -423")
L 02/21/2021 - 06:32:00: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")
L 02/21/2021 - 06:32:30: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")
L 02/21/2021 - 06:29:49: World triggered "Round_Win" (winner "Red")
L 02/21/2021 - 06:29:49: World triggered "Round_Length" (seconds "398.10")
L 02/21/2021 - 06:29:49: Team "Red" current score "1" with "2" players
L 02/21/2021 - 06:29:57: "Hacksaw<12><[U:1:68745073]><Red>" say "gg"
L 02/21/2021 - 06:29:59: "Desmos Calculator<10><[U:1:1132396177]><Red>" say_team "gg"
L 02/21/2021 - 06:33:41: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "domination" against "Dzefersons14<8><[U:1:1080653073]><Blue>"
L 02/21/2021 - 06:33:43: "Cybermorphic<15><[U:1:901503117]><Unassigned>" disconnected (reason "Disconnect by user.")
L 02/21/2021 - 06:35:37: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "revenge" against "Desmos Calculator<10><[U:1:1132396177]><Red>"
L 02/21/2021 - 06:37:20: World triggered "Round_Overtime"
L 02/21/2021 - 06:40:19: "potato<16><[U:1:385661040]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-163 324 -272")
L 02/21/2021 - 06:42:13: World triggered "Game_Over" reason "Reached Win Limit"
L 02/21/2021 - 06:42:13: Team "Red" final score "2" with "3" players
L 02/21/2021 - 06:42:13: Team "RED" triggered "Intermission_Win_Limit"
L 02/21/2021 - 06:42:33: [META] Loaded 0 plugins (1 already loaded)
L 02/21/2021 - 06:42:33: Log file closed.`
	var units []httpTestUnit

	ctx, cancel := context.WithCancel(gCtx)
	defer cancel()
	go logReader(ctx, logRawQueue)
	token := createToken(76561198084134025, model.PAdmin)
	for _, tc := range strings.Split(exampleLog, "\n") {
		units = append(units, httpTestUnit{
			newTestReq("POST", "/api/log", []LogPayload{{
				ServerName: "test-1",
				Message:    tc,
			}}, token),
			httpTestResult{Code: http.StatusCreated},
			"Failed to add log message",
		})
	}
	config.General.Mode = "test"
	testUnits(t, units)
}

func testUnits(t *testing.T, testCases []httpTestUnit) {
	for _, unit := range testCases {
		testResponse(t, unit, func(w *httptest.ResponseRecorder) bool {
			if unit.e.Code > 0 {
				require.Equal(t, unit.e.Code, w.Code, unit.m)
				return true
			}
			return false
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	s := model.Server{
		ServerName:     golib.RandomString(10),
		Token:          "",
		Address:        "localhost",
		Port:           27015,
		RCON:           "password",
		ReservedSlots:  8,
		Password:       "",
		TokenCreatedOn: config.Now(),
		CreatedOn:      config.Now(),
		UpdatedOn:      config.Now(),
	}

	req := newTestReq("POST", "/api/server", s,
		createToken(76561198084134025, model.PAuthenticated))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)

	reqOK := newTestReq("POST", "/api/server", s,
		createToken(76561198084134025, model.PAdmin))
	wOK := httptest.NewRecorder()
	router.ServeHTTP(wOK, reqOK)
	require.Equal(t, http.StatusOK, wOK.Code)
}
