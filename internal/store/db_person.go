package store

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
)

type UserNotification struct {
	NotificationId int64                       `json:"person_notification_id"`
	SteamId        steamid.SID64               `json:"steam_id,string"`
	Read           bool                        `json:"read"`
	Deleted        bool                        `json:"deleted"`
	Severity       consts.NotificationSeverity `json:"severity"`
	Message        string                      `json:"message"`
	Link           string                      `json:"link"`
	Count          int                         `json:"count"`
	CreatedOn      time.Time                   `json:"created_on"`
}

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID          steamid.SID64         `db:"steam_id" json:"steam_id,string"`
	CreatedOn        time.Time             `json:"created_on"`
	UpdatedOn        time.Time             `json:"updated_on"`
	PermissionLevel  consts.Privilege      `json:"permission_level"`
	Muted            bool                  `json:"muted"`
	IsNew            bool                  `json:"-"`
	DiscordID        string                `json:"discord_id"`
	IPAddr           net.IP                `json:"-"` // TODO Allow json for admins endpoints
	CommunityBanned  bool                  `json:"community_banned"`
	VACBans          int                   `json:"vac_bans"`
	GameBans         int                   `json:"game_bans"`
	EconomyBan       steamweb.EconBanState `json:"economy_ban"`
	DaysSinceLastBan int                   `json:"days_since_last_ban"`
	UpdatedOnSteam   time.Time             `json:"updated_on_steam"`
	*steamweb.PlayerSummary
}

func (p *Person) ToURL() string {
	return config.ExtURL("/profile/%d", p.SteamID.Int64())
}

