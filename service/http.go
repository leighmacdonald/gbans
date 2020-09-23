package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/golib"
	log "github.com/sirupsen/logrus"
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

func init() {
	router = gin.Default()
	router.GET("/v1/ban", onGetBan())
	router.POST("/v1/auth", onPostAuth())
}

func Listen(addr string) {
	log.Infof("Starting gbans service")
	if err := router.Run(addr); err != nil {
		log.Errorf("Error shutting down service: %v", err)
	}
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
			//c.JSON(500, authResp{Status: false})
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
		SteamID    string       `json:"steam_id"`
		AuthorID   string       `json:"author_id"`
		Duration   string       `json:"duration"`
		IP         string       `json:"ip"`
		Reason     model.Reason `json:"reason"`
		ReasonText string       `json:"reason_text"`
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
				Message: `Invalid duration. Examples: "300ms", "-1.5h" or "2h45m". 
Valid time units are "ns", "us", "ms", "s", "m", "h".`,
			})
		}
		ip := net.ParseIP(r.IP)
		if err := Ban(r.SteamID, r.AuthorID, duration, ip, r.Reason, r.ReasonText); err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
		}
	}
}

func onGetBan() gin.HandlerFunc {
	type banStateRequest struct {
		SteamID string `json:"steam_id"`
	}

	type banStateResponse struct {
		SteamID  string        `json:"steam_id"`
		BanState model.BanType `json:"ban_type"`
		Msg      string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req banStateRequest

		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, banStateResponse{
				SteamID:  "",
				BanState: model.Unknown,
				Msg:      "Error determining state",
			})
			return
		}
		c.JSON(200, gin.H{"status": model.OK})
	}
}
