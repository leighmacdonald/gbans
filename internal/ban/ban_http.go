package ban

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
)

type banHandler struct {
	Bans

	siteName       string
	authorizedKeys []string
}

func NewHandlerBans(engine *gin.Engine, bans Bans, config Config, siteName string) {
	var b AppealState
	httphelper.Decoder.RegisterConverter(&b, func(input string) reflect.Value {
		value, errValue := strconv.ParseInt(input, 10, 64)
		if errValue != nil {
			return reflect.Value{}
		}
		state := AppealState(value)

		return reflect.ValueOf(&state)
	})

	handler := banHandler{
		Bans:           bans,
		authorizedKeys: strings.Split(config.AuthorizedKeys, ","),
		siteName:       siteName,
	}

	if config.BDEnabled {
		engine.GET("/export/bans/tf2bd", handler.onAPIExportBansTF2BD())
	}

	if config.ValveEnabled {
		engine.GET("/export/bans/valve/steamid", handler.onAPIExportBansValveSteamID())
	}
}

func (h banHandler) onAPIExportBansValveSteamID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(h.authorizedKeys) > 0 {
			key, ok := ctx.GetQuery("key")
			if !ok || !slices.Contains(h.authorizedKeys, key) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, permission.ErrDenied,
					"You do not have permission to access this resource. You can try contacting the administrator to obtain an api key."))

				return
			}
		}

		// TODO limit to perm?
		bans, errBans := h.Query(ctx, QueryOpts{})
		if errBans != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, httphelper.ErrInternal)))

			return
		}

		var entries strings.Builder

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}

			entries.WriteString(fmt.Sprintf("banid 0 %s\n", ban.TargetID.Steam(false)))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(entries.String()))
	}
}

func (h banHandler) onAPIExportBansTF2BD() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(h.authorizedKeys) > 0 {
			key, ok := ctx.GetQuery("key")
			if !ok || !slices.Contains(h.authorizedKeys, key) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, permission.ErrDenied,
					"You do not have permission to access this resource. You can try contacting the administrator to obtain an api key."))

				return
			}
		}

		bans, errBans := h.Query(ctx, QueryOpts{})
		if errBans != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, httphelper.ErrInternal)))

			return
		}

		var filtered []Ban

		for _, curBan := range bans {
			if curBan.Reason != reason.Cheating || curBan.Deleted || !curBan.IsEnabled {
				continue
			}

			filtered = append(filtered, curBan)
		}

		out := thirdparty.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
			FileInfo: thirdparty.FileInfo{
				Authors:     []string{h.siteName},
				Description: "Players permanently banned for cheating",
				Title:       h.siteName + " Cheater List",
				UpdateURL:   "/export/bans/tf2bd",
			},
			Players: []thirdparty.Players{},
		}

		for _, ban := range filtered {
			out.Players = append(out.Players, thirdparty.Players{
				Attributes: []string{"cheater"},
				Steamid:    ban.TargetID,
				LastSeen: thirdparty.LastSeen{
					PlayerName: ban.TargetID.String(),
					Time:       int(ban.UpdatedOn.Unix()),
				},
			})
		}

		ctx.JSON(http.StatusOK, out)
	}
}
