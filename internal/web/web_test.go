package web_test

// func testHTTPResponse(t *testing.T, r *gin.Engine, req *http.Request, f func(w *httptest.ResponseRecorder) bool) {
//	w := httptest.NewRecorder()
//	r.ServeHTTP(w, req)
//	if !f(w) {
//		t.Fail()
//	}
// }
//
// func testResponse(t *testing.T, unit httpTestUnit, f func(w *httptest.ResponseRecorder) bool) {
//	e := gin.New()
//	web.New()
//	web.SetupRouter(e, logPayloadChan)
//	w := httptest.NewRecorder()
//	e.ServeHTTP(w, unit.r)
//	if !f(w) {
//		t.Fail()
//	}
// }

// func newTestReq(method string, route string, body any, token string) *http.Request {
//	b, _ := json.Marshal(body)
//	req, _ := http.NewRequest(method, route, bytes.NewReader(b))
//	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
//	return req
// }

// type httpTestResult struct {
//	Code int
//	Body any
// }

// type httpTestUnit struct {
//	r *http.Request
//	e httpTestResult
//	m string
// }

// func createToken(sid steamid.SID64, pr model.Privilege) string {
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
//	defer cancel()
//	p, _ := store.GetOrCreatePersonBySteamID(ctx, sid)
//	p.PermissionLevel = pr
//	_ = store.SavePerson(ctx, p)
//	token, _ := web.newUserJWT(p.SteamID)
//	return token
// }
//

// func TestAPICheck(t *testing.T) {
//	e := gin.New()
//	setupRouter(e, logPayloadChan)
//	req := newTestReq("POST", "/api/check", web.CheckRequest{
//		ClientID: 10,
//		SteamID:  string(steamid.SID64ToSID(76561197961279983)),
//		IP:       net.ParseIP("10.10.10.10"),
//	}, "")
//
//	w := httptest.NewRecorder()
//	e.ServeHTTP(w, req)
//	require.Equal(t, http.StatusForbidden, w.Code)
// }

// func TestOnAPIPostBan(t *testing.T) {
//	type req struct {
//		// TODO replace string with SID64 when steam package gets fixed
//		SteamID    string        `json:"steam_id"`
//		Duration   string        `json:"duration"`
//		BanType    model.BanType `json:"ban_type"`
//		Reason     model.Reason  `json:"reason"`
//		ReasonText string        `json:"reason_text"`
//		Network    string        `json:"network"`
//	}
//	token := createToken(76561198044052046, model.PAdmin)
//	s1 := fmt.Sprintf("%d", 76561197960265728+rand.Int63n(100000000))
//	units := []httpTestUnit{
//		{newTestReq("POST", "/api/ban", req{
//			SteamID:    s1,
//			Duration:   "1d",
//			BanType:    model.Banned,
//			Reason:     0,
//			ReasonText: "test",
//			Network:    "",
//		}, token),
//			httpTestResult{Code: http.StatusCreated},
//			"Failed to successfully create steam ban"},
//		{newTestReq("POST", "/api/ban", req{
//			SteamID:    s1,
//			Duration:   "1d",
//			BanType:    model.Banned,
//			Reason:     0,
//			ReasonText: "test",
//			Network:    "",
//		}, token),
//			httpTestResult{Code: http.StatusConflict},
//			"Failed to successfully handle duplicate ban creation"},
//	}
//	testUnits(t, units)
// }
//
// func TestAPIGetServers(t *testing.T) {
//	e := gin.New()
//	web.SetupRouter(e, logPayloadChan)
//	req, _ := http.NewRequest("GET", "/api/servers", nil)
//	testHTTPResponse(t, e, req, func(w *httptest.ResponseRecorder) bool {
//		if w.Code != http.StatusOK {
//			return false
//		}
//		var r web.apiResponse
//		b, errRead := ioutil.ReadAll(w.Body)
//		require.NoError(t, errRead, "Failed to read body")
//		require.NoError(t, json.Unmarshal(b, &r), "Failed to unmarshall body")
//		return true
//	})
// }
//
// func testUnits(t *testing.T, testCases []httpTestUnit) {
//	for _, unit := range testCases {
//		testResponse(t, unit, func(w *httptest.ResponseRecorder) bool {
//			if unit.e.Code > 0 {
//				require.Equal(t, unit.e.Code, w.Code, unit.m)
//				return true
//			}
//			return false
//		})
//	}
// }
//
// func TestAuthMiddleware(t *testing.T) {
//	s := model.Server{
//		ServerNameShort:     golib.RandomString(10),
//		Token:          "",
//		ServerAddress:        "localhost",
//		Port:           27015,
//		RCON:           "password",
//		ReservedSlots:  8,
//		Password:       "",
//		TokenCreatedOn: config.Now(),
//		CreatedOn:      config.Now(),
//		UpdatedOn:      config.Now(),
//	}
//	e := gin.New()
//	web.SetupRouter(e, logPayloadChan)
//	req := newTestReq("POST", "/api/server", s,
//		createToken(76561198084134025, model.PAuthenticated))
//	w := httptest.NewRecorder()
//	e.ServeHTTP(w, req)
//	require.Equal(t, http.StatusForbidden, w.Code)
//
//	reqOK := newTestReq("POST", "/api/server", s,
//		createToken(76561198084134025, model.PAdmin))
//	wOK := httptest.NewRecorder()
//	e.ServeHTTP(wOK, reqOK)
//	require.Equal(t, http.StatusOK, wOK.Code)
// }
//
// func TestWebSocketClient(t *testing.T) {
//	e := gin.New()
//	web.SetupRouter(e, logPayloadChan)
//	s := httptest.NewServer(e)
//	defer s.Close()
//	u := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
//
//	// Start to the server
//	ws, _, errDialer := websocket.DefaultDialer.Dial(u, nil)
//	if errDialer != nil {
//		t.Fatalf("%v", errDialer)
//	}
//	defer ws.Close()
//
//	checkResp := func(t *testing.T, pt web.Type, req any, rt web.Type, res any) {
//		p, errEnc := web.EncodeWSPayload(pt, req)
//		if errEnc != nil {
//			t.FailNow()
//		}
//		if errW := ws.WriteMessage(websocket.TextMessage, p); errW != nil {
//			t.Fatalf("%v", errW)
//		}
//		_, respBytes, errR := ws.ReadMessage()
//		if errR != nil {
//			t.Fatalf("%v", errR)
//		}
//		var resp web.SocketPayload
//		state := int32(web.Closed)
//		require.NoError(t, json.Unmarshal(respBytes, &resp), "Failed to decode response")
//		require.Equal(t, rt, resp.Type, "Got invalid payload type")
//		switch resp.Type {
//		case web.ErrType:
//			var wsErr web.WSErrRes
//			require.NoError(t, json.Unmarshal(resp.Data, &wsErr))
//			require.EqualValues(t, res.(web.WSErrRes), wsErr)
//		case web.AuthFailType:
//			var wsErr web.WSErrRes
//			require.NoError(t, json.Unmarshal(resp.Data, &wsErr))
//			require.EqualValues(t, res.(web.WSErrRes), wsErr)
//		case web.AuthOKType:
//			atomic.SwapInt32(&state, int32(web.Authenticated))
//		}
//	}
//
//	checkResp(t, web.AuthType, web.SocketAuthReq{}, web.AuthFailType, web.WSErrRes{Error: "Auth invalid"})
//
// }
