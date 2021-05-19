// Package action defines a set of common structures and simple message passing
// channels to move them around.
//
// This package was created to decouple the interfaces, currently discord and http, further
// from the core application. Communication of results occurs using the Action.Result channel
// Since we are getting values from external sources, such as discord bots, sticking to simple string
// values makes it easier to do parsing/validation in one location. In other words, its better to let
// the core application to the parsing / validation of arguments so that it is centralized.
//
// As an example of making a async request and waiting for the results to be sent back on the
// results channel.
//
// 		req := NewKick("76561199040918801", "76561197992870439", "test")
//		req.Enqueue()
//		result := <-req.Done()
//
// If you do not care about the results, fire and forget. Then use the EnqueueIgnore() function
// instead which will omit sending results to the channel and will just close it.
//
//		req2 := NewBan("76561199040918801", "76561197992870439", "test", "1m")
//  	req2.EnqueueIgnore()
//
package action

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"net"
	"time"
)

var (
	queue          chan *Action
	ErrInvalidArgs = errors.New("Invalid args")
)

type Origin int

const (
	Core Origin = iota
	Discord
	Web
)

type Type int

const (
	Kick Type = iota
	Mute
	Ban
	BanNet
	Unban
	Find
	FindByCIDR
	GetBan
	GetBanNet
	GetHistoryIP
	GetHistoryChat
	GetPersonByID
	GetOrCreatePersonByID
	GetOrCreateProfileBySteamID
	SetSteamID
	GetASNRecord
	GetLocationRecord
	GetProxyRecord
	GetChatHistory
	Servers
	ServerByName
	Say
	CSay
	PSay
	AddFilter
	DelFilter
	CheckFilter
)

type Result struct {
	Err     error
	Message string
	Value   interface{}
}

type Action struct {
	Type         Type
	Args         interface{}
	Origin       Origin
	Result       chan Result
	Created      time.Time
	IgnoreResult bool
}

func (a *Action) Done() <-chan Result {
	return a.Result
}

func Register(receiver chan *Action) {
	queue = receiver
}

func (a *Action) Enqueue() *Action {
	queue <- a
	return a
}

func (a *Action) EnqueueIgnore() *Action {
	a.IgnoreResult = true
	close(a.Result)
	queue <- a
	return a
}

func (a *Action) SetResult(r Result) {
	if a.IgnoreResult {
		return
	}
	a.Result <- r
}

// New should generally not be called directly. Prefer to use New* helper methods
// whenever possible
func New(t Type, o Origin, args interface{}) Action {
	return Action{
		Type:         t,
		Args:         args,
		Origin:       o,
		Result:       make(chan Result),
		Created:      config.Now(),
		IgnoreResult: false,
	}
}

type GetChatHistoryRequest struct {
	Target
	Page int
}

type GetOrCreatePersonByIDRequest struct {
	Target
	IPAddr string
}

type GetOrCreateProfileBySteamIDRequest GetOrCreatePersonByIDRequest

type FilterAddRequest struct {
	Source
	Filter string
}

type FilterDelRequest struct {
	Source
	FilterID int
}

type FilterCheckRequest struct {
	Source
	Message string
}

type ServerByNameRequest struct {
	ServerName string
}

type SayRequest struct {
	Source
	Server  string
	Message string
}

type CSayRequest SayRequest

type PSayRequest struct {
	Source
	Target
	Message string
}

type KickRequest struct {
	Target
	Source
	Reason string
}

type BanRequest struct {
	Target
	Source
	Duration
	Reason string
}

type MuteRequest BanRequest
type UnbanRequest KickRequest

type BanNetRequest struct {
	Target
	Source
	Duration
	CIDR   string
	Reason string
}

type ProfileRequest struct {
	Target
	IPAddr string
}

type FindCIDRRequest struct{ CIDR *net.IPNet }
type FindRequest struct{ Query string }
type GetBanRequest struct{ Target }
type GetBanNetRequest GetBanRequest
type GetHistoryIPRequest struct {
	Source
	Target
}
type GetHistoryChatRequest GetHistoryIPRequest
type GetPersonByIDRequest GetBanRequest
type SetSteamIDRequest struct {
	Target
	DiscordID string
}
type GetASNRecordRequest struct{ IPAddr string }
type GetLocationRecordRequest GetASNRecordRequest
type GetProxyRecordRequest GetASNRecordRequest

func NewFindByCIDR(cidr *net.IPNet) Action {
	return New(FindByCIDR, Core, FindCIDRRequest{CIDR: cidr})
}

