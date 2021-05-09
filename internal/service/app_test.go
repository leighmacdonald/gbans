package service

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"math/rand"
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
			ServerName: golib.RandomString(8), Token: golib.RandomString(40), Address: "127.0.0.1", Port: 27015,
			RCON: "test", ReservedSlots: 0, Password: golib.RandomString(20),
			TokenCreatedOn: config.Now(), CreatedOn: config.Now(), UpdatedOn: config.Now(),
		},
		{
			ServerName: golib.RandomString(8), Token: golib.RandomString(40), Address: "127.0.0.1", Port: 27025,
			RCON: "test", ReservedSlots: 4, Password: golib.RandomString(20),
			TokenCreatedOn: config.Now(), CreatedOn: config.Now(), UpdatedOn: config.Now(),
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	for _, server := range servers {
		if err := SaveServer(ctx, &server); err != nil {
			log.Fatalf("Failed to setup test server: %v", err)
		}
	}

	filteredWords := []string{"frick", "heck"}
	for _, fw := range filteredWords {
		if err := saveFilteredWord(ctx, fw); err != nil && !errors.Is(err, errDuplicate) {
			log.Fatalf("Failed to setup test filtered words: %v", err)
		}
	}
	steamIds := []steamid.SID64{76561198072115209, 76561197961279983, 76561197992870439, 76561198003911389}
	for i, sid := range steamIds {
		sum, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{sid})
		if err != nil {
			log.Fatalf("Failed to get player summary: %v", err)
		}
		p, err := GetOrCreatePersonBySteamID(ctx, sid)
		if err != nil {
			log.Fatalf("Failed to get person: %v", err)
		}
		s := sum[0]
		p.SteamID = sid
		p.IPAddr = net.ParseIP(fmt.Sprintf("24.56.78.%d", i+1))
		p.PlayerSummary = &s
		if err := SavePerson(ctx, p); err != nil {
			log.Fatalf("Failed to save test person: %v", err)
		}
	}

	if _, err := BanPlayer(context.Background(), steamIds[0], config.General.Owner, time.Minute*30,
		model.Cheating, "Aimbot", model.System); err != nil && err != errDuplicate {
		log.Fatalf("Failed to create test ban #1: %v", err)
	}
	if _, err := BanPlayer(context.Background(), steamIds[1], config.General.Owner, 0,
		model.Cheating, "Aimbot", model.System); err != nil && err != errDuplicate {
		log.Fatalf("Failed to create test ban #2: %v", err)
	}
	randSN := func() int {
		i := rand.Intn(255)
		if i <= 0 {
			return 1
		}
		return i
	}
	randIP := func() string {
		return fmt.Sprintf("%d.%d.%d.%d", randSN(), randSN(), randSN(), randSN())
	}
	randCidr := fmt.Sprintf("%d.%d.%d.0/24", randSN(), randSN(), randSN())
	for i, cidr := range []string{randIP() + "/32", randIP() + "/32", randCidr} {
		ip, mask, _ := net.ParseCIDR(cidr)
		log.Println(ip)
		if err := saveBanNet(ctx, &model.BanNet{
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

func TestMain(m *testing.M) {
	config.Read()
	config.General.Mode = "test"
	initStore()
	initRouter()
	GenTestData()
	os.Exit(m.Run())
}
