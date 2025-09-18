package message

import (
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/olekukonko/tablewriter"
)

const (
	ColourSuccess = 302673
	ColourInfo    = 3581519
	ColourWarn    = 14327864
	ColourError   = 13631488
)

func DefaultTable(writer io.Writer) *tablewriter.Table {
	tbl := tablewriter.NewTable(writer)
	// tbl.Configure(func(cfg *tablewriter.Config) {
	// 	cfg.Header.Formatting = tw.AlignLeft
	// })
	// tbl.W(tablewriter.WithHeaderAlignment(tw.AlignLeft))
	// tbl.Config().Header.Formatting = tablewriter.
	// 	tbl.SetAutoFormatHeaders(true)
	// tbl.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	// tbl.SetCenterSeparator("")
	// tbl.SetColumnSeparator("")
	// tbl.SetRowSeparator("")
	// tbl.SetHeaderLine(false)
	// tbl.SetTablePadding("")
	// tbl.SetAutoMergeCells(true)
	// tbl.SetAlignment(tablewriter.ALIGN_LEFT)

	return tbl
}

func ErrorMessage(command string, err error) *discordgo.MessageEmbed {
	return NewEmbed("Error Returned").Embed().
		SetColor(ColourError).
		AddField("command", command).
		SetDescription(err.Error()).MessageEmbed
}

// func BanMessage(ban model.BannedSteamPerson, link string, target model.PersonInfo, source model.PersonInfo) *discordgo.MessageEmbed {
//	msgEmbed := NewEmbed()
//	msgEmbed.Embed().
//		SetTitle(fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)).
//		SetDescription(ban.Note).
//		SetURL(link).
//		SetColor(ColourSuccess)
//
//	if ban.ReasonText != "" {
//		msgEmbed.Embed().AddField("Reason", ban.ReasonText)
//	}
//
//	msgEmbed.Embed().SetImage(target.GetAvatar().Full())
//
//	msgEmbed.AddAuthorPersonInfo(source, "")
//
//	if ban.ValidUntil.Year()-time.Now().Year() > 5 {
//		msgEmbed.Embed().AddField("Expires In", "Permanent")
//		msgEmbed.Embed().AddField("Expires At", "Permanent")
//	} else {
//		msgEmbed.Embed().AddField("Expires In", util.FmtDuration(ban.ValidUntil))
//		msgEmbed.Embed().AddField("Expires At", util.FmtTimeShort(ban.ValidUntil))
//	}
//
//	msgEmbed.AddFieldsSteamID(ban.TargetID)
//
//	return msgEmbed.Embed().Truncate().MessageEmbed
// }

func InfString(f float64) string {
	if f == -1 {
		return "∞"
	}

	return fmt.Sprintf("%.1f", f)
}
