package service

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"github.com/toorop/gin-logrus"
	"net"
	"net/http"
	"time"
)

var (
	router *gin.Engine
)

type StatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func checkServerAuth(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || len(token) != 40 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	log.Debugf("Authed as: %s", token)
	if !store.TokenValid(token) {
		log.Warnf("Received invalid server token from %s", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	c.Next()
}

func startHTTP(ctx context.Context, addr string) {
	router = gin.New()
	router.Use(ginlogrus.Logger(log.StandardLogger()), gin.Recovery())
	router.POST("/v1/auth", onPostAuth())
	authed := router.Group("/", checkServerAuth)
	authed.GET("/v1/ban", onGetBan())
	authed.POST("/v1/check", onPostCheck())
	log.Infof("Starting gbans service")
	go func() {
		if err := router.Run(addr); err != nil {
			log.Errorf("Error shutting down service: %v", err)
		}
	}()
	<-ctx.Done()
}

func onPostAuth() gin.HandlerFunc {
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

func onPostBan() gin.HandlerFunc {
	type req struct {
		SteamID    string        `json:"steam_id"`
		AuthorID   string        `json:"author_id"`
		Duration   string        `json:"duration"`
		IP         string        `json:"ip"`
		BanType    model.BanType `json:"ban_type"`
		Reason     model.Reason  `json:"reason"`
		ReasonText string        `json:"reason_text"`
	}

	return func(c *gin.Context) {
		var r req
		if err := c.BindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
			return
		}
		duration, err := time.ParseDuration(r.Duration)
		if err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: `Invalid duration. Examples: "300m", "1.5h" or "2h45m". 
Valid time units are "s", "m", "h".`,
			})
		}
		ip := net.ParseIP(r.IP)
		if err := Ban(c, r.SteamID, r.AuthorID, duration, ip, r.BanType, r.Reason, r.ReasonText, model.Web); err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
		}
	}
}

func onPostCheck() gin.HandlerFunc {
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

func onGetBan() gin.HandlerFunc {
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
