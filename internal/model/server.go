package model

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

type ServerPermission struct {
	SteamID         steamid.SID      `json:"steam_id"`
	PermissionLevel consts.Privilege `json:"permission_level"`
	Flags           string           `json:"flags"`
}

func NewServer(shortName string, address string, port int) Server {
	return Server{
		ShortName:      shortName,
		Address:        address,
		Port:           port,
		RCON:           util.SecureRandomString(10),
		ReservedSlots:  0,
		Password:       util.SecureRandomString(10),
		IsEnabled:      true,
		EnableStats:    true,
		TokenCreatedOn: time.Unix(0, 0),
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
	}
}

type Server struct {
	// Auto generated id
	ServerID int `db:"server_id" json:"server_id"`
	// ShortName is a short reference name for the server eg: us-1
	ShortName string `json:"short_name"`
	Name      string `json:"name"`
	// Address is the ip of the server
	Address string `db:"address" json:"address"`
	// Port is the port of the server
	Port int `db:"port" json:"port"`
	// RCON is the RCON password for the server
	RCON          string `db:"rcon" json:"rcon"`
	ReservedSlots int    `db:"reserved_slots" json:"reserved_slots"`
	// Password is what the server uses to generate a token to make authenticated calls (permanent refresh token)
	Password    string  `db:"password" json:"password"`
	IsEnabled   bool    `json:"is_enabled"`
	Deleted     bool    `json:"deleted"`
	Region      string  `json:"region"`
	CC          string  `json:"cc"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	LogSecret   int     `json:"log_secret"`
	EnableStats bool    `json:"enable_stats"`
	// TokenCreatedOn is set when changing the token
	TokenCreatedOn time.Time `db:"token_created_on" json:"token_created_on"`
	CreatedOn      time.Time `db:"created_on" json:"created_on"`
	UpdatedOn      time.Time `db:"updated_on" json:"updated_on"`
}

func (s Server) IP(ctx context.Context) (net.IP, error) {
	parsedIP := net.ParseIP(s.Address)
	if parsedIP != nil {
		// We already have an ip
		return parsedIP, nil
	}
	// TODO proper timeout for ctx
	ips, errResolve := net.DefaultResolver.LookupIP(ctx, "ip4", s.Address)
	if errResolve != nil || len(ips) == 0 {
		return nil, errors.Wrap(errResolve, "Could not resolve address")
	}

	return ips[0], nil
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}
