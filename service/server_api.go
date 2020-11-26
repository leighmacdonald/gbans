// This file contains handlers for communication with the sourcemod client on the
// game servers themselves
package service

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/bot"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/gbans/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func onPostLogMessage() gin.HandlerFunc {
	type logReq struct {
		ServerID  string        `json:"server_id"`
		SteamID   steamid.SID64 `json:"steam_id"`
		Name      string        `json:"name"`
		Message   string        `json:"message"`
		TeamSay   bool          `json:"team_say"`
		Timestamp int           `json:"timestamp"`
	}
	return func(c *gin.Context) {
		var req logReq
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("Failed to decode log message: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}
		filtered, word := util.IsFilteredWord(req.Message)
		if filtered {
			addWarning(req.SteamID, warnLanguage)
			for _, c := range config.Relay.ChannelIDs {
				bot.Send(bot.NewMessage(c, fmt.Sprintf("<@&%d> Word filter triggered: %s", config.Discord.ModRoleID, word)))
			}
		}
		// [us-2] 76561198017946808 name: message
		msgBody := req.Message
		if req.TeamSay {
			msgBody = "(Team) " + msgBody
		}
		msg := fmt.Sprintf(`[%s] %d **%s** %s`, req.ServerID, req.SteamID, req.Name, msgBody)
		for _, channelID := range config.Relay.ChannelIDs {
			bot.Send(bot.NewMessage(channelID, msg))
		}
		c.Status(200)
	}
}

func onPostServerAuth() gin.HandlerFunc {
	type authReq struct {
		ServerName string `json:"server_name"`
		Key        string `json:"key"`
	}
	type authResp struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
	}
	return func(c *gin.Context) {
		var req authReq
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("Failed to decode auth request: %v", err)
			c.JSON(500, authResp{Status: false})
			return
		}
		srv, err := store.GetServerByName(req.ServerName)
		if err != nil {
			c.JSON(http.StatusNotFound, authResp{Status: false})
			return
		}
		srv.Token = golib.RandomString(40)
		srv.TokenCreatedOn = time.Now().Unix()
		if err := store.SaveServer(&srv); err != nil {
			log.Errorf("Failed to updated server token: %v", err)
			c.JSON(500, authResp{Status: false})
			return
		}
		c.JSON(200, authResp{
			Status: true,
			Token:  srv.Token,
		})
	}
}

func onPostServerCheck() gin.HandlerFunc {
	type checkRequest struct {
		ClientID int    `json:"client_id"`
		SteamID  string `json:"steam_id"`
		IP       string `json:"ip"`
	}
	type checkResponse struct {
		ClientID int           `json:"client_id"`
		SteamID  string        `json:"steam_id"`
		BanType  model.BanType `json:"ban_type"`
		Msg      string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req checkRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		resp := checkResponse{
			ClientID: req.ClientID,
			SteamID:  req.SteamID,
			BanType:  model.Unknown,
			Msg:      "",
		}
		// Check IP first
		banNet, err := store.GetBanNet(req.IP)
		if err == nil {
			resp.BanType = model.Banned
			resp.Msg = banNet.Reason
			c.JSON(200, resp)
			return
		}
		// Check SteamID
		steamID, err := steamid.ResolveSID64(context.Background(), req.SteamID)
		if err != nil || !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			c.JSON(500, resp)
		}
		ban, err := store.GetBan(steamID)
		if err != nil {
			if store.DBErr(err) == store.ErrNoResult {
				resp.BanType = model.OK
				c.JSON(200, resp)
				return
			}
			resp.Msg = "Error determining state"
			c.JSON(500, resp)
			return
		}
		resp.BanType = ban.BanType
		resp.Msg = ban.ReasonText
		c.JSON(200, resp)
	}
}

func onGetServerBan() gin.HandlerFunc {
	type banStateRequest struct {
		SteamID string `json:"steam_id"`
	}
	type banStateResponse struct {
		SteamID string        `json:"steam_id"`
		BanType model.BanType `json:"ban_type"`
		Msg     string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req banStateRequest

		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, banStateResponse{
				SteamID: "",
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		c.JSON(200, gin.H{"status": model.OK})
	}
}
