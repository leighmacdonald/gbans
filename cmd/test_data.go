package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/internal/service"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

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
		b := []steamid.SID64{
			76561198083950961,
			76561198970645474,
			76561198186070461,
			76561198042277652,
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
			v.PlayerSummary = &sum[0]
			if err := service.SavePerson(v); err != nil {
				log.Errorf("Failed to save person: %v", err)
				return
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
		s := model.Server{
			ServerName:     "cg-1",
			Address:        "localhost",
			Port:           27015,
			RCON:           "test",
			Password:       "",
			TokenCreatedOn: config.Now(),
			CreatedOn:      config.Now(),
			UpdatedOn:      config.Now(),
		}
		if err := service.SaveServer(&s); err != nil {
			log.Errorf("Failed to add server: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testDataCmd)
}
