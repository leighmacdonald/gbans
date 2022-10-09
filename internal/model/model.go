// Package model defines common model structures used in many places throughout the application.
package model

import (
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"
)

type Linkable interface {
	ToURL() string
}

type PersonChat struct {
	PersonChatId int64
	SteamId      steamid.SID64
	ServerId     int
	TeamChat     bool
	Message      string
	CreatedOn    time.Time
}

// PersonIPRecord holds a composite result of the more relevant ip2location results
type PersonIPRecord struct {
	IPAddr      net.IP
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
	ServerID int `db:"server_id" json:"server_id"`
	// ServerNameShort is a short reference name for the server eg: us-1
	ServerNameShort string `db:"short_name" json:"server_name"`
	ServerNameLong  string `db:"server_name_long" json:"server_name_long"`
	// Token is the current valid authentication token that the server uses to make authenticated requests
	Token string `db:"token" json:"token"`
	// Address is the ip of the server
	Address string `db:"address" json:"address"`
	// Port is the port of the server
	Port int `db:"port" json:"port"`
	// RCON is the RCON password for the server
	RCON          string `db:"rcon" json:"rcon"`
	ReservedSlots int    `db:"reserved_slots" json:"reserved_slots"`
	// Password is what the server uses to generate a token to make authenticated calls
	Password   string  `db:"password" json:"password"`
	IsEnabled  bool    `json:"is_enabled"`
	Deleted    bool    `json:"deleted"`
	Region     string  `json:"region"`
	CC         string  `json:"cc"`
	Latitude   float32 `json:"latitude"`
	Longitude  float32 `json:"longitude"`
	DefaultMap string  `json:"default_map"`
	LogSecret  int     `json:"log_secret"`
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
type ServerStateCollection []ServerState

func (c ServerStateCollection) ByName(name string, state *ServerState) bool {
	for _, server := range c {
		if strings.EqualFold(server.NameShort, name) {
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
		ServerNameShort: name,
		Address:         address,
		Port:            port,
		RCON:            golib.RandomString(10),
		ReservedSlots:   0,
		Password:        golib.RandomString(20),
		DefaultMap:      "",
		IsEnabled:       true,
		TokenCreatedOn:  time.Unix(0, 0),
		CreatedOn:       config.Now(),
		UpdatedOn:       config.Now(),
	}
}

// ServerState contains the entire state for the servers. This
// contains sensitive information and should only be used where needed
// by admins.
type ServerState struct {
	// Database
	ServerId    int       `json:"server_id"`
	Name        string    `json:"name"`
	NameShort   string    `json:"name_short"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Enabled     bool      `json:"enabled"`
	Region      string    `json:"region"`
	CountryCode string    `json:"cc"`
	Latitude    float32   `json:"latitude"`
	Longitude   float32   `json:"longitude"`
	Reserved    int       `json:"reserved"`
	LastUpdate  time.Time `json:"last_update"`
	// A2S
	NameA2S  string `json:"name_a2s"` // The live name can differ from default
	Protocol uint8  `json:"protocol"`
	Map      string `json:"map"`
	// Name of the folder containing the game files.
	Folder string `json:"folder"`
	// Full name of the game.
	Game string `json:"game"`
	// Steam Application ID of game.
	AppId uint16 `json:"app_id"`
	// Number of players on the server.
	PlayerCount int `json:"player_count"`
	// Maximum number of players the server reports it can hold.
	MaxPlayers int `json:"max_players"`
	// Number of bots on the server.
	Bots int `json:"Bots"`
	// Indicates the type of server
	// Rag Doll Kung Fu servers always return 0 for "Server type."
	ServerType string `json:"server_type"`
	// Indicates the operating system of the server
	ServerOS string `json:"server_os"`
	// Indicates whether the server requires a password
	Password bool `json:"password"`
	// Specifies whether the server uses VAC
	VAC bool `json:"vac"`
	// Version of the game installed on the server.
	Version string `json:"version"`
	// Server's SteamID.
	SteamID steamid.SID64 `json:"steam_id"`
	// Tags that describe the game according to the server (for future use.)
	Keywords []string `json:"keywords"`
	// The server's 64-bit GameID. If this is present, a more accurate AppID is present in the low 24 bits. The earlier AppID could have been truncated as it was forced into 16-bit storage.
	GameID uint64 `json:"game_id"` // Needed?
	// Spectator port number for SourceTV.
	STVPort uint16 `json:"stv_port"`
	// Name of the spectator server for SourceTV.
	STVName string `json:"stv_name"`

	// RCON Sourced
	Players []ServerStatePlayer `json:"players"`
}

type ServerStatePlayer struct {
	UserID        int           `json:"user_id"`
	Name          string        `json:"name"`
	SID           steamid.SID64 `json:"steam_id"`
	ConnectedTime time.Duration `json:"connected_time"`
	State         string        `json:"state"`
	Ping          int           `json:"ping"`
	Loss          int           `json:"-"`
	IP            net.IP        `json:"-"`
	Port          int           `json:"-"`
}

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID          steamid.SID64 `db:"steam_id" json:"steam_id,string"`
	CreatedOn        time.Time     `json:"created_on"`
	UpdatedOn        time.Time     `json:"updated_on"`
	PermissionLevel  Privilege     `json:"permission_level"`
	Muted            bool          `json:"muted"`
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

func (p *Person) ToURL() string {
	return config.ExtURL("/profile/%d", p.SteamID.Int64())
}

// UserProfile is the model used in the webui representing the logged-in user.
type UserProfile struct {
	SteamID         steamid.SID64 `db:"steam_id" json:"steam_id,string"`
	CreatedOn       time.Time     `json:"created_on"`
	UpdatedOn       time.Time     `json:"updated_on"`
	PermissionLevel Privilege     `json:"permission_level"`
	DiscordID       string        `json:"discord_id"`
	Name            string        `json:"name"`
	Avatar          string        `json:"avatar"`
	AvatarFull      string        `json:"avatarfull"`
	BanID           int64         `json:"ban_id"`
	Muted           bool          `json:"muted"`
}

func (p UserProfile) ToURL() string {
	return config.ExtURL("/profile/%d", p.SteamID.Int64())
}

// LoggedIn checks for a valid steamID
func (p *Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

// AsTarget checks for a valid steamID
func (p *Person) AsTarget() StringSID {
	return StringSID(p.SteamID.String())
}

// NewPerson allocates a new default person instance
func NewPerson(sid64 steamid.SID64) Person {
	t0 := config.Now()
	return Person{
		SteamID:          sid64,
		CreatedOn:        t0,
		UpdatedOn:        t0,
		PermissionLevel:  PUser,
		Muted:            false,
		IsNew:            true,
		DiscordID:        "",
		IPAddr:           nil,
		CommunityBanned:  false,
		VACBans:          0,
		GameBans:         0,
		EconomyBan:       "none",
		DaysSinceLastBan: 0,
		UpdatedOnSteam:   t0,
		PlayerSummary:    &steamweb.PlayerSummary{},
	}
}

// NewUserProfile allocates a new default person instance
func NewUserProfile(sid64 steamid.SID64) UserProfile {
	t0 := config.Now()
	return UserProfile{
		SteamID:         sid64,
		CreatedOn:       t0,
		UpdatedOn:       t0,
		PermissionLevel: PUser,
		Name:            "Guest",
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

type Filter struct {
	WordID           int64      `json:"word_id,omitempty"`
	Patterns         []string   `json:"patterns,omitempty"`
	CreatedOn        time.Time  `json:"created_on"`
	UpdatedOn        time.Time  `json:"updated_on"`
	DiscordId        string     `json:"discord_id,omitempty"`
	DiscordCreatedOn *time.Time `json:"discord_created_on"`
	FilterName       string     `json:"filter_name"`
}

func (f *Filter) Match(value string) bool {
	for _, pattern := range f.Patterns {
		if util.GlobString(pattern, value) {
			return true
		}
	}
	return false
}

// RawLogEvent represents a full representation of a server log entry including all metadata attached to the log.
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
	Player  *ServerStatePlayer
	Server  *Server
	SteamID steamid.SID64
	InGame  bool
	Valid   bool
}

func NewPlayerInfo() PlayerInfo {
	return PlayerInfo{
		Player:  &ServerStatePlayer{},
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

//func NewDemoFile(serverId int64, title string, rawData []byte) (DemoFile, error) {
//	size := int64(len(rawData))
//	if size == 0 {
//		return DemoFile{}, errors.New("Empty demo")
//	}
//	return DemoFile{
//		ServerID:  serverId,
//		Title:     title,
//		Data:      rawData,
//		CreatedOn: config.Now(),
//		Size:      size,
//		Downloads: 0,
//	}, nil
//}

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

func (status ReportStatus) String() string {
	switch status {
	case ClosedWithoutAction:
		return "Closed without action"
	case ClosedWithAction:
		return "Closed with action"
	case Opened:
		return "Opened"
	default:
		return "Need more information"
	}
}

type Report struct {
	ReportId     int64         `json:"report_id"`
	AuthorId     steamid.SID64 `json:"author_id,string"`
	ReportedId   steamid.SID64 `json:"reported_id,string"`
	Description  string        `json:"description"`
	ReportStatus ReportStatus  `json:"report_status"`
	Reason       Reason        `json:"reason"`
	ReasonText   string        `json:"reason_text"`
	Deleted      bool          `json:"deleted"`
	CreatedOn    time.Time     `json:"created_on"`
	UpdatedOn    time.Time     `json:"updated_on"`
}

func (report Report) ToURL() string {
	return config.ExtURL("/report/%d", report.ReportId)
}

func NewReport() Report {
	return Report{
		ReportId:     0,
		AuthorId:     0,
		Description:  "",
		ReportStatus: 0,
		CreatedOn:    config.Now(),
		UpdatedOn:    config.Now(),
	}
}

type NewsEntry struct {
	NewsId      int       `json:"news_id"`
	Title       string    `json:"title"`
	BodyMD      string    `json:"body_md"`
	IsPublished bool      `json:"is_published"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
}

type UserUploadedFile struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}

type PersonConnection struct {
	PersonConnectionId int64          `json:"person_connection_id"`
	IPAddr             net.IP         `json:"ip_addr"`
	SteamId            steamid.SID64  `json:"steam_id,string"`
	PersonaName        string         `json:"persona_name"`
	CreatedOn          time.Time      `json:"created_on"`
	IPInfo             PersonIPRecord `json:"ip_info"`
}

type PersonConnections []PersonConnection

type PersonMessage struct {
	PersonMessageId int64         `json:"person_message_id"`
	SteamId         steamid.SID64 `json:"steam_id,string"`
	PersonaName     string        `json:"persona_name"`
	ServerName      string        `json:"server_name"`
	ServerId        int           `json:"server_id"`
	Body            string        `json:"body"`
	Team            bool          `json:"team"`
	CreatedOn       time.Time     `json:"created_on"`
}

type PersonMessages []PersonMessage

type Media struct {
	MediaId   int           `json:"media_id"`
	AuthorId  steamid.SID64 `json:"author_id,string"`
	MimeType  string        `json:"mime_type"`
	Contents  []byte        `json:"-"`
	Name      string        `json:"name"`
	Size      int64         `json:"size"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
}

const unknownMediaTag = "__unknown__"

var MediaSafeMimeTypesImages = []string{
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
}

func NewMedia(author steamid.SID64, name string, mime string, content []byte) Media {
	mtype := mimetype.Detect(content)
	if !mtype.Is(mime) && mime != unknownMediaTag {
		// Should never actually happen unless user is trying nefarious stuff.
		log.WithFields(log.Fields{"mime": mime, "detected": mtype.String()}).
			Warnf("Detected mimetype different than provided")
	}
	t0 := config.Now()
	return Media{
		AuthorId:  author,
		MimeType:  mtype.String(),
		Name:      strings.Replace(name, " ", "_", -1),
		Size:      int64(len(content)),
		Contents:  content,
		Deleted:   false,
		CreatedOn: t0,
		UpdatedOn: t0,
	}
}

type LocalTF2StatsSnapshot struct {
	StatId          int64          `json:"local_stats_players_stat_id"`
	Players         int            `json:"players"`
	CapacityFull    int            `json:"capacity_full"`
	CapacityEmpty   int            `json:"capacity_empty"`
	CapacityPartial int            `json:"capacity_partial"`
	MapTypes        map[string]int `json:"map_types"`
	Regions         map[string]int `json:"regions"`
	CreatedOn       time.Time      `json:"created_on"`
}

func (stats LocalTF2StatsSnapshot) TrimMapTypes() map[string]int {
	const minSize = 5
	out := map[string]int{}
	for k, v := range stats.MapTypes {
		mapKey := k
		if v < minSize {
			mapKey = "unknown"
		}
		out[mapKey] = v
	}
	return out
}

type GlobalTF2StatsSnapshot struct {
	StatId           int64          `json:"stat_id"`
	Players          int            `json:"players"`
	Bots             int            `json:"bots"`
	Secure           int            `json:"secure"`
	ServersCommunity int            `json:"servers_community"`
	ServersTotal     int            `json:"servers_total"`
	CapacityFull     int            `json:"capacity_full"`
	CapacityEmpty    int            `json:"capacity_empty"`
	CapacityPartial  int            `json:"capacity_partial"`
	MapTypes         map[string]int `json:"map_types"`
	Regions          map[string]int `json:"regions"`
	CreatedOn        time.Time      `json:"created_on"`
}

func (stats GlobalTF2StatsSnapshot) TrimMapTypes() map[string]int {
	const minSize = 5
	out := map[string]int{}
	for k, v := range stats.MapTypes {
		mapKey := k
		if v < minSize {
			mapKey = "unknown"
		}
		out[mapKey] = v
	}
	return out
}

func NewGlobalTF2Stats() GlobalTF2StatsSnapshot {
	return GlobalTF2StatsSnapshot{
		MapTypes:  map[string]int{},
		Regions:   map[string]int{},
		CreatedOn: config.Now(),
	}
}
func NewLocalTF2Stats() LocalTF2StatsSnapshot {
	return LocalTF2StatsSnapshot{
		MapTypes:  map[string]int{},
		Regions:   map[string]int{},
		CreatedOn: config.Now(),
	}
}

type ServerLocation struct {
	ip2location.LatLong
	steamweb.Server
}
