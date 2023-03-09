package cmd

import (
	"context"
	"encoding/json"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var testRconPass = "testing"
var seedFile = "seed.json"

type seedData struct {
	Admins  steamid.Collection `json:"admins"`
	Players steamid.Collection `json:"players"`
	Servers []struct {
		ShortName string  `json:"short_name"`
		Host      string  `json:"host"`
		Port      int     `json:"port,omitempty"`
		Password  string  `json:"password"`
		Latitude  float32 `json:"latitude"`
		Longitude float32 `json:"longitude"`
		Enabled   bool    `json:"enabled"`
		Region    string  `json:"region"`
		CC        string  `json:"cc"`
	} `json:"servers"`
	Settings struct {
		Rcon string `json:"rcon"`
	}
}

// seedCmd loads the db schema
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Add testing data",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		database, errStore := store.New(ctx, config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to initialize database connection: %v", errStore)
		}
		if !golib.Exists(seedFile) {
			log.Fatalf("Seedfile does not exist: %s", seedFile)
		}
		rawSeedData, errReadFile := os.ReadFile(seedFile)
		if errReadFile != nil {
			log.Fatalf("Failed to read seed file: %v", errReadFile)
		}
		var seed seedData
		if errUnmarshal := json.Unmarshal(rawSeedData, &seed); errUnmarshal != nil {
			log.Fatalf("failed to unmarshal seed file: %v", errUnmarshal)
		}
		for _, adminSid64 := range seed.Admins {
			person := model.NewPerson(adminSid64)
			if errGetPerson := database.GetOrCreatePersonBySteamID(ctx, adminSid64, &person); errGetPerson != nil {
				log.Fatalf("Failed to get person: %v", errGetPerson)
			}
			summary, errSummary := steamweb.PlayerSummaries(steamid.Collection{person.SteamID})
			if errSummary != nil {
				log.Errorf("Failed to get player summary: %v", errSummary)
				return
			}
			person.PermissionLevel = model.PAdmin
			person.PlayerSummary = &summary[0]
			if errSave := database.SavePerson(ctx, &person); errSave != nil {
				log.Errorf("Failed to save person: %v", errSave)
				return
			}
		}
		for _, playerSid := range seed.Players {
			person := model.NewPerson(playerSid)
			if errGetPerson := database.GetOrCreatePersonBySteamID(ctx, playerSid, &person); errGetPerson != nil {
				log.Fatalf("Failed to get person: %v", errGetPerson)
			}
			summary, errSummary := steamweb.PlayerSummaries(steamid.Collection{person.SteamID})
			if errSummary != nil {
				log.Errorf("Failed to get player summary: %v", errSummary)
				return
			}
			person.PermissionLevel = model.PUser
			person.PlayerSummary = &summary[0]
			if errSave := database.SavePerson(ctx, &person); errSave != nil {
				log.Errorf("Failed to save person: %v", errSave)
				return
			}
		}
		friendList, errFriendList := steamweb.GetFriendList(76561197961279983)
		if errFriendList != nil {
			log.Errorf("Failed to get friendlist")
		}
		for _, friend := range friendList {
			sid, errSid := steamid.StringToSID64(friend.Steamid)
			if errSid != nil {
				log.Panicf("Failed to parse friend steam id: %v", errSid)
			}
			person := model.NewPerson(sid)
			if errGetPerson := database.GetOrCreatePersonBySteamID(ctx, sid, &person); errGetPerson != nil {
				log.Errorf("Failed to create person: %v", errGetPerson)
			}
			var banSteam model.BanSteam
			if errNewBan := app.NewBanSteam(
				model.StringSID(config.General.Owner.String()),
				model.StringSID(seed.Players[0].String()), "500h", model.External,
				"", "imported", model.System, 0, model.Banned, &banSteam); errNewBan != nil {
				log.Errorf("Failed to create ban: %v", errNewBan)
			}
			if errSaveBan := database.SaveBan(ctx, &banSteam); errSaveBan != nil {
				log.Errorf("Failed to make ban: %v", errSaveBan)
			}
		}
		for _, server := range seed.Servers {
			pw := golib.RandomString(20)
			if server.Password != "" {
				pw = server.Password
			}
			rconPass := seed.Settings.Rcon
			if testRconPass != "" {
				rconPass = testRconPass
			}
			port := 27015
			if server.Port > 0 {
				port = server.Port
			}
			s := model.Server{
				ServerNameShort: server.ShortName,
				Address:         server.Host,
				Port:            port,
				RCON:            rconPass,
				ReservedSlots:   8,
				Password:        pw,
				IsEnabled:       server.Enabled,
				Region:          server.Region,
				CC:              server.CC,
				Latitude:        server.Latitude,
				Longitude:       server.Longitude,
				TokenCreatedOn:  config.Now(),
				CreatedOn:       config.Now(),
				UpdatedOn:       config.Now(),
			}
			if errSaveServer := database.SaveServer(ctx, &s); errSaveServer != nil {
				log.Errorf("Failed to add server: %v", errSaveServer)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)

	seedCmd.Flags().StringVarP(&testRconPass, "rcon", "r", "testing", "Sets the rcon password for seed data")
	seedCmd.Flags().StringVarP(&seedFile, "seed", "s", "seed.json", "Seed the database with this content")
}
