package cmd

import (
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/service"
	"github.com/leighmacdonald/golib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var addServer = model.Server{
	ServerName: "",
	Token:      "",
	Address:    "",
	Port:       27015,
	RCON:       "",
	Password:   golib.RandomString(20),
}

// addServerCmd represents the addserver command
var addServerCmd = &cobra.Command{
	Use:   "addserver",
	Short: "Add a new server",
	Run: func(cmd *cobra.Command, args []string) {
		service.Init(config.DB.DSN)
		if addServer.ServerName == "" {
			log.Fatal("Server name cannot be empty")
		}
		if addServer.Address == "" {
			log.Fatal("Server address cannot be empty")
		}
		if addServer.Port <= 0 || addServer.Port > 65535 {
			log.Fatal("Invalid server port")
		}
		if addServer.RCON == "" {
			log.Fatal("RCON password cannot be empty")
		}
		if err := service.SaveServer(&addServer); err != nil {
			log.Fatalf("Could not create server: %v", err)
		}
		log.Infof("Added server %s with token %s - This token must be added to your servers gbans.cfg",
			addServer.ServerName, addServer.Password)
	},
}

func init() {
	rootCmd.AddCommand(addServerCmd)

	addServerCmd.Flags().StringVarP(&addServer.ServerName, "name", "n", "", "Short server ID eg: us-1")
	addServerCmd.Flags().StringVarP(&addServer.Address, "host", "H", "", "Server hostname/ip eg: us-1.myserver.com")
	addServerCmd.Flags().IntVarP(&addServer.Port, "port", "p", 27015, "Server port")
	addServerCmd.Flags().StringVarP(&addServer.RCON, "rcon", "r", "", "Server RCON password")
}