// LoggedIn checks for a valid steamID.
func (p *Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

// AsTarget checks for a valid steamID.
func (p *Person) AsTarget() StringSID {
	return StringSID(p.SteamID.String())
}

// NewPerson allocates a new default person instance.
func NewPerson(sid64 steamid.SID64) Person {
	t0 := config.Now()
	return Person{
		SteamID:          sid64,
		CreatedOn:        t0,
		UpdatedOn:        t0,
		PermissionLevel:  consts.PUser,
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
		PlayerSummary: &steamweb.PlayerSummary{
			SteamID: sid64,
		},
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

type PersonChat struct {
	PersonChatId int64
	SteamId      steamid.SID64
	ServerId     int
	TeamChat     bool
	Message      string
	CreatedOn    time.Time
}

// PersonIPRecord holds a composite result of the more relevant ip2location results.
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

type AppealOverview struct {
	BanSteam

	SourceSteamId     steamid.SID64 `json:"source_steam_id"`
	SourcePersonaName string        `json:"source_persona_name"`
	SourceAvatar      string        `json:"source_avatar"`
	SourceAvatarFull  string        `json:"source_avatar_full"`

	TargetSteamId     steamid.SID64 `json:"target_steam_id"`
	TargetPersonaName string        `json:"target_persona_name"`
	TargetAvatar      string        `json:"target_avatar"`
	TargetAvatarFull  string        `json:"target_avatar_full"`
}

type UserMessage struct {
	ParentId  int64         `json:"parent_id"`
	MessageId int64         `json:"message_id"`
	AuthorId  steamid.SID64 `json:"author_id,string"`
	Message   string        `json:"contents"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
}

func NewUserMessage(parentId int64, authorId steamid.SID64, message string) UserMessage {
	return UserMessage{
		ParentId:  parentId,
		AuthorId:  authorId,
		Message:   message,
		CreatedOn: config.Now(),
		UpdatedOn: config.Now(),
	}
}

type PersonAuth struct {
	PersonAuthId int64         `json:"person_auth_id"`
	SteamId      steamid.SID64 `json:"steam_id"`
	IpAddr       net.IP        `json:"ip_addr"`
	RefreshToken string        `json:"refresh_token"`
	CreatedOn    time.Time     `json:"created_on"`
}

const refreshTokenLen = 80

func NewPersonAuth(sid64 steamid.SID64, addr net.IP) PersonAuth {
	return PersonAuth{
		PersonAuthId: 0,
		SteamId:      sid64,
		IpAddr:       addr,
		RefreshToken: golib.RandomString(refreshTokenLen),
		CreatedOn:    config.Now(),
	}
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

func DropPerson(ctx context.Context, steamID steamid.SID64) error {
	query, args, errQueryArgs := sb.Delete("person").Where(sq.Eq{"steam_id": steamID}).ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	return nil
}

// SavePerson will insert or update the person record.
func SavePerson(ctx context.Context, person *Person) error {
	person.UpdatedOn = config.Now()
	// FIXME
	if person.PermissionLevel == 0 {
		person.PermissionLevel = 10
	}
	if !person.IsNew {
		return updatePerson(ctx, person)
	}
	person.CreatedOn = person.UpdatedOn
	return insertPerson(ctx, person)
}

func updatePerson(ctx context.Context, person *Person) error {
	person.UpdatedOn = config.Now()
	const query = `
		UPDATE person 
		SET 
		    updated_on = $2, communityvisibilitystate = $3, profilestate = $4, personaname = $5, profileurl = $6, avatar = $7,
    		avatarmedium = $8, avatarfull = $9, avatarhash = $10, personastate = $11, realname = $12, timecreated = $13,
		    loccountrycode = $14, locstatecode = $15, loccityid = $16, permission_level = $17, discord_id = $18,
		    community_banned = $19, vac_bans = $20, game_bans = $21, economy_ban = $22, days_since_last_ban = $23,
			updated_on_steam = $24, muted = $25
		WHERE steam_id = $1`
	if errExec := Exec(ctx, query, person.SteamID, person.UpdatedOn,
		person.PlayerSummary.CommunityVisibilityState, person.PlayerSummary.ProfileState,
		person.PlayerSummary.PersonaName, person.PlayerSummary.ProfileURL, person.PlayerSummary.Avatar,
		person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull, person.PlayerSummary.AvatarHash,
		person.PlayerSummary.PersonaState, person.PlayerSummary.RealName, person.TimeCreated,
		person.PlayerSummary.LocCountryCode, person.PlayerSummary.LocStateCode, person.PlayerSummary.LocCityID,
		person.PermissionLevel, person.DiscordID, person.CommunityBanned, person.VACBans, person.GameBans,
		person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam, person.Muted); errExec != nil {
		return Err(errExec)
	}
	return nil
}

func insertPerson(ctx context.Context, person *Person) error {
	query, args, errQueryArgs := sb.
		Insert("person").
		Columns("created_on", "updated_on", "steam_id", "communityvisibilitystate", "profilestate",
			"personaname", "profileurl", "avatar", "avatarmedium", "avatarfull", "avatarhash", "personastate",
			"realname", "timecreated", "loccountrycode", "locstatecode", "loccityid", "permission_level",
			"discord_id", "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted").
		Values(person.CreatedOn, person.UpdatedOn, person.SteamID, person.PlayerSummary.CommunityVisibilityState,
			person.PlayerSummary.ProfileState, person.PlayerSummary.PersonaName, person.PlayerSummary.ProfileURL,
			person.PlayerSummary.Avatar, person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull,
			person.PlayerSummary.AvatarHash, person.PlayerSummary.PersonaState, person.PlayerSummary.RealName,
			person.PlayerSummary.TimeCreated, person.PlayerSummary.LocCountryCode, person.PlayerSummary.LocStateCode,
			person.PlayerSummary.LocCityID, person.PermissionLevel, person.DiscordID, person.CommunityBanned,
			person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam,
			person.Muted).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	errExec := Exec(ctx, query, args...)
	if errExec != nil {
		return Err(errExec)
	}
	person.IsNew = false
	return nil
}

// "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban".
var profileColumns = []string{
	"steam_id", "created_on", "updated_on",
	"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
	"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
	"loccountrycode", "locstatecode", "loccityid", "permission_level", "discord_id",
	"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban", "updated_on_steam",
	"muted",
}

// GetPersonBySteamID returns a person by their steam_id. ErrNoResult is returned if the steam_id
// is not known.
func GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *Person) error {
	const query = `
    	SELECT p.steam_id,
			p.created_on,
			p.updated_on,
			p.communityvisibilitystate,
			p.profilestate,
			p.personaname,
			p.profileurl,
			p.avatar,
			p.avatarmedium,
			p.avatarfull,
			p.avatarhash,
			p.personastate,
			p.realname,
			p.timecreated,
			p.loccountrycode,
			p.locstatecode,
			p.loccityid,
			p.permission_level,
			p.discord_id,
			/*		   //(
			//   SELECT (e.meta_data ->> 'address')::inet
			//   FROM server_log e
			//   WHERE e.event_type = 1004
			//	 AND e.source_id = person.steam_id
			//   ORDER BY e.created_on DESC
			//   LIMIT 1
			//),*/
			p.community_banned,
			p.vac_bans,
			p.game_bans,
			p.economy_ban,
			p.days_since_last_ban,
			p.updated_on_steam,
			p.muted
	FROM person p
	WHERE p.steam_id = $1;`
	if !sid64.Valid() {
		return consts.ErrInvalidSID
	}
	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	errQuery := QueryRow(ctx, query, sid64.Int64()).Scan(&person.SteamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID /*&person.IPAddr,*/, &person.CommunityBanned,
		&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam,
		&person.Muted)
	if errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

// TODO search cached people first?
func GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (People, error) {
	queryBuilder := sb.Select(profileColumns...).From("person").Where(sq.Eq{"steam_id": fp.Uniq[steamid.SID64](steamIds)})
	query, args, errQueryArgs := queryBuilder.ToSql()
	if errQueryArgs != nil {
		return nil, errQueryArgs
	}
	var people People
	rows, errQuery := Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		person := NewPerson(0)
		if errScan := rows.Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel, &person.DiscordID,
			&person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
			&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errScan
		}
		people = append(people, person)
	}
	return people, nil
}

func GetPeople(ctx context.Context, queryFilter QueryFilter) (People, error) {
	queryBuilder := sb.Select(profileColumns...).From("person")
	if queryFilter.Query != "" {
		// TODO add lower-cased functional index to avoid tableName scan
		queryBuilder = queryBuilder.Where(sq.ILike{"personaname": strings.ToLower(queryFilter.Query)})
	}
	if queryFilter.Offset > 0 {
		queryBuilder = queryBuilder.Offset(queryFilter.Offset)
	}
	if queryFilter.OrderBy != "" {
		queryBuilder = queryBuilder.OrderBy(queryFilter.orderString())
	}
	if queryFilter.Limit == 0 {
		queryBuilder = queryBuilder.Limit(100)
	} else {
		queryBuilder = queryBuilder.Limit(queryFilter.Limit)
	}
	query, args, errQueryArgs := queryBuilder.ToSql()
	if errQueryArgs != nil {
		return nil, errQueryArgs
	}
	var people People
	rows, errQuery := Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		person := NewPerson(0)
		if errScan := rows.Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar,
			&person.AvatarMedium, &person.AvatarFull, &person.AvatarHash, &person.PersonaState,
			&person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
			&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
			&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
			&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errScan
		}
		people = append(people, person)
	}
	return people, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *Person) error {
	errGetPerson := GetPersonBySteamID(ctx, sid64, person)
	if errGetPerson != nil && Err(errGetPerson) == ErrNoResult {
		// FIXME
		newPerson := NewPerson(sid64)
		*person = newPerson
		return SavePerson(ctx, person)
	}
	return errGetPerson
}

// GetPersonByDiscordID returns a person by their discord_id.
func GetPersonByDiscordID(ctx context.Context, discordId string, person *Person) error {
	query, args, errQueryArgs := sb.Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": discordId}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	errQuery := QueryRow(ctx, query, args...).Scan(&person.SteamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans,
		&person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted)
	if errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func GetExpiredProfiles(ctx context.Context, limit uint64) ([]Person, error) {
	query, args, errArgs := sb.
		Select(profileColumns...).
		From("person").
		OrderBy("updated_on").
		Limit(limit).
		ToSql()
	if errArgs != nil {
		return nil, Err(errArgs)
	}
	var people []Person
	rows, errQuery := Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		person := NewPerson(0)
		if errScan := rows.Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel,
			&person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans,
			&person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, Err(errScan)
		}
		people = append(people, person)
	}
	return people, nil
}

func AddChatHistory(ctx context.Context, message *PersonMessage) error {
	const q = `INSERT INTO person_messages 
    		(steam_id, server_id, body, team, created_on, persona_name) 
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING person_message_id`
	if errScan := QueryRow(ctx, q, message.SteamId, message.ServerId, message.Body, message.Team,
		message.CreatedOn, message.PersonaName).
		Scan(&message.PersonMessageId); errScan != nil {
		return Err(errScan)
	}
	return nil
}

func GetPersonMessageById(ctx context.Context, personMessageId int64, msg *PersonMessage) error {
	query, args, errQuery := sb.Select(
		"m.person_message_id",
		"m.steam_id",
		"m.server_id",
		"m.body",
		"m.team",
		"m.created_on",
		"m.persona_name",
		"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		Where(sq.Eq{"m.person_message_id": personMessageId}).
		ToSql()
	if errQuery != nil {
		return errors.Wrap(errQuery, "Failed to create query")
	}
	return Err(QueryRow(ctx, query, args...).
		Scan(&msg.PersonMessageId,
			&msg.SteamId,
			&msg.ServerId,
			&msg.Body,
			&msg.Team,
			&msg.CreatedOn,
			&msg.PersonaName,
			&msg.ServerName))
}

type ChatHistoryQueryFilter struct {
	QueryFilter
	// TODO Index this string query
	PersonaName string `json:"persona_name,omitempty"`
	SteamId     string `json:"steam_id,omitempty"`
	// TODO Index this body query
	ServerId   int        `json:"server_id,omitempty"`
	SentAfter  *time.Time `json:"sent_after,omitempty"`
	SentBefore *time.Time `json:"sent_before,omitempty"`
}

func QueryChatHistory(ctx context.Context, query ChatHistoryQueryFilter) (PersonMessages, error) {
	qb := sb.Select(
		"m.person_message_id",
		"m.steam_id",
		"m.server_id",
		"m.body",
		"m.team",
		"m.created_on",
		"m.persona_name",
		"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id")
	if query.Offset > 0 {
		qb = qb.Offset(query.Offset)
	}
	if query.Limit > 0 {
		qb = qb.Limit(query.Limit)
	}
	if query.OrderBy != "" {
		if query.SortDesc {
			qb = qb.OrderBy(query.OrderBy + " DESC")
		} else {
			qb = qb.OrderBy(query.OrderBy + " ASC")
		}
	}
	if query.ServerId > 0 {
		qb = qb.Where(sq.Eq{"m.server_id": query.ServerId})
	}
	if query.SteamId != "" {
		qb = qb.Where(sq.Eq{"m.steam_id": query.SteamId})
	}
	if query.PersonaName != "" {
		qb = qb.Where(sq.ILike{"m.persona_name": fmt.Sprintf("%%%s%%", strings.ToLower(query.PersonaName))})
	}
	if query.Query != "" {
		qb = qb.Where(sq.ILike{"m.body": fmt.Sprintf("%%%s%%", strings.ToLower(query.Query))})
	}
	if query.SentBefore != nil {
		qb = qb.Where(sq.Lt{"m.created_on": query.SentBefore})
	}
	if query.SentAfter != nil {
		qb = qb.Where(sq.Gt{"m.created_on": query.SentAfter})
	}
	q, a, qErr := qb.ToSql()
	if qErr != nil {
		return nil, errors.Wrap(qErr, "Failed to build query")
	}
	rows, errQuery := Query(ctx, q, a...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var messages PersonMessages
	for rows.Next() {
		var message PersonMessage
		if errScan := rows.Scan(
			&message.PersonMessageId,
			&message.SteamId,
			&message.ServerId,
			&message.Body,
			&message.Team,
			&message.CreatedOn,
			&message.PersonaName,
			&message.ServerName,
		); errScan != nil {
			return nil, Err(errScan)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (PersonConnections, error) {
	qb := sb.
		Select(
			"DISTINCT on (pn, pc.ip_addr) coalesce(pc.persona_name, pc.steam_id::text) as pn",
			"pc.person_connection_id",
			"pc.steam_id",
			"pc.ip_addr",
			"pc.created_on",
			"coalesce(loc.city_name, '')",
			"coalesce(loc.country_name, '')",
			"coalesce(loc.country_code, '')").
		From("person_connections pc").
		LeftJoin("net_location loc ON pc.ip_addr <@ loc.ip_range").

		// Join("LEFT JOIN net_proxy proxy ON pc.ip_addr <@ proxy.ip_range").
		OrderBy("1").
		Limit(limit)
	qb = qb.Where(sq.Eq{"pc.steam_id": sid64.Int64()})
	query, args, errCreateQuery := qb.ToSql()
	if errCreateQuery != nil {
		return nil, errors.Wrap(errCreateQuery, "Failed to build query")
	}
	rows, errQuery := Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var connections PersonConnections
	for rows.Next() {
		var c PersonConnection
		if errScan := rows.Scan(&c.PersonaName, &c.PersonConnectionId, &c.SteamId, &c.IPAddr, &c.CreatedOn,
			&c.IPInfo.CityName, &c.IPInfo.CountryName, &c.IPInfo.CountryCode,
		); errScan != nil {
			return nil, Err(errScan)
		}
		connections = append(connections, c)
	}
	return connections, nil
}

func AddConnectionHistory(ctx context.Context, conn *PersonConnection) error {
	const q = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on) 
		VALUES ($1, $2, $3, $4) RETURNING person_connection_id`
	if errQuery := QueryRow(ctx, q, conn.SteamId, conn.IPAddr, conn.PersonaName, conn.CreatedOn).
		Scan(&conn.PersonConnectionId); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

var personAuthColumns = []string{"person_auth_id", "steam_id", "ip_addr", "refresh_token", "created_on"}

func GetPersonAuth(ctx context.Context, sid64 steamid.SID64, ipAddr net.IP, auth *PersonAuth) error {
	query, args, errQuery := sb.
		Select(personAuthColumns...).
		From("person_auth").
		Where(sq.And{sq.Eq{"steam_id": sid64}, sq.Eq{"ip_addr": ipAddr.String()}}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	return Err(QueryRow(ctx, query, args...).
		Scan(&auth.PersonAuthId, &auth.SteamId, &auth.IpAddr, &auth.RefreshToken, &auth.CreatedOn))
}

func GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *PersonAuth) error {
	query, args, errQuery := sb.
		Select(personAuthColumns...).
		From("person_auth").
		Where(sq.And{sq.Eq{"refresh_token": token}}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	return Err(QueryRow(ctx, query, args...).
		Scan(&auth.PersonAuthId, &auth.SteamId, &auth.IpAddr, &auth.RefreshToken, &auth.CreatedOn))
}

func SavePersonAuth(ctx context.Context, auth *PersonAuth) error {
	query, args, errQuery := sb.
		Insert("person_auth").
		Columns("steam_id", "ip_addr", "refresh_token", "created_on").
		Values(auth.SteamId, auth.IpAddr.String(), auth.RefreshToken, auth.CreatedOn).
		Suffix("RETURNING \"person_auth_id\"").
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	return Err(QueryRow(ctx, query, args...).Scan(&auth.PersonAuthId))
}

func DeletePersonAuth(ctx context.Context, authId int64) error {
	query, args, errQuery := sb.
		Delete("person_auth").
		Where(sq.Eq{"person_auth_id": authId}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	return Err(Exec(ctx, query, args...))
}

func PrunePersonAuth(ctx context.Context) error {
	query, args, errQuery := sb.
		Delete("person_auth").
		Where(sq.Gt{"created_on + interval '1 month'": config.Now()}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	return Err(Exec(ctx, query, args...))
}

func SendNotification(ctx context.Context, targetId steamid.SID64, severity consts.NotificationSeverity, message string, link string) error {
	query, args, errQuery := sb.
		Insert("person_notification").
		Columns("steam_id", "severity", "message", "link", "created_on").
		Values(targetId, severity, message, link, config.Now()).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	return nil
}

type NotificationQuery struct {
	QueryFilter
	SteamID steamid.SID64 `json:"steam_id,string"`
}

func GetPersonNotifications(ctx context.Context, steamId steamid.SID64) ([]UserNotification, error) {
	var notifications []UserNotification
	query, args, errQuery := sb.
		Select("person_notification_id", "steam_id", "read", "deleted", "severity", "message", "link", "count", "created_on").
		From("person_notification").
		Where(sq.And{sq.Eq{"steam_id": steamId}, sq.Eq{"deleted": false}}).
		OrderBy("person_notification_id desc").
		ToSql()
	if errQuery != nil {
		return notifications, Err(errQuery)
	}
	rows, errRows := Query(ctx, query, args...)
	if errRows != nil {
		return notifications, errRows
	}
	defer rows.Close()
	for rows.Next() {
		var n UserNotification
		if errScan := rows.Scan(&n.NotificationId, &n.SteamId, &n.Read, &n.Deleted,
			&n.Severity, &n.Message, &n.Link, &n.Count, &n.CreatedOn); errScan != nil {
			return notifications, errScan
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func SetNotificationsRead(ctx context.Context, notificationIds []int64) error {
	query, args, errQuery := sb.
		Update("person_notification").
		Set("deleted", true).
		Where(sq.Eq{"person_notification_id": notificationIds}).
		ToSql()
	if errQuery != nil {
		return errQuery
	}
	return Err(Exec(ctx, query, args...))
}

func GetSteamIdsAbove(ctx context.Context, privilege consts.Privilege) (steamid.Collection, error) {
	query, args, errQuery := sb.
		Select("steam_id").
		From("person").
		Where(sq.GtOrEq{"permission_level": privilege}).
		ToSql()
	if errQuery != nil {
		return nil, errQuery
	}
	rows, errRows := Query(ctx, query, args...)
	if errRows != nil {
		return nil, errRows
	}
	defer rows.Close()
	var ids steamid.Collection
	for rows.Next() {
		var sid steamid.SID64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, errScan
		}
		ids = append(ids, sid)
	}
	return ids, nil
}
