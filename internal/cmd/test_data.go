package cmd

import (
	"context"
	"encoding/json"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"time"
)

var testRconPass = "testing"

// testDataCmd loads the db schema
var testDataCmd = &cobra.Command{
	Use:   "test_data",
	Short: "Add testing data",
	Run: func(cmd *cobra.Command, args []string) {
		store.Init(config.DB.DSN)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()
		p, _ := store.GetOrCreatePersonBySteamID(ctx, steamid.SID64(76561198084134025))
		sum1, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{p.SteamID})
		if err != nil {
			log.Errorf("Failed to get player summary: %v", err)
			return
		}
		p.PermissionLevel = model.PAdmin
		p.PlayerSummary = &sum1[0]
		if err := store.SavePerson(ctx, p); err != nil {
			log.Errorf("Failed to save person: %v", err)
			return
		}
		type BDIds struct {
			FileInfo struct {
				Authors     []string `json:"authors"`
				Description string   `json:"description"`
				Title       string   `json:"title"`
				UpdateURL   string   `json:"update_url"`
			} `json:"file_info"`
			Schema  string `json:"$schema"`
			Players []struct {
				Steamid    int64    `json:"steamid"`
				Attributes []string `json:"attributes"`
				LastSeen   struct {
					PlayerName string `json:"player_name"`
					Time       int    `json:"time"`
				} `json:"last_seen"`
			} `json:"players"`
			Version int `json:"version"`
		}
		resp, err := http.Get("https://tf2bdd.pazer.us/v1/steamids")
		if err != nil {
			log.Fatalf("Could not download ids: %v", err)
		}
		body, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			log.Fatalf("Could not read body of ids: %v", err2)
		}
		var j BDIds
		if err := json.Unmarshal(body, &j); err != nil {
			log.Fatalf("Could not decode ids: %v", err)
		}
		b := []steamid.SID64{
			76561198083950961,
			76561198970645474,
			76561198186070461,
			76561198042277652,
		}
		for i, v := range j.Players {
			b = append(b, steamid.SID64(v.Steamid))
			if i == 125 {
				break
			}
		}
		for _, bid := range b {
			v, err := store.GetOrCreatePersonBySteamID(ctx, bid)
			if err != nil {
				log.Errorf("error creating person: %v", err)
				return
			}
			sum, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{bid})
			if err != nil {
				log.Errorf("Failed to get player summary: %v", err)
				return
			}
			if len(sum) == 0 {
				continue
			}
			v.PlayerSummary = &sum[0]
			if err := store.SavePerson(ctx, v); err != nil {
				log.Warnf("Failed to save person: %v", err)
				continue
			}
		}
		for _, v := range [][]string{
			{"sea-1", "sea-1.us.uncletopia.com"},
			{"lax-1", "lax-1.us.uncletopia.com"},
			{"sfo-1", "sfo-1.us.uncletopia.com"},
			{"dal-1", "dal-1.us.uncletopia.com"},
			{"chi-1", "chi-1.us.uncletopia.com"},
			{"nyc-1", "nyc-1.us.uncletopia.com"},
			{"atl-1", "atl-1.us.uncletopia.com"},
			{"frk-1", "frk-1.de.uncletopia.com"},
			{"ber-1", "ber-1.de.uncletopia.com"},
			{"ham-1", "ham-1.de.uncletopia.com"},
			{"lon-1", "lon-1.uk.uncletopia.com"},
		} {
			s := model.Server{
				ServerName:     v[0],
				Address:        v[1],
				Port:           27015,
				RCON:           testRconPass,
				ReservedSlots:  8,
				Token:          "0123456789012345678901234567890123456789",
				Password:       golib.RandomString(20),
				TokenCreatedOn: config.Now(),
				CreatedOn:      config.Now(),
				UpdatedOn:      config.Now(),
			}
			if err := store.SaveServer(ctx, &s); err != nil {
				log.Errorf("Failed to add server: %v", err)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(testDataCmd)

	testDataCmd.Flags().StringVarP(&testRconPass, "rcon", "r", "testing", "Sets the rcon password for test data")
}
