package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var (
	asn          = int64(0)
	steamProfile = ""
	cidr         = ""
	reason       = ""
	duration     = ""
)

// serverCmd represents the addserver command
var banCmd = &cobra.Command{
	Use:   "ban",
	Short: "ban functions",
	Long:  `Functionality for ban, or modifying bans`,
}

var banSteamCmd = &cobra.Command{
	Use:   "steam",
	Short: "create a steam ban",
	Long:  `Create a new steam ban in the database`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		db, errNewStore := store.New(ctx, config.DB.DSN)
		if errNewStore != nil {
			log.Fatalf("Failed to setup db connection: %v", errNewStore)
		}
		if reason == "" {
			log.Fatal("Ban reason cannot be empty")
		}
		if duration == "" {
			log.Fatal("Duration cannot be empty")
		}
		if steamProfile == "" {
			log.Fatal("Steam ID cannot be empty")
		}
		sid, errSid := steamid.ResolveSID64(ctx, steamProfile)
		if errSid != nil {
			log.Fatalf("Failed to resolve steam id: %v", errSid)
		}
		if !sid.Valid() {
			log.Fatalf("Invalid steam id")
		}
		var banSteam model.BanSteam
		if errOpts := app.NewBanSteam(
			model.StringSID("0"),
			model.StringSID(sid.String()),
			model.Duration(duration),
			model.Cheating,
			"",
			"",
			model.System,
			0,
			model.Banned,
			&banSteam,
		); errOpts != nil {
			log.Fatalf("Invalid option: %v", errOpts)
		}
		if errSaveBan := db.SaveBan(ctx, &banSteam); errSaveBan != nil {
			log.WithFields(log.Fields{"sid": sid.String()}).Fatalf("Could not create create ban: %v", errSaveBan)
		}
		log.WithFields(log.Fields{"reason": reason, "until": banSteam.ValidUntil.String()}).
			Info("Added ban successfully")
	},
}

var banCIDRCmd = &cobra.Command{
	Use:   "cidr",
	Short: "Create an CIDR ban",
	Long: `Create an CIDR ban. This bans connections from all hosts within the CIDR range. Use 1.2.3.4/32 to add 
	a single IP`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		db, errNewStore := store.New(ctx, config.DB.DSN)
		if errNewStore != nil {
			log.Fatalf("Failed to setup db connection: %v", errNewStore)
		}
		if reason == "" {
			log.Fatal("Ban reason cannot be empty")
		}
		if duration == "" {
			log.Fatal("Duration cannot be empty")
		}
		if cidr == "" {
			log.Fatal("CIDR cannot be empty")
		}
		var banCIDR model.BanCIDR
		if errNewBanNet := app.NewBanCIDR(
			model.StringSID(config.General.Owner.String()),
			model.StringSID("0"),
			model.Duration(duration),
			model.Cheating,
			"",
			"",
			model.System,
			cidr,
			model.Banned,
			&banCIDR,
		); errNewBanNet != nil {
			log.WithFields(log.Fields{"cidr": cidr}).Fatalf("Failed to create BanCIDR instance: %v", errNewBanNet)
		}
		if errSaveBanNet := db.SaveBanNet(ctx, &banCIDR); errSaveBanNet != nil {
			if errors.Is(errSaveBanNet, store.ErrNoResult) {
				log.WithFields(log.Fields{"cidr": cidr}).Fatalf("Duplicate cidr ban found: %s", serverId)
			}
			log.WithFields(log.Fields{"cidr": cidr}).Fatalf("Failed to setup db connection: %v", errSaveBanNet)
		}
		log.WithFields(log.Fields{"cidr": cidr}).Infof("CIDR ban created successfully")
	},
}

var banASNCmd = &cobra.Command{
	Use:   "asn",
	Short: "Create an ASN ban",
	Long:  `Create an ASN ban. This bans connections from all networks under control of the ASN`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		db, errNewStore := store.New(ctx, config.DB.DSN)
		if errNewStore != nil {
			log.Fatalf("Failed to setup db connection: %v", errNewStore)
		}
		var banASN model.BanASN
		if errBanASN := app.NewBanASN(
			model.StringSID(config.General.Owner.String()),
			"0",
			model.Duration(duration),
			model.Cheating,
			"",
			"",
			model.System,
			asn,
			model.Banned,
			&banASN,
		); errBanASN != nil {
			log.Fatalf("Error: %v", errBanASN)
		}
		if errSave := db.SaveBanASN(ctx, &banASN); errSave != nil {
			log.WithFields(log.Fields{"asn": asn}).Fatalf("Failed to save netban: %v", errSave)
		}
		log.WithFields(log.Fields{"asn": asn}).Infof("ASN ban create successfully")
	},
}

func init() {
	banSteamCmd.Flags().StringVarP(&steamProfile, "sid", "s", "", "SteamID or profile to ban")
	banSteamCmd.Flags().StringVarP(&reason, "reason", "r", "", "Ban reason")
	banSteamCmd.Flags().StringVarP(&duration, "duration", "d", "0", "Duration of ban")

	banCIDRCmd.Flags().StringVarP(&steamProfile, "sid", "s", "", "SteamID or profile to ban")
	banCIDRCmd.Flags().StringVarP(&reason, "reason", "r", "", "Ban reason")
	banCIDRCmd.Flags().StringVarP(&duration, "duration", "d", "0", "Duration of ban")
	banCIDRCmd.Flags().StringVarP(&cidr, "cidr", "n", "", "Network CIDR: 1.2.3.0/24, 1.2.3.4/32")

	banASNCmd.Flags().Int64VarP(&asn, "asn", "a", 0, "Autonomous Systems Number to ban eg: 10551")
	banASNCmd.Flags().StringVarP(&reason, "reason", "r", "", "Ban reason")
	banASNCmd.Flags().StringVarP(&duration, "duration", "d", "0", "Duration of ban")

	banCmd.AddCommand(banSteamCmd)
	banCmd.AddCommand(banCIDRCmd)
	banCmd.AddCommand(banASNCmd)

	rootCmd.AddCommand(banCmd)

}
