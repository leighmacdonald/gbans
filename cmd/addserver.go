package cmd

import (
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/service"
	"github.com/leighmacdonald/golib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"strconv"
)

// relayCmd represents the addserver command
var addserverCmd = &cobra.Command{
	Use:   "addserver",
	Short: "Add a new server",
	Long: `Add a new server.
	
gban addserver <server_name> <addr> <port> <rcon>
`,
	Run: func(cmd *cobra.Command, args []string) {
		service.Init(config.DB.DSN)
		if len(args) != 4 {
			log.Fatalf("Invalid arg count")
		}
		portStr := args[2]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("Invalid port")
		}
		s := model.Server{
			ServerName: args[0],
			Token:      "",
			Address:    args[1],
			Port:       port,
			RCON:       args[3],
			Password:   golib.RandomString(20),
		}
		if err := service.SaveServer(&s); err != nil {
			log.Fatalf("Could not create server: %v", err)
		}
		log.Infof("Added server %s with token %s", s.ServerName, s.Password)
	},
}

func init() {
	rootCmd.AddCommand(addserverCmd)
}
