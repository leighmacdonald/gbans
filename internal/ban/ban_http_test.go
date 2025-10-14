package ban_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/tests"
)

func TestHTTPBan(t *testing.T) {
	bans := ban.NewBans(ban.NewRepository(fixture.Database, fixture.Persons), fixture.Persons, fixture.Config, nil, nil, nil)
	router := fixture.CreateRouter()
	ban.NewHandlerBans(router, bans, fixture.Config, &tests.StaticAuthenticator{
		Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID),
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), "POST", "/api/bans", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	tokens := tests.AuthTokens{}
	tests.TestEndpoint(t, router, "POST", "/api/bans", nil, 200, &tokens)
}
