package app

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func onAPIPostServer(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req serverUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		server := store.NewServer(req.ServerNameShort, req.Host, req.Port)
		server.Name = req.ServerName
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save new server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("Server config created",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

type serverUpdateRequest struct {
	ServerName      string  `json:"server_name"`
	ServerNameShort string  `json:"server_name_short"`
	Host            string  `json:"host"`
	Port            int     `json:"port"`
	ReservedSlots   int     `json:"reserved_slots"`
	RCON            string  `json:"rcon"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	CC              string  `json:"cc"`
	DefaultMap      string  `json:"default_map"`
	Region          string  `json:"region"`
	IsEnabled       bool    `json:"is_enabled"`
}

func onAPIPostServerUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req serverUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		server.ShortName = req.ServerNameShort
		server.Name = req.ServerName
		server.Address = req.Host
		server.Port = req.Port
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to update server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("Server config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIPostServerDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		server.Deleted = true

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)
		log.Info("Server config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIGetServersAdmin(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var filter store.ServerQueryFilter
		if !bind(ctx, log, &filter) {
			return
		}

		servers, count, errServers := app.db.GetServers(ctx, filter)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, servers))
	}
}

func onAPIPutPlayerPermission(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updatePermissionLevel struct {
		PermissionLevel consts.Privilege `json:"permission_level"`
	}

	return func(ctx *gin.Context) {
		steamID, errParam := getSID64Param(ctx, "steam_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var req updatePermissionLevel
		if !bind(ctx, log, &req) {
			return
		}

		var person store.Person
		if errGet := app.db.GetPersonBySteamID(ctx, steamID, &person); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to load person", zap.Error(errGet))

			return
		}

		if steamID == app.conf.General.Owner {
			responseErr(ctx, http.StatusConflict, errors.New("Cannot alter site owner permissions"))

			return
		}

		person.PermissionLevel = req.PermissionLevel

		if errSave := app.db.SavePerson(ctx, &person); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save person", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, person)

		log.Info("Player permission updated",
			zap.Int64("steam_id", steamID.Int64()),
			zap.String("permissions", person.PermissionLevel.String()))
	}
}
