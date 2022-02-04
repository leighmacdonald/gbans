// Package model defines common model structures used in many places throughout the application.
package model

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	"net"
	"regexp"
	"strings"
	"time"
)

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

// BanType defines the state of the ban for a user, 0 being no ban
type BanType int

const (
	// Unknown means the ban state could not be determined, failing-open to allowing players
	// to connect.
	Unknown BanType = -1
	// OK Ban state is clean
	OK BanType = 0
	// NoComm means the player cannot communicate while playing voice + chat
	NoComm BanType = 1
	// Banned means the player cannot join the server at all
	Banned BanType = 2
)

// Origin defines the origin of the ban or action
type Origin int

const (
	// System is an automatic ban triggered by the service
	System Origin = 0
	// Bot is a ban using the discord bot interface
	Bot Origin = 1
	// Web is a ban using the web-ui
	Web Origin = 2
	// InGame is a ban using the sourcemod plugin
	InGame Origin = 3
)

func (s Origin) String() string {
	switch s {
	case System:
		return "System"
	case Bot:
		return "Bot"
	case Web:
		return "Web"
	case InGame:
		return "In-Game"
	default:
		return "Unknown"
	}
}

// Reason defined a set of predefined ban reasons
// TODO make this fully dynamic?
type Reason int

const (
	Custom           Reason = 1
	External         Reason = 2
	Cheating         Reason = 3
	Racism           Reason = 4
	Harassment       Reason = 5
	Exploiting       Reason = 6
	WarningsExceeded Reason = 7
	Spam             Reason = 8
	Language         Reason = 9
)

var reasonStr = map[Reason]string{
	Custom:           "",
	External:         "3rd party",
	Cheating:         "Cheating",
	Racism:           "Racism",
	Harassment:       "Person Harassment",
	Exploiting:       "Exploiting",
	WarningsExceeded: "Warnings Exceeding",
	Spam:             "Spam",
	Language:         "Language",
}

func (r Reason) String() string {
	return reasonStr[r]
}

// LogPayload is the container for log/message payloads
type LogPayload struct {
	ServerName string `json:"server_name"`
	Message    string `json:"message"`
}

type BanASN struct {
	BanASNId   int64
	ASNum      int64
	Origin     Origin
	AuthorID   steamid.SID64
	TargetID   steamid.SID64
	Reason     string
	ValidUntil time.Time
	CreatedOn  time.Time
	UpdatedOn  time.Time
}