func NewFind(o Origin, q string) Action {
	return New(Find, o, FindRequest{Query: q})
}

func NewMute(o Origin, target string, author string, reason string, duration string) Action {
	return New(Mute, o, MuteRequest{
		Target:   Target(target),
		Source:   Source(author),
		Reason:   reason,
		Duration: Duration(duration),
	})
}

func NewKick(o Origin, target string, author string, reason string) Action {
	return New(Kick, o, KickRequest{
		Target: Target(target),
		Source: Source(author),
		Reason: reason,
	})
}

func NewBan(o Origin, target string, author string, reason string, duration string) Action {
	return New(Ban, o, BanRequest{
		Target:   Target(target),
		Source:   Source(author),
		Reason:   reason,
		Duration: Duration(duration),
	})
}

func NewBanNet(o Origin, target string, author string, reason string, duration string, cidr string) Action {
	return New(BanNet, o, BanNetRequest{
		Target:   Target(target),
		Source:   Source(author),
		Reason:   reason,
		Duration: Duration(duration),
		CIDR:     cidr,
	})
}

func NewUnban(o Origin, target string, author string, reason string) Action {
	return New(Unban, o, UnbanRequest{
		Target: Target(target),
		Source: Source(author),
		Reason: reason,
	})
}

func NewGetBan(o Origin, target string) Action {
	return New(GetBan, Core, GetBanRequest{Target: Target(target)})
}

func NewGetBanNet(target string) Action {
	return New(GetBanNet, Core, GetBanNetRequest{Target: Target(target)})
}

func NewGetHistoryIP(target string) Action {
	return New(GetHistoryIP, Core, GetHistoryIPRequest{Target: Target(target)})
}

func NewGetHistoryChat(target string) Action {
	return New(GetHistoryChat, Core, GetHistoryChatRequest{Target: Target(target)})
}

func NewGetPersonByID(target string) Action {
	return New(GetPersonByID, Core, GetPersonByIDRequest{Target: Target(target)})
}

func NewSetSteamID(o Origin, target string, discordID string) Action {
	return New(SetSteamID, o, SetSteamIDRequest{
		Target:    Target(target),
		DiscordID: discordID,
	})
}

func NewGetASNRecord(ipAddr string) Action {
	return New(GetASNRecord, Core, GetASNRecordRequest{IPAddr: ipAddr})
}

func NewGetLocationRecord(ipAddr string) Action {
	return New(GetLocationRecord, Core, GetLocationRecordRequest{IPAddr: ipAddr})
}

func NewGetProxyRecord(ipAddr string) Action {
	return New(GetProxyRecord, Core, GetProxyRecordRequest{IPAddr: ipAddr})
}

func NewSay(o Origin, server string, message string) Action {
	return New(Say, o, SayRequest{Server: server, Message: message})
}

func NewCSay(o Origin, server string, message string) Action {
	return New(CSay, o, CSayRequest{Server: server, Message: message})
}

func NewPSay(o Origin, target string, message string) Action {
	return New(PSay, o, PSayRequest{
		Message: message,
		Target:  Target(target),
	})
}

func NewServers() Action {
	return New(Servers, Core, nil)
}

func NewServerByName(serverID string) Action {
	return New(ServerByName, Core, ServerByNameRequest{ServerName: serverID})
}

func NewFilterAdd(o Origin, filter string) Action {
	return New(AddFilter, o, FilterAddRequest{Filter: filter})
}

func NewFilterDel(o Origin, filterID int) Action {
	return New(DelFilter, o, FilterDelRequest{FilterID: filterID})
}

func NewFilterCheck(o Origin, message string) Action {
	return New(CheckFilter, o, FilterCheckRequest{Message: message})
}

func NewGetOrCreatePersonByID(target string, ipAddr string) Action {
	return New(GetOrCreatePersonByID, Core, GetOrCreatePersonByIDRequest{
		Target: Target(target),
		IPAddr: ipAddr,
	})
}

func NewGetOrCreateProfileBySteamID(target string, ipAddr string) Action {
	return New(GetOrCreateProfileBySteamID, Core, GetOrCreateProfileBySteamIDRequest{
		Target: Target(target),
		IPAddr: ipAddr,
	})
}

func NewGetChatHistory(o Origin, target string, page int) Action {
	return New(GetChatHistory, o, GetChatHistoryRequest{
		Target: Target(target),
		Page:   page,
	})
}

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
