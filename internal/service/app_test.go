package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func testHTTPResponse(t *testing.T, r *gin.Engine, req *http.Request, f func(w *httptest.ResponseRecorder) bool) {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !f(w) {
		t.Fail()
	}
}

func TestAddWarning(t *testing.T) {
	addWarning(76561197961279983, warnLanguage)
	addWarning(76561197961279983, warnLanguage)
	require.True(t, len(warnings[76561197961279983]) == 2)
}

func GenTestData() {
	servers := []model.Server{
		{
			ServerName: "test-1", Token: golib.RandomString(40), Address: "127.0.0.1", Port: 27015,
			RCON: "test", ReservedSlots: 0, Password: golib.RandomString(20),
			TokenCreatedOn: config.Now(), CreatedOn: config.Now(), UpdatedOn: config.Now(),
		},
		{
			ServerName: "test-2", Token: golib.RandomString(40), Address: "127.0.0.1", Port: 27025,
			RCON: "test", ReservedSlots: 4, Password: golib.RandomString(20),
			TokenCreatedOn: config.Now(), CreatedOn: config.Now(), UpdatedOn: config.Now(),
		},
	}
	for _, server := range servers {
		if err := SaveServer(&server); err != nil {
			log.Fatalf("Failed to setup test server: %v", err)
		}
	}

	filteredWords := []string{"frick", "heck"}
	for _, fw := range filteredWords {
		if err := saveFilteredWord(fw); err != nil {
			log.Fatalf("Failed to setup test filtered words: %v", err)
		}
	}
	steamIds := []steamid.SID64{76561198072115209, 76561197961279983, 76561197992870439, 76561198003911389}
	for i, sid := range steamIds {
		sum, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{sid})
		if err != nil {
			log.Fatalf("Failed to get player summary: %v", err)
		}
		p, err := GetOrCreatePersonBySteamID(sid)
		if err != nil {
			log.Fatalf("Failed to get person: %v", err)
		}
		s := sum[0]
		p.SteamID = sid
		p.IPAddr = fmt.Sprintf("24.56.78.%d", i+1)
		p.PlayerSummary = &s
		if err := SavePerson(p); err != nil {
			log.Fatalf("Failed to save test person: %v", err)
		}
	}

	if _, err := BanPlayer(context.Background(), steamIds[0], config.General.Owner, time.Minute*30,
		model.Cheating, "Aimbot", model.System); err != nil {
		log.Fatalf("Failed to create test ban #1: %v", err)
	}
	if _, err := BanPlayer(context.Background(), steamIds[1], config.General.Owner, 0,
		model.Cheating, "Aimbot", model.System); err != nil {
		log.Fatalf("Failed to create test ban #2: %v", err)
	}

	for i, cidr := range []string{"50.50.50.0/24", "60.60.60.60/32"} {
		_, mask, _ := net.ParseCIDR(cidr)
		if err := saveBanNet(&model.BanNet{
			CIDR:       mask,
			Source:     0,
			Reason:     "",
			CreatedOn:  config.Now().AddDate(0, -(i + 1), 0),
			UpdatedOn:  config.Now().AddDate(0, -(i + 1), 0),
			ValidUntil: config.DefaultExpiration(),
		}); err != nil {
			log.Fatalf("Failed to generate test ban_net #%d: %v", i, err)
		}
	}
}

func clearDB() {
	ctx := context.Background()
	for _, table := range tableList {
		q := fmt.Sprintf(`drop table if exists %s cascade;`, table)
		if _, err := db.Exec(ctx, q); err != nil {
			log.Panicf("Failed to prep database: %s", err.Error())
		}
	}
}

func TestMain(m *testing.M) {
	config.Read()
	config.General.Mode = "test"
	initStore()
	clearDB()
	if err := Migrate(true); err != nil {
		log.Fatal(err)
	}
	defer clearDB()
	initRouter()
	GenTestData()
	os.Exit(m.Run())
}
