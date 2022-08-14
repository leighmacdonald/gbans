package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"time"
)

var unbanCmd = &cobra.Command{
	Use:   "unban",
	Short: "unban functions",
	Long:  `Functionality for unbans`,
}

var unbanSteamCmd = &cobra.Command{
	Use:   "steam",
	Short: "Unban an existing steam profile ban",
	Long:  `Unban an existing steam profile ban`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		database, errStore := store.New(ctx, config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to setup database connection: %v", errStore)
		}
		if steamProfile == "" {
			log.Fatal("Steam ID cannot be empty")
		}
		sid64, errResolve := steamid.ResolveSID64(context.Background(), steamProfile)
		if errResolve != nil {
			log.Fatalf("Failed to resolve steam id: %v", errResolve)
		}
		if !sid64.Valid() {
			log.Fatalf("Invalid steam id")
		}
		ban := model.NewBannedPerson()
		if errGetBan := database.GetBanBySteamID(ctx, sid64, false, &ban); errGetBan != nil {
			if errors.Is(errGetBan, store.ErrNoResult) {
				log.WithFields(log.Fields{"sid64": sid64.String()}).Fatalf("No ban found for steamid")
			}
			log.Fatalf("Invalid steam id")
		}
		if errDrop := database.DropBan(ctx, &ban.Ban, false); errDrop != nil {
			log.Fatalf("Failed to delete ban: %v", errDrop)
		}
		log.WithFields(log.Fields{"sid64": sid64.String()}).Info("Unbanned steam profile successfully")
	},
}

var unbanCIDRCmd = &cobra.Command{
	Use:   "cidr",
	Short: "Unban CIDR ban",
	Long:  `Unban CIDR ban`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		database, errStore := store.New(ctx, config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to setup database connection: %v", errStore)
		}
		if cidr == "" {
			log.Fatal("CIDR cannot be empty")
		}
		ip, _, errParse := net.ParseCIDR(cidr)
		if errParse != nil {
			log.WithFields(log.Fields{"cidr": cidr}).Fatalf("Failed to parse cidr: %v", errParse)
		}
		banNets, errGetBanNet := database.GetBanNetByAddress(ctx, ip)
		if errGetBanNet != nil {
			if errors.Is(errGetBanNet, store.ErrNoResult) {
				log.WithFields(log.Fields{"cidr": cidr}).Fatalf("No intersecting cidr found")
			}
			log.WithFields(log.Fields{"cidr": cidr}).Fatalf("Failed to fetch matching cidr ban: %v", errGetBanNet)
		}
		if len(banNets) == 0 {
			log.WithFields(log.Fields{"cidr": cidr}).Fatal("Failed to find matching banned cidr networks")
		}
		for _, banNet := range banNets {
			if errDropBanNet := database.DropBanNet(ctx, &banNet); errDropBanNet != nil {
				log.Fatalf("Failed to drop ban net: %v", errDropBanNet)
			}
			log.WithFields(log.Fields{"cidr": banNet.CIDR.String()}).Infof("BanSteam net dropped")
		}
	},
}

var unbanASNCmd = &cobra.Command{
	Use:   "asn",
	Short: "Unban an ASN ban",
	Long:  `Unban an ASN ban`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		database, errStore := store.New(ctx, config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to setup database connection: %v", errStore)
		}
		var asnBan model.BanASN
		if errFetch := database.GetBanASN(ctx, asn, &asnBan); errFetch != nil {
			if errors.Is(errFetch, store.ErrNoResult) {
				log.Fatalf("Existing ASN ban not found")
			}
			log.WithFields(log.Fields{"asn": asn}).Fatalf("Failed to fetch asn ban: %v", errFetch)
		}
		if errDrop := database.DropBanASN(ctx, &asnBan); errDrop != nil {
			log.WithFields(log.Fields{"asn": asn}).Fatalf("Failed to drop asn ban: %v", errDrop)
		}
		log.WithFields(log.Fields{"asn": asn}).Infof("ASN ban create successfully")
	},
}

func init() {
	unbanSteamCmd.Flags().StringVarP(&steamProfile, "sid", "s", "", "SteamID or profile to ban")
	unbanCIDRCmd.Flags().StringVarP(&cidr, "cidr", "n", "", "Network CIDR: 1.2.3.0/24, 1.2.3.4/32")
	unbanASNCmd.Flags().Int64VarP(&asn, "asn", "a", 0, "Autonomous Systems Number to ban eg: 10551")

	unbanCmd.AddCommand(unbanSteamCmd)
	unbanCmd.AddCommand(unbanCIDRCmd)
	unbanCmd.AddCommand(unbanASNCmd)

	rootCmd.AddCommand(unbanCmd)

}