func NewBanASN(asn int64, authorId steamid.SID64, reason string, duration time.Duration) BanASN {
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return BanASN{
		ASNum:      asn,
		Origin:     System,
		AuthorID:   authorId,
		TargetID:   0,
		Reason:     reason,
		ValidUntil: config.Now().Add(duration),
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
}

type BanNet struct {
	NetID      int64         `db:"net_id"`
	SteamID    steamid.SID64 `db:"steam_id"`
	AuthorID   steamid.SID64 `db:"author_id"`
	CIDR       *net.IPNet    `db:"cidr"`
	Source     Origin        `db:"source"`
	Reason     string        `db:"reason"`
	CreatedOn  time.Time     `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time     `db:"updated_on" json:"updated_on"`
	ValidUntil time.Time     `db:"valid_until"`
}

func NewBan(steamID steamid.SID64, authorID steamid.SID64, duration time.Duration) Ban {
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return Ban{
		SteamID:    steamID,
		AuthorID:   authorID,
		BanType:    Banned,
		Reason:     Custom,
		ReasonText: "Unspecified",
		Note:       "",
		Source:     System,
		ValidUntil: config.Now().Add(duration),
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
}

func NewBanNet(cidr string, reason string, duration time.Duration, source Origin) (BanNet, error) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return BanNet{}, err
	}
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return BanNet{
		CIDR:       n,
		Source:     source,
		Reason:     reason,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
		ValidUntil: config.Now().Add(duration),
	}, nil
}

func (b BanNet) String() string {
	return fmt.Sprintf("Net: %s Origin: %s Reason: %s", b.CIDR, b.Source, b.Reason)
}

type Ban struct {
	BanID uint64 `db:"ban_id" json:"ban_id"`
	// SteamID is the steamID of the banned person
	SteamID  steamid.SID64 `db:"steam_id" json:"steam_id"`
	AuthorID steamid.SID64 `db:"author_id" json:"author_id"`
	// Reason defines the overall ban classification
	BanType BanType `db:"ban_type" json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `db:"reason" json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText string `db:"reason_text" json:"reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note   string `db:"note" json:"note"`
	Source Origin `json:"ban_source" db:"ban_source"`
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" db:"valid_until"`
	CreatedOn  time.Time `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time `db:"updated_on" json:"updated_on"`
}

func (b Ban) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v",
		b.SteamID.Int64(), b.Source, b.ReasonText, b.BanType)
}

type BannedPerson struct {
	Ban                Ban               `json:"ban"`
	Person             Person            `json:"person"`
	HistoryChat        []logparse.SayEvt `json:"history_chat" db:"-"`
	HistoryPersonaName []string          `json:"history_personaname" db:"-"`
	HistoryConnections []string          `json:"history_connections" db:"-"`
	HistoryIP          []PersonIPRecord  `json:"history_ip" db:"-"`
}

func NewBannedPerson() BannedPerson {
	return BannedPerson{
		Ban: Ban{
			CreatedOn: config.Now(),
			UpdatedOn: config.Now(),
		},
		Person: Person{
			CreatedOn:     config.Now(),
			UpdatedOn:     config.Now(),
			PlayerSummary: &steamweb.PlayerSummary{},
		},
		HistoryChat:        nil,
		HistoryPersonaName: nil,
		HistoryConnections: nil,
		HistoryIP:          nil,
	}
}

type ChatLog struct {
	Message   string
	CreatedOn time.Time
}

type IPRecord struct {
	IPAddr    net.IP    `json:"ip_addr"`
	CreatedOn time.Time `json:"created_on"`
}

// PersonIPRecord holds a composite result of the more relevant ip2location results
type PersonIPRecord struct {
	IP          net.IP
	CreatedOn   time.Time
	CityName    string
	CountryName string
	CountryCode string
	ASName      string
	ASNum       int
	ISP         string
	UsageType   string
	Threat      string
	DomainUsed  string
}

type Server struct {
	// Auto generated id
	ServerID int64 `db:"server_id" json:"server_id"`
	// ServerName is a short reference name for the server eg: us-1
	ServerName     string `db:"short_name" json:"server_name"`
	ServerNameLong string `db:"server_name_long" json:"server_name_long"`
	// Token is the current valid authentication token that the server uses to make authenticated requests
	Token string `db:"token" json:"token"`
	// Address is the ip of the server
	Address string `db:"address" json:"address"`
	// Port is the port of the server
	Port int `db:"port" json:"port"`
	// RCON is the RCON password for the server
	RCON          string `db:"rcon" json:"-"`
	ReservedSlots int    `db:"reserved_slots" json:"reserved_slots"`
	// Password is what the server uses to generate a token to make authenticated calls
	Password   string              `db:"password" json:"password"`
	IsEnabled  bool                `json:"is_enabled"`
	Deleted    bool                `json:"deleted"`
	Region     string              `json:"region"`
	CC         string              `json:"cc"`
	Location   ip2location.LatLong `json:"location"`
	DefaultMap string              `json:"default_map"`
	// TokenCreatedOn is set when changing the token
	TokenCreatedOn time.Time `db:"token_created_on" json:"token_created_on"`
	CreatedOn      time.Time `db:"created_on" json:"created_on"`
	UpdatedOn      time.Time `db:"updated_on" json:"updated_on"`
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}

// TODO move findPlayerBy* methods to here
type ServerStateCollection map[string]ServerState

func (c ServerStateCollection) ByName(name string, state *ServerState) bool {
	for _, server := range c {
		if strings.EqualFold(server.Name, name) {
			*state = server
			return true
		}
	}
	return false
}

func (c ServerStateCollection) ByRegion() map[string][]ServerState {
	rm := map[string][]ServerState{}
	for serverId, server := range c {
		_, exists := rm[server.Region]
		if !exists {
			rm[server.Region] = []ServerState{}
		}
		rm[server.Region] = append(rm[server.Region], c[serverId])
	}
	return rm
}

func NewServer(name string, address string, port int) Server {
	return Server{
		ServerName:     name,
		Address:        address,
		Port:           port,
		RCON:           golib.RandomString(10),
		ReservedSlots:  0,
		Password:       golib.RandomString(20),
		DefaultMap:     "",
		IsEnabled:      true,
		TokenCreatedOn: time.Unix(0, 0),
		CreatedOn:      config.Now(),
		UpdatedOn:      config.Now(),
	}
}

type ServerState struct {
	NameLong    string
	Name        string
	Host        string
	Enabled     bool
	Region      string
	CountryCode string
	Reserved    int
	A2S         a2s.ServerInfo
	Status      extra.Status
	Players     []extra.Player
	LastUpdate  time.Time
}

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID          steamid.SID64 `db:"steam_id" json:"steam_id,string"`
	CreatedOn        time.Time     `json:"created_on"`
	UpdatedOn        time.Time     `json:"updated_on"`
	PermissionLevel  Privilege     `json:"permission_level"`
	IsNew            bool          `json:"-"`
	DiscordID        string        `json:"discord_id"`
	IPAddr           net.IP        `json:"-"` // TODO Allow json for admins endpoints
	CommunityBanned  bool          `json:"community_banned"`
	VACBans          int           `json:"vac_bans"`
	GameBans         int           `json:"game_bans"`
	EconomyBan       string        `json:"economy_ban"`
	DaysSinceLastBan int           `json:"days_since_last_ban"`
	UpdatedOnSteam   time.Time     `json:"updated_on_steam"`
	*steamweb.PlayerSummary
}

// LoggedIn checks for a valid steamID
func (p *Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

// AsTarget checks for a valid steamID
func (p *Person) AsTarget() Target {
	return Target(p.SteamID.String())
}

// NewPerson allocates a new default person instance
func NewPerson(sid64 steamid.SID64) Person {
	t0 := config.Now()
	return Person{
		SteamID:         sid64,
		IsNew:           true,
		CreatedOn:       t0,
		UpdatedOn:       t0,
		PlayerSummary:   &steamweb.PlayerSummary{},
		PermissionLevel: PAuthenticated,
	}
}

type People []Person

func (p People) AsMap() map[steamid.SID64]Person {
	m := map[steamid.SID64]Person{}
	for _, person := range p {
		m[person.SteamID] = person
	}
	return m
}

// AppealState is the current state of a users ban appeal, if any.
type AppealState int

const (
	// ASNew is a user has initiated an appeal
	ASNew AppealState = 0
	// ASDenied the appeal was denied
	ASDenied AppealState = 1
	// The appeal was granted
	//ASGranted AppealState = 2
)

type Appeal struct {
	AppealID    int         `db:"appeal_id" json:"appeal_id"`
	BanID       uint64      `db:"ban_id" json:"ban_id"`
	AppealText  string      `db:"appeal_text" json:"appeal_text"`
	AppealState AppealState `db:"appeal_state" json:"appeal_state"`
	Email       string      `db:"email" json:"email"`
	CreatedOn   time.Time   `db:"created_on" json:"created_on"`
	UpdatedOn   time.Time   `db:"updated_on" json:"updated_on"`
}

type Stats struct {
	BansTotal     int `json:"bans"`
	BansDay       int `json:"bans_day"`
	BansWeek      int `json:"bans_week"`
	BansMonth     int `json:"bans_month"`
	Bans3Month    int `json:"bans_3month"`
	Bans6Month    int `json:"bans_6month"`
	BansYear      int `json:"bans_year"`
	BansCIDRTotal int `json:"bans_cidr"`
	AppealsOpen   int `json:"appeals_open"`
	AppealsClosed int `json:"appeals_closed"`
	FilteredWords int `json:"filtered_words"`
	ServersAlive  int `json:"servers_alive"`
	ServersTotal  int `json:"servers_total"`
}

// ServerEvent is a flat struct encapsulating a parsed log event
// Fields being present is event dependent, so do not assume everything will be
// available
type ServerEvent struct {
	LogID int64 `json:"log_id"`
	// Server is where the event happened
	Server    *Server            `json:"server"`
	EventType logparse.EventType `json:"event_type"`
	// Source is the player or thing initiating the event/action
	Source *Person `json:"source"`
	// Target is the optional target of an event created by a Source
	Target *Person `json:"target"`
	// PlayerClass is the last known class the player was as tracked by the playerStateCache OR the class that
	// a player switch to in the case of a spawned_as event
	PlayerClass logparse.PlayerClass `json:"player_class"`
	// Weapon is the weapon used to perform certain events
	Weapon logparse.Weapon `json:"weapon"`
	// Damage is how much (real) damage or in the case of medi-guns, healing
	Damage  int64 `json:"damage"`
	Healing int64 `json:"healing"`
	// Item is the item a player picked up
	Item logparse.PickupItem `json:"item"`
	// AttackerPOS is the 3d position of the source of the event
	AttackerPOS logparse.Pos `json:"attacker_pos"`
	VictimPOS   logparse.Pos `json:"victim_pos"`
	AssisterPOS logparse.Pos `json:"assister_pos"`
	// Team is the last known team the player was as tracked by the playerStateCache OR the team that
	// a player switched to in a join_team event
	Team      logparse.Team  `json:"team"`
	CreatedOn time.Time      `json:"created_on"`
	MetaData  map[string]any `json:"meta_data"`
}

func NewServerEvent() ServerEvent {
	return ServerEvent{
		Server:      &Server{},
		Source:      &Person{PlayerSummary: &steamweb.PlayerSummary{}},
		Target:      &Person{PlayerSummary: &steamweb.PlayerSummary{}},
		AssisterPOS: logparse.Pos{},
		AttackerPOS: logparse.Pos{},
		VictimPOS:   logparse.Pos{},
	}
}

type Filter struct {
	WordID    int
	Pattern   *regexp.Regexp
	CreatedOn time.Time
}

func (f *Filter) Match(value string) bool {
	return f.Pattern.MatchString(value)
}

// RawLogEvent represents a full representation of a server log entry including all meta data attached to the log.
type RawLogEvent struct {
	LogID     int64              `json:"log_id"`
	Type      logparse.EventType `json:"event_type"`
	Event     map[string]string  `json:"event"`
	Server    Server             `json:"server"`
	Player1   *Person            `json:"player1"`
	Player2   *Person            `json:"player2"`
	Assister  *Person            `json:"assister"`
	RawEvent  string             `json:"raw_event"`
	CreatedOn time.Time          `json:"created_on"`
}

// Unmarshal is just a helper to
func (e *RawLogEvent) Unmarshal(output any) error {
	return logparse.Unmarshal(e.Event, output)
}

type PlayerInfo struct {
	Player  *extra.Player
	Server  *Server
	SteamID steamid.SID64
	InGame  bool
	Valid   bool
}

func NewPlayerInfo() PlayerInfo {
	return PlayerInfo{
		Player:  &extra.Player{},
		Server:  nil,
		SteamID: 0,
		InGame:  false,
		Valid:   false,
	}
}

type LogQueryOpts struct {
	LogTypes   []logparse.EventType `json:"log_types"`
	Limit      uint64               `json:"limit"`
	OrderDesc  bool                 `json:"order_desc"`
	Query      string               `json:"query"`
	SourceID   string               `json:"source_id"`
	TargetID   string               `json:"target_id"`
	Servers    []int                `json:"servers"`
	Network    string               `json:"network"`
	SentBefore *time.Time           `json:"sent_before,omitempty"`
	SentAfter  *time.Time           `json:"sent_after,omitempty"`
}

func (lqo *LogQueryOpts) ValidRecordType(t logparse.EventType) bool {
	if len(lqo.LogTypes) == 0 {
		// No filters == Any
		return true
	}
	for _, mt := range lqo.LogTypes {
		if mt == t {
			return true
		}
	}
	return false
}

type BDIds struct {
	FileInfo struct {
		Authors     []string `json:"authors"`
		Description string   `json:"description"`
		Title       string   `json:"title"`
		UpdateURL   string   `json:"update_url"`
	} `json:"file_info"`
	Schema  string `json:"$schema"`
	Players []struct {
		Steamid    int64    `json:"steamid"`
		Attributes []string `json:"attributes"`
		LastSeen   struct {
			PlayerName string `json:"player_name"`
			Time       int    `json:"time"`
		} `json:"last_seen"`
	} `json:"players"`
	Version int `json:"version"`
}

type DemoFile struct {
	DemoID    int64     `json:"demo_id"`
	ServerID  int64     `json:"server_id"`
	Title     string    `json:"title"`
	Data      []byte    `json:"-"` // Dont send mega data to frontend by accident
	CreatedOn time.Time `json:"created_on"`
	Size      int64     `json:"size"`
	Downloads int64     `json:"downloads"`
}

func NewDemoFile(serverId int64, title string, rawData []byte) (DemoFile, error) {
	size := int64(len(rawData))
	if size == 0 {
		return DemoFile{}, errors.New("Empty demo")
	}
	return DemoFile{
		ServerID:  serverId,
		Title:     title,
		Data:      rawData,
		CreatedOn: config.Now(),
		Size:      size,
		Downloads: 0,
	}, nil
}

// CommonStats contains shared stats that are used across all models
type CommonStats struct {
	Kills        int64 `json:"kills"`
	Assists      int64 `json:"assists"`
	Damage       int64 `json:"damage"`
	Healing      int64 `json:"healing"`
	Shots        int64 `json:"shots"`
	Hits         int64 `json:"hits"`
	Suicides     int64 `json:"suicides"`
	Extinguishes int64 `json:"extinguishes"`

	PointCaptures int64 `json:"point_captures"`
	PointDefends  int64 `json:"point_defends"`

	MedicDroppedUber int64 `json:"medic_dropped_uber"`

	ObjectBuilt     int64 `json:"object_built"`
	ObjectDestroyed int64 `json:"object_destroyed"`

	Messages     int64 `json:"messages"`
	MessagesTeam int64 `json:"messages_team"`

	PickupAmmoLarge  int64 `json:"pickup_ammo_large"`
	PickupAmmoMedium int64 `json:"pickup_ammo_medium"`
	PickupAmmoSmall  int64 `json:"pickup_ammo_small"`
	PickupHPLarge    int64 `json:"pickup_hp_large"`
	PickupHPMedium   int64 `json:"pickup_hp_medium"`
	PickupHPSmall    int64 `json:"pickup_hp_small"`

	SpawnScout    int64 `json:"spawn_scout"`
	SpawnSoldier  int64 `json:"spawn_soldier"`
	SpawnPyro     int64 `json:"spawn_pyro"`
	SpawnDemo     int64 `json:"spawn_demo"`
	SpawnHeavy    int64 `json:"spawn_heavy"`
	SpawnEngineer int64 `json:"spawn_engineer"`
	SpawnMedic    int64 `json:"spawn_medic"`
	SpawnSniper   int64 `json:"spawn_sniper"`
	SpawnSpy      int64 `json:"spawn_spy"`

	Dominations int64 `json:"dominations"`
	Revenges    int64 `json:"revenges"`

	Playtime   time.Duration `json:"playtime"`
	EventCount int64         `json:"event_count"`
}

type GlobalStats struct {
	CommonStats
	UniquePlayers int64 `json:"unique_players"`
}

type MapStats struct {
	CommonStats
}

type PlayerStats struct {
	CommonStats
	Deaths       int64 `json:"deaths"`
	Games        int64 `json:"games"`
	Wins         int64 `json:"wins"`
	Losses       int64 `json:"losses"`
	DamageTaken  int64 `json:"damage_taken"`
	Dominated    int64 `json:"dominated"`
	HealingTaken int64 `json:"healing_taken"`
}

type ServerStats struct {
	CommonStats
}

type ReportStatus int

const (
	Opened ReportStatus = iota
	NeedMoreInfo
	ClosedWithoutAction
	ClosedWithAction
)

type ReportMedia struct {
	ReportMediaId int           `json:"report_media_id"`
	ReportId      int           `json:"report_id"`
	AuthorId      steamid.SID64 `json:"author_id"`
	MimeType      string        `json:"mime_type"`
	Size          int64         `json:"size"`
	Contents      []byte        `json:"contents"`
	Deleted       bool          `json:"deleted"`
	CreatedOn     time.Time     `json:"created_on"`
	UpdatedOn     time.Time     `json:"updated_on"`
}

type ReportMessage struct {
	ReportMessageId int           `json:"report_message_id"`
	ReportId        int           `json:"report_id"`
	AuthorId        steamid.SID64 `json:"author_id"`
	Message         string        `json:"contents"`
	Deleted         bool          `json:"deleted"`
	CreatedOn       time.Time     `json:"created_on"`
	UpdatedOn       time.Time     `json:"updated_on"`
}

type Report struct {
	ReportId     int           `json:"report_id"`
	AuthorId     steamid.SID64 `json:"author_id"`
	ReportedId   steamid.SID64 `json:"reported_id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	ReportStatus ReportStatus  `json:"report_status"`
	Deleted      bool          `json:"deleted"`
	CreatedOn    time.Time     `json:"created_on"`
	UpdatedOn    time.Time     `json:"updated_on"`
	MediaIds     []int         `json:"media_ids"`
}

func NewReport() Report {
	return Report{
		ReportId:     0,
		AuthorId:     0,
		Title:        "",
		Description:  "",
		ReportStatus: 0,
		CreatedOn:    config.Now(),
		UpdatedOn:    config.Now(),
	}
}

func NewReportMedia(reportId int) ReportMedia {
	return ReportMedia{
		ReportMediaId: 0,
		ReportId:      reportId,
		AuthorId:      0,
		MimeType:      "",
		Contents:      nil,
		CreatedOn:     config.Now(),
		UpdatedOn:     config.Now(),
	}
}

func NewReportMessage(reportId int, authorId steamid.SID64, message string) ReportMessage {
	return ReportMessage{
		ReportMessageId: 0,
		ReportId:        reportId,
		AuthorId:        authorId,
		Message:         message,
		CreatedOn:       config.Now(),
		UpdatedOn:       config.Now(),
	}
}
