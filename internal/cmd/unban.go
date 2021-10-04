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
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		if steamProfile == "" {
			log.Fatal("Steam ID cannot be empty")
		}
		sid, errSid := steamid.ResolveSID64(context.Background(), steamProfile)
		if errSid != nil {
			log.Fatalf("Failed to resolve steam id: %v", errSid)
		}
		if !sid.Valid() {
			log.Fatalf("Invalid steam id")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		ban := model.NewBannedPerson()
		if errBan := db.GetBanBySteamID(ctx, sid, false, &ban); errBan != nil {
			if errors.Is(errBan, store.ErrNoResult) {
				log.WithFields(log.Fields{"sid": sid.String()}).Fatalf("No ban found for steamid")
			}
			log.Fatalf("Invalid steam id")
		}
		if errDrop := db.DropBan(ctx, &ban.Ban); errDrop != nil {
			log.Fatalf("Failed to delete ban: %v", errDrop)
		}
		log.WithFields(log.Fields{"sid": sid.String()}).Info("Unbanned steam profile successfully")
	},
}

var unbanCIDRCmd = &cobra.Command{
	Use:   "cidr",
	Short: "Unban CIDR ban",
	Long:  `Unban CIDR ban`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		if cidr == "" {
			log.Fatal("CIDR cannot be empty")
		}
		ip, _, errParse := net.ParseCIDR(cidr)
		if errParse != nil {
			log.WithFields(log.Fields{"cidr": cidr}).Fatalf("Failed to parse cidr: %v", errParse)
		}
		banNets, errFetch := db.GetBanNet(ctx, ip)
		if errFetch != nil {
			if errors.Is(errFetch, store.ErrNoResult) {
				log.WithFields(log.Fields{"cidr": cidr}).Fatalf("No intersecting cidr found")
			}
			log.WithFields(log.Fields{"cidr": cidr}).Fatalf("Failed to fetch matching cidr ban: %v", errFetch)
		}
		if len(banNets) == 0 {
			log.WithFields(log.Fields{"cidr": cidr}).Fatal("Failed to find matching banned cidr networks")
		}
		for _, bn := range banNets {
			if err := db.DropBanNet(ctx, &bn); err != nil {
				log.Fatalf("Failed to drop ban net: %v", err)
			}
			log.WithFields(log.Fields{"cidr": bn.CIDR.String()}).Infof("Ban net dropped")
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
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		var asnBan model.BanASN
		if errFetch := db.GetBanASN(ctx, asn, &asnBan); errFetch != nil {
			if errors.Is(errFetch, store.ErrNoResult) {
				log.Fatalf("Existing ASN ban not found")
			}
			log.WithFields(log.Fields{"asn": asn}).Fatalf("Failed to fetch asn ban: %v", errFetch)
		}
		if errDrop := db.DropBanASN(ctx, &asnBan); errDrop != nil {
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
