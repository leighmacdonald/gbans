package service

import (
	"bytes"
	"encoding/json"
	"github.com/leighmacdonald/gbans/model"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testResponse(t *testing.T, unit httpTestUnit, f func(w *httptest.ResponseRecorder) bool) {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, unit.r)
	if !f(w) {
		t.Fail()
	}
}

func newTestReq(method string, key routeKey, body interface{}) *http.Request {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, routeRaw(string(key)), bytes.NewReader(b))
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
	s1 := "76561198031215761"
	units := []httpTestUnit{
		{newTestReq("POST", routeAPIBans, req{
			SteamID:    s1,
			Duration:   "1d",
			BanType:    model.Banned,
			Reason:     0,
			ReasonText: "test",
			Network:    "",
		}),
			httpTestResult{Code: http.StatusCreated},
			"Failed to successfully create steam ban"},
		{newTestReq("POST", routeAPIBans, req{
			SteamID:    s1,
			Duration:   "1d",
			BanType:    model.Banned,
			Reason:     0,
			ReasonText: "test",
			Network:    "",
		}),
			httpTestResult{Code: http.StatusConflict},
			"Failed to successfully handle duplicate ban creation"},
	}
	for _, unit := range units {
		testResponse(t, unit, func(w *httptest.ResponseRecorder) bool {
			if unit.e.Code > 0 {
				assert.Equal(t, unit.e.Code, w.Code, unit.m)
				return false
			}
			return true
		})
	}
}
