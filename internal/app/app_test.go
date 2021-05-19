package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

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
		if err := store.SaveServer(ctx, &server); err != nil {
			log.Fatalf("Failed to setup test Server: %v", err)
		}
	}
	filteredWords := []string{"frick", "heck"}
	for _, fw := range filteredWords {
		if _, err := store.InsertFilter(ctx, fw); err != nil && !errors.Is(err, store.ErrDuplicate) {
			log.Fatalf("Failed to setup test filtered words: %v", err)
		}
	}
	steamIds := []steamid.SID64{76561198072115209, 76561197961279983, 76561197992870439, 76561198003911389}
	for i, sid := range steamIds {
		sum, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{sid})
		if err != nil {
			log.Fatalf("Failed to get Player summary: %v", err)
		}
		p, err := store.GetOrCreatePersonBySteamID(ctx, sid)
		if err != nil {
			log.Fatalf("Failed to get person: %v", err)
		}
		s := sum[0]
		p.SteamID = sid
		p.IPAddr = net.ParseIP(fmt.Sprintf("24.56.78.%d", i+1))
		p.PlayerSummary = &s
		if err := store.SavePerson(ctx, p); err != nil {
			log.Fatalf("Failed to save test person: %v", err)
		}
	}

	if _, err := ban(context.Background(), action.BanRequest{
		Target:   action.Target(steamIds[0].String()),
		Source:   action.Source(config.General.Owner.String()),
		Duration: "30m",
		Reason:   "Aimbot",
	}); err != nil && err != store.ErrDuplicate {
		log.Fatalf("Failed to create test ban #1: %v", err)
	}

	if _, err := ban(context.Background(), action.BanRequest{
		Target:   action.Target(steamIds[1].String()),
		Source:   action.Source(config.General.Owner.String()),
		Duration: "0",
		Reason:   "Aimbot",
	}); err != nil && err != store.ErrDuplicate {
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
		if err := store.SaveBanNet(ctx, &model.BanNet{
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
