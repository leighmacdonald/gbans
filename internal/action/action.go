// Package action defines a set of common argument structures
package action

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"net"
	"time"
)

// Executor implements the common interface for the core application functionality. Its currently implemented
// like this so that we can avoid cyclic dependency issues. This is a strong candidate for a better refactor.
// The secondary purpose is to allow a common interface for executing action logic in a common manner from
// multiple different interfaces, such as web and discord.
type Executor interface {
	Find(playerStr string, ip string, pi *model.PlayerInfo) error
	FindPlayerByCIDR(ipNet *net.IPNet, pi *model.PlayerInfo) error
	PersonBySID(sid steamid.SID64, ipAddr string, p *model.Person) error
	GetOrCreateProfileBySteamID(ctx context.Context, sid steamid.SID64, ipAddr string, p *model.Person) error
	Mute(args MuteRequest, pi *model.PlayerInfo) error
	Ban(args BanRequest, b *model.Ban) error
	Unban(args UnbanRequest) (bool, error)
	BanNetwork(args BanNetRequest, net *model.BanNet) error
	Kick(args KickRequest, pi *model.PlayerInfo) error
	Say(args SayRequest) error
	CSay(args CSayRequest) error
	PSay(args PSayRequest) error
	ResolveSID(sidStr string) (steamid.SID64, error)
	SetSteam(args SetSteamIDRequest) (bool, error)
	ServerState() model.ServerStateCollection
	ContainsFilteredWord(body string) (bool, model.Filter)
	FilterDel(ctx context.Context, args FilterDelRequest) (bool, error)
	FilterAdd(args FilterAddRequest) (model.Filter, error)
	FilterCheck(args FilterCheckRequest) []model.Filter
}

type Origin int

const (
	// Core is when internal systems triggered the event. eg: background workers/word filters
	Core Origin = iota
	// Discord is a event that was via discord application commands
	Discord
	// Web is for events triggered via the web or API interface
	Web
)

// BaseOrigin defines the base struct for all actions. It just marks where the event originated.
type BaseOrigin struct {
	Origin Origin
}

type GetChatHistoryRequest struct {
	BaseOrigin
	Target
	Page int
}

type GetOrCreatePersonByIDRequest struct {
	BaseOrigin
	Target
	IPAddr string
}

type GetOrCreateProfileBySteamIDRequest GetOrCreatePersonByIDRequest

type FilterAddRequest struct {
	BaseOrigin
	Source
	Filter string
}

type FilterDelRequest struct {
	BaseOrigin
	Source
	FilterID int
}

type FilterCheckRequest struct {
	BaseOrigin
	Source
	Message string
}

type ServerByNameRequest struct {
	BaseOrigin
	ServerName string
}

type SayRequest struct {
	BaseOrigin
	Source
	Server  string
	Message string
}

type CSayRequest SayRequest

type PSayRequest struct {
	BaseOrigin
	Source
	Target
	Message string
}

type KickRequest struct {
	BaseOrigin
	Target
	Source
	Reason string
}

type BanRequest struct {
	BaseOrigin
	Target
	Source
	Duration
	Reason string
}

type MuteRequest BanRequest
type UnbanRequest KickRequest

type BanNetRequest struct {
	BaseOrigin
	Target
	Source
	Duration
	CIDR   string
	Reason string
}

type ProfileRequest struct {
	BaseOrigin
	Target
	IPAddr string
}

type FindCIDRRequest struct {
	BaseOrigin
	CIDR *net.IPNet
}
type FindRequest struct {
	BaseOrigin
	Query string
}
type GetBanRequest struct {
	BaseOrigin
	Target
}
type GetBanNetRequest GetBanRequest
type GetHistoryIPRequest struct {
	BaseOrigin
	Source
	Target
}
type GetHistoryChatRequest GetHistoryIPRequest
type GetPersonByIDRequest GetBanRequest
type SetSteamIDRequest struct {
	BaseOrigin
	Target
	DiscordID string
}
type GetASNRecordRequest struct {
	BaseOrigin
	IPAddr string
}
type GetLocationRecordRequest GetASNRecordRequest
type GetProxyRecordRequest GetASNRecordRequest

func NewFindByCIDR(o Origin, cidr *net.IPNet) FindCIDRRequest {
	return FindCIDRRequest{
		BaseOrigin: BaseOrigin{Origin: o},
		CIDR:       cidr,
	}
}

func NewFind(o Origin, q string) FindRequest {
	return FindRequest{BaseOrigin: BaseOrigin{o}, Query: q}
}

func NewMute(o Origin, target string, author string, reason string, duration string) MuteRequest {
	return MuteRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		Source:     Source(author),
		Reason:     reason,
		Duration:   Duration(duration),
	}
}

func NewKick(o Origin, target string, author string, reason string) KickRequest {
	return KickRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		Source:     Source(author),
		Reason:     reason,
	}
}

func NewBan(o Origin, target string, author string, reason string, duration string) BanRequest {
	return BanRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		Source:     Source(author),
		Reason:     reason,
		Duration:   Duration(duration),
	}
}

