package cmd

import (
	"context"
	"encoding/json"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/service"
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
		service.Init(config.DB.DSN)
		p, _ := service.GetOrCreatePersonBySteamID(steamid.SID64(76561198084134025))
		sum1, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{p.SteamID})
		if err != nil {
			log.Errorf("Failed to get player summary: %v", err)
			return
		}
		p.PlayerSummary = &sum1[0]
		if err := service.SavePerson(p); err != nil {
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
		c := context.Background()
		for i, bid := range b {
			v, err := service.GetOrCreatePersonBySteamID(bid)
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
			if err := service.SavePerson(v); err != nil {
				log.Warnf("Failed to save person: %v", err)
				continue
			}
			var t time.Duration
			if i%2 == 0 {
				t = 0
			} else {
				t = time.Hour * 24
			}
			if _, err := service.BanPlayer(c, v.SteamID, p.SteamID, t, model.Cheating, "Cheater", model.System); err != nil {
				log.Errorf(err.Error())
			}
		}
		for _, v := range [][]string{
			{"lo-1", "192.168.0.101"},
			{"us-1", "us1.uncledane.com"},
			{"us-2", "us2.uncledane.com"},
			{"us-3", "us3.uncledane.com"},
			{"us-4", "us4.uncledane.com"},
			{"us-5", "us5.uncledane.com"},
			{"us-6", "us6.uncledane.com"},
			{"eu-1", "eu1.uncledane.com"},
			{"eu-2", "eu2.uncledane.com"},
			{"au-1", "au1.uncledane.com"},
		} {
			s := model.Server{
				ServerName:     v[0],
				Address:        v[1],
				Port:           27015,
				RCON:           testRconPass,
				Token:          "0123456789012345678901234567890123456789",
				Password:       golib.RandomString(20),
				TokenCreatedOn: config.Now(),
				CreatedOn:      config.Now(),
				UpdatedOn:      config.Now(),
			}
			if err := service.SaveServer(&s); err != nil {
				log.Errorf("Failed to add server: %v", err)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(testDataCmd)

	testDataCmd.Flags().StringVarP(&testRconPass, "rcon", "r", "testing", "Sets the rcon password for test data")
}
