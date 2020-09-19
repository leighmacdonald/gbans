package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
)

type BanState int

const (
	Unknown BanState = -1
	OK      BanState = 0
	NoComm  BanState = 1
	Banned  BanState = 2
)

var (
	router *gin.Engine
)

func init() {
	db = sqlx.MustConnect("sqlite3", "./db.sqlite")
	if err := setupDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	router = gin.Default()
	router.GET("/v1/ban", onGetBan())
	router.POST("/v1/auth", onPostAuth())
}

func main() {
	log.Infof("Starting gbans service")
	if err := router.Run("0.0.0.0:6969"); err != nil {
		log.Errorf("Error shutting down service: %v", err)
	}
}

func onPostAuth() gin.HandlerFunc {
	type authReq struct {
		ServerID string `json:"server_id"`
		Key      string `json:"key"`
	}
	type authResp struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
	}
	return func(c *gin.Context) {
		var req authReq
		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, authResp{Status: false})
			return
		}
		c.JSON(200, authResp{
			Status: true,
			Token:  "a_shiny_new_token",
		})
	}
}

func onGetBan() gin.HandlerFunc {
	type banStateRequest struct {
		SteamID string `json:"steam_id"`
	}

	type banStateResponse struct {
		SteamID  string   `json:"steam_id"`
		BanState BanState `json:"ban_state"`
		Msg      string   `json:"msg"`
	}
	return func(c *gin.Context) {
		var req banStateRequest

		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, banStateResponse{
				SteamID:  "",
				BanState: Unknown,
				Msg:      "Error determining state",
			})
			return
		}
		c.JSON(200, gin.H{"status": OK})
	}
}