func NewBanNet(o Origin, target string, author string, reason string, duration string, cidr string) BanNetRequest {
	return BanNetRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		Source:     Source(author),
		Reason:     reason,
		Duration:   Duration(duration),
		CIDR:       cidr,
	}
}

func NewUnban(o Origin, target string, author string, reason string) UnbanRequest {
	return UnbanRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		Source:     Source(author),
		Reason:     reason,
	}
}

func NewGetBan(o Origin, target string) GetBanRequest {
	return GetBanRequest{BaseOrigin: BaseOrigin{o}, Target: Target(target)}
}

func NewGetBanNet(o Origin, target string) GetBanNetRequest {
	return GetBanNetRequest{BaseOrigin: BaseOrigin{o}, Target: Target(target)}
}

func NewGetHistoryIP(o Origin, target string) GetHistoryIPRequest {
	return GetHistoryIPRequest{BaseOrigin: BaseOrigin{o}, Target: Target(target)}
}

func NewGetHistoryChat(o Origin, target string) GetHistoryChatRequest {
	return GetHistoryChatRequest{BaseOrigin: BaseOrigin{o}, Target: Target(target)}
}

func NewGetPersonByID(o Origin, target string) GetPersonByIDRequest {
	return GetPersonByIDRequest{BaseOrigin: BaseOrigin{o}, Target: Target(target)}
}

func NewSetSteamID(o Origin, target string, discordID string) SetSteamIDRequest {
	return SetSteamIDRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		DiscordID:  discordID,
	}
}

func NewGetASNRecord(o Origin, ipAddr string) GetASNRecordRequest {
	return GetASNRecordRequest{BaseOrigin: BaseOrigin{o}, IPAddr: ipAddr}
}

func NewGetLocationRecord(o Origin, ipAddr string) GetLocationRecordRequest {
	return GetLocationRecordRequest{BaseOrigin: BaseOrigin{o}, IPAddr: ipAddr}
}

func NewGetProxyRecord(o Origin, ipAddr string) GetProxyRecordRequest {
	return GetProxyRecordRequest{BaseOrigin: BaseOrigin{o}, IPAddr: ipAddr}
}

func NewSay(o Origin, server string, message string) SayRequest {
	return SayRequest{BaseOrigin: BaseOrigin{o}, Server: server, Message: message}
}

func NewCSay(o Origin, server string, message string) CSayRequest {
	return CSayRequest{BaseOrigin: BaseOrigin{o}, Server: server, Message: message}
}

func NewPSay(o Origin, target string, message string) PSayRequest {
	return PSayRequest{
		BaseOrigin: BaseOrigin{o},
		Message:    message,
		Target:     Target(target),
	}
}

func NewServerByName(o Origin, serverID string) ServerByNameRequest {
	return ServerByNameRequest{BaseOrigin: BaseOrigin{o}, ServerName: serverID}
}

func NewFilterAdd(o Origin, filter string) FilterAddRequest {
	return FilterAddRequest{BaseOrigin: BaseOrigin{o}, Filter: filter}
}

func NewFilterDel(o Origin, filterID int) FilterDelRequest {
	return FilterDelRequest{BaseOrigin: BaseOrigin{o}, FilterID: filterID}
}

func NewFilterCheck(o Origin, message string) FilterCheckRequest {
	return FilterCheckRequest{
		BaseOrigin: BaseOrigin{o},
		Message:    message}
}

func NewGetOrCreatePersonByID(o Origin, target string, ipAddr string) GetOrCreatePersonByIDRequest {
	return GetOrCreatePersonByIDRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		IPAddr:     ipAddr,
	}
}

func NewGetOrCreateProfileBySteamID(o Origin, target string, ipAddr string) GetOrCreateProfileBySteamIDRequest {
	return GetOrCreateProfileBySteamIDRequest{
		BaseOrigin: BaseOrigin{o},
		Target:     Target(target),
		IPAddr:     ipAddr,
	}
}

func NewGetChatHistory(o Origin, target string, page int) GetChatHistoryRequest {
	return GetChatHistoryRequest{
		Target: Target(target),
		Page:   page,
	}
}

// Target defines who the request is being made against
type Target string

func (t Target) SID64() (steamid.SID64, error) {
	v, err := steamid.ResolveSID64(context.Background(), string(t))
	if err != nil {
		return 0, consts.ErrInvalidSID
	}
	if !v.Valid() {
		return 0, consts.ErrInvalidSID
	}
	return v, nil
}

// Source defines who initiated the request
type Source string

func (a Source) SID64() (steamid.SID64, error) {
	v, err := steamid.ResolveSID64(context.Background(), string(a))
	if err != nil {
		return 0, consts.ErrInvalidSID
	}
	if !v.Valid() {
		return 0, consts.ErrInvalidSID
	}
	return v, nil
}

// Duration defines the length of time the action should be valid for
// A duration of 0 will be interpreted as permanent and set to 10 years in the future
type Duration string

func (d Duration) Value() (time.Duration, error) {
	dur, err := config.ParseDuration(string(d))
	if err != nil {
		return 0, consts.ErrInvalidDuration
	}
	if dur < 0 {
		return 0, consts.ErrInvalidDuration
	}
	if dur == 0 {
		dur = time.Hour * 24 * 365 * 10
	}
	return dur, nil
}
