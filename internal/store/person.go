package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
)

type UserNotification struct {
	PersonNotificationID int64                       `json:"person_notification_id"`
	SteamID              steamid.SID64               `json:"steam_id"`
	Read                 bool                        `json:"read"`
	Deleted              bool                        `json:"deleted"`
	Severity             consts.NotificationSeverity `json:"severity"`
	Message              string                      `json:"message"`
	Link                 string                      `json:"link"`
	Count                int                         `json:"count"`
	CreatedOn            time.Time                   `json:"created_on"`
}

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID          steamid.SID64         `db:"steam_id" json:"steam_id"`
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

func (p Person) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
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
	curTime := time.Now()

	return Person{
		SteamID:          sid64,
		CreatedOn:        curTime,
		UpdatedOn:        curTime,
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
		UpdatedOnSteam:   curTime,
		PlayerSummary: &steamweb.PlayerSummary{
			SteamID: sid64,
		},
	}
}

type People []Person

func (p People) ToSteamIDCollection() steamid.Collection {
	var collection steamid.Collection

	for _, person := range p {
		collection = append(collection, person.SteamID)
	}

	return collection
}

func (p People) AsMap() map[steamid.SID64]Person {
	m := map[steamid.SID64]Person{}
	for _, person := range p {
		m[person.SteamID] = person
	}

	return m
}

type PersonChat struct {
	PersonChatID int64
	SteamID      steamid.SID64
	ServerID     int
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

	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}

type UserMessage struct {
	ParentID  int64         `json:"parent_id"`
	MessageID int64         `json:"message_id"`
	AuthorID  steamid.SID64 `json:"author_id"`
	Contents  string        `json:"contents"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
}

func NewUserMessage(parentID int64, authorID steamid.SID64, message string) UserMessage {
	return UserMessage{
		ParentID:  parentID,
		AuthorID:  authorID,
		Contents:  message,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
}

type PersonAuth struct {
	PersonAuthID int64         `json:"person_auth_id"`
	SteamID      steamid.SID64 `json:"steam_id"`
	IPAddr       net.IP        `json:"ip_addr"`
	RefreshToken string        `json:"refresh_token"`
	CreatedOn    time.Time     `json:"created_on"`
}

func SecureRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-"

	ret := make([]byte, n)

	for currentChar := 0; currentChar < n; currentChar++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}

		ret[currentChar] = letters[num.Int64()]
	}

	return string(ret)
}

func NewPersonAuth(sid64 steamid.SID64, addr net.IP, fingerPrint string) PersonAuth {
	return PersonAuth{
		PersonAuthID: 0,
		SteamID:      sid64,
		IPAddr:       addr,
		RefreshToken: fingerPrint,
		CreatedOn:    time.Now(),
	}
}

type PersonConnection struct {
	PersonConnectionID int64         `json:"person_connection_id"`
	IPAddr             net.IP        `json:"ip_addr"`
	SteamID            steamid.SID64 `json:"steam_id"`
	PersonaName        string        `json:"persona_name"`
	CreatedOn          time.Time     `json:"created_on"`
}

type PersonConnections []PersonConnection

type PersonMessage struct {
	PersonMessageID int64         `json:"person_message_id"`
	MatchID         uuid.UUID     `json:"match_id"`
	SteamID         steamid.SID64 `json:"steam_id"`
	AvatarHash      string        `json:"avatar_hash"`
	PersonaName     string        `json:"persona_name"`
	ServerName      string        `json:"server_name"`
	ServerID        int           `json:"server_id"`
	Body            string        `json:"body"`
	Team            bool          `json:"team"`
	CreatedOn       time.Time     `json:"created_on"`
	Flagged         bool          `json:"flagged"`
}

type PersonMessages []PersonMessage

func (db *Store) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	query, args, errQueryArgs := db.sb.Delete("person").Where(sq.Eq{"steam_id": steamID.Int64()}).ToSql()
	if errQueryArgs != nil {
		return errors.Wrapf(errQueryArgs, "Failed to create query")
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	return nil
}

// SavePerson will insert or update the person record.
func (db *Store) SavePerson(ctx context.Context, person *Person) error {
	person.UpdatedOn = time.Now()
	// FIXME
	if person.PermissionLevel == 0 {
		person.PermissionLevel = 10
	}

	if !person.IsNew {
		return db.updatePerson(ctx, person)
	}

	person.CreatedOn = person.UpdatedOn

	return db.insertPerson(ctx, person)
}

func (db *Store) updatePerson(ctx context.Context, person *Person) error {
	const query = `
		UPDATE person 
		SET 
		    updated_on = $2, communityvisibilitystate = $3, profilestate = $4, personaname = $5, profileurl = $6, avatar = $7,
    		avatarmedium = $8, avatarfull = $9, avatarhash = $10, personastate = $11, realname = $12, timecreated = $13,
		    loccountrycode = $14, locstatecode = $15, loccityid = $16, permission_level = $17, discord_id = $18,
		    community_banned = $19, vac_bans = $20, game_bans = $21, economy_ban = $22, days_since_last_ban = $23,
			updated_on_steam = $24, muted = $25
		WHERE steam_id = $1`

	person.UpdatedOn = time.Now()

	if errExec := db.
		Exec(ctx, query, person.SteamID.Int64(), person.UpdatedOn,
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

func (db *Store) insertPerson(ctx context.Context, person *Person) error {
	query, args, errQueryArgs := db.sb.
		Insert("person").
		Columns("created_on", "updated_on", "steam_id", "communityvisibilitystate", "profilestate",
			"personaname", "profileurl", "avatar", "avatarmedium", "avatarfull", "avatarhash", "personastate",
			"realname", "timecreated", "loccountrycode", "locstatecode", "loccityid", "permission_level",
			"discord_id", "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted").
		Values(person.CreatedOn, person.UpdatedOn, person.SteamID.Int64(), person.PlayerSummary.CommunityVisibilityState,
			person.PlayerSummary.ProfileState, person.PlayerSummary.PersonaName, person.PlayerSummary.ProfileURL,
			person.PlayerSummary.Avatar, person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull,
			person.PlayerSummary.AvatarHash, person.PlayerSummary.PersonaState, person.PlayerSummary.RealName,
			person.PlayerSummary.TimeCreated, person.PlayerSummary.LocCountryCode, person.PlayerSummary.LocStateCode,
			person.PlayerSummary.LocCityID, person.PermissionLevel, person.DiscordID, person.CommunityBanned,
			person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam,
			person.Muted).
		ToSql()
	if errQueryArgs != nil {
		return errors.Wrapf(errQueryArgs, "Failed to create query")
	}

	errExec := db.Exec(ctx, query, args...)
	if errExec != nil {
		return Err(errExec)
	}

	person.IsNew = false

	return nil
}

// "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban".
var profileColumns = []string{ //nolint:gochecknoglobals
	"steam_id", "created_on", "updated_on",
	"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
	"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
	"loccountrycode", "locstatecode", "loccityid", "permission_level", "discord_id",
	"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban", "updated_on_steam",
	"muted",
}

// GetPersonBySteamID returns a person by their steam_id. ErrNoResult is returned if the steam_id
// is not known.
func (db *Store) GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *Person) error {
	const query = `
    	SELECT p.created_on,
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
	person.SteamID = sid64

	errQuery := db.
		QueryRow(ctx, query, sid64.Int64()).
		Scan(&person.CreatedOn,
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

func (db *Store) GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (People, error) {
	var ids []int64 //nolint:prealloc
	for _, sid := range fp.Uniq[steamid.SID64](steamIds) {
		ids = append(ids, sid.Int64())
	}

	queryBuilder := db.sb.
		Select(profileColumns...).
		From("person").
		Where(sq.Eq{"steam_id": ids})

	query, args, errQueryArgs := queryBuilder.ToSql()
	if errQueryArgs != nil {
		return nil, errors.Wrapf(errQueryArgs, "Failed to create query")
	}

	var people People

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			steamID int64
			person  = NewPerson("")
		)

		if errScan := rows.Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel, &person.DiscordID,
			&person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
			&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errors.Wrapf(errScan, "Failedto scan person")
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	return people, nil
}

type PlayerQuery struct {
	QueryFilter
	SteamID     StringSID `json:"steam_id"`
	Personaname string    `json:"personaname"`
}

func (db *Store) GetPeople(ctx context.Context, queryFilter PlayerQuery) (People, int64, error) {
	queryBuilder := db.sb.
		Select("p.steam_id", "p.created_on", "p.updated_on",
			"p.communityvisibilitystate", "p.profilestate", "p.personaname", "p.profileurl", "p.avatar",
			"p.avatarmedium", "p.avatarfull", "p.avatarhash", "p.personastate", "p.realname", "p.timecreated",
			"p.loccountrycode", "p.locstatecode", "p.loccityid", "p.permission_level", "p.discord_id",
			"p.community_banned", "p.vac_bans", "p.game_bans", "p.economy_ban", "p.days_since_last_ban",
			"p.updated_on_steam", "p.muted").
		From("person p")

	conditions := sq.And{}

	if queryFilter.SteamID != "" {
		steamID, errSteamID := queryFilter.SteamID.SID64(ctx)
		if errSteamID != nil {
			return nil, 0, errors.Wrap(errSteamID, "Invalid Steam ID")
		}

		conditions = append(conditions, sq.Eq{"p.steam_id": steamID.Int64()})
	}

	if queryFilter.Personaname != "" {
		// TODO add lower-cased functional index to avoid table scan
		conditions = append(conditions, sq.ILike{"p.personaname": strings.ToLower(queryFilter.Personaname)})
	}

	limit := uint64(25)

	if queryFilter.Limit == 0 && queryFilter.Limit <= 100 {
		limit = queryFilter.Limit
	}

	queryBuilder = queryBuilder.Limit(limit)

	if queryFilter.Offset > 0 {
		queryBuilder = queryBuilder.Offset(queryFilter.Offset * limit)
	}

	direction := "DESC"
	if !queryFilter.Desc {
		direction = "ASC"
	}

	var orderBy string
	if queryFilter.OrderBy != "" {
		orderBy = fmt.Sprintf("p.%s", queryFilter.OrderBy)
	} else {
		orderBy = "p.updated_on_steam"
	}

	queryBuilder = queryBuilder.
		OrderBy(fmt.Sprintf("%s %s", orderBy, direction)).
		Where(conditions)

	query, args, errQueryArgs := queryBuilder.ToSql()
	if errQueryArgs != nil {
		return nil, 0, errors.Wrap(errQueryArgs, "Failed to create query")
	}

	var people People

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, 0, errQuery
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person  = NewPerson("")
			steamID int64
		)

		if errScan := rows.
			Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
				&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar,
				&person.AvatarMedium, &person.AvatarFull, &person.AvatarHash, &person.PersonaState,
				&person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
				&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
				&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
				&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, 0, errors.Wrapf(errScan, "Failed to scan person")
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	count, errCount := db.GetCount(ctx, db.sb.
		Select("COUNT(p.steam_id)").
		From("person p").
		Where(conditions))
	if errCount != nil {
		return nil, 0, errors.Wrap(errCount, "Failed to exec count query")
	}

	return people, count, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func (db *Store) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *Person) error {
	errGetPerson := db.GetPersonBySteamID(ctx, sid64, person)
	if errGetPerson != nil && errors.Is(Err(errGetPerson), ErrNoResult) {
		// FIXME
		newPerson := NewPerson(sid64)
		*person = newPerson

		return db.SavePerson(ctx, person)
	}

	return errGetPerson
}

// GetPersonByDiscordID returns a person by their discord_id.
func (db *Store) GetPersonByDiscordID(ctx context.Context, discordID string, person *Person) error {
	query, args, errQueryArgs := db.sb.Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": discordID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	var steamID int64

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}

	errQuery := db.QueryRow(ctx, query, args...).Scan(&steamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans,
		&person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted)
	if errQuery != nil {
		return Err(errQuery)
	}

	person.SteamID = steamid.New(steamID)

	return nil
}

func (db *Store) GetExpiredProfiles(ctx context.Context, limit uint64) ([]Person, error) {
	query, args, errArgs := db.sb.
		Select(profileColumns...).
		From("person").
		OrderBy("updated_on_steam").
		Limit(limit).
		ToSql()
	if errArgs != nil {
		return nil, Err(errArgs)
	}

	var people []Person

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person  = NewPerson("")
			steamID int64
		)

		if errScan := rows.Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel,
			&person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans,
			&person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, Err(errScan)
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	return people, nil
}

func (db *Store) AddChatHistory(ctx context.Context, message *PersonMessage) error {
	const query = `INSERT INTO person_messages 
    		(steam_id, server_id, body, team, created_on, persona_name, match_id) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING person_message_id`

	if errScan := db.
		QueryRow(ctx, query, message.SteamID.Int64(), message.ServerID, message.Body, message.Team,
			message.CreatedOn, message.PersonaName, message.MatchID).
		Scan(&message.PersonMessageID); errScan != nil {
		return Err(errScan)
	}

	return nil
}

func (db *Store) GetPersonMessageByID(ctx context.Context, personMessageID int64, msg *PersonMessage) error {
	query, args, errQuery := db.sb.Select(
		"m.person_message_id",
		"m.steam_id",
		"m.server_id",
		"m.body",
		"m.team",
		"m.created_on",
		"m.persona_name",
		"m.match_id",
		"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		Where(sq.Eq{"m.person_message_id": personMessageID}).
		ToSql()
	if errQuery != nil {
		return errors.Wrap(errQuery, "Failed to create query")
	}

	var steamID int64

	errRow := db.
		QueryRow(ctx, query, args...).
		Scan(&msg.PersonMessageID,
			&steamID,
			&msg.ServerID,
			&msg.Body,
			&msg.Team,
			&msg.CreatedOn,
			&msg.PersonaName,
			&msg.MatchID,
			&msg.ServerName)
	if errRow != nil {
		return Err(errRow)
	}

	msg.SteamID = steamid.New(steamID)

	return nil
}

type ConnectionHistoryQueryFilter struct {
	QueryFilter
	IP       string    `json:"ip"`
	SourceID StringSID `json:"source_id"`
}

type QueryConnectionHistoryResult struct {
	PersonConnection
	Count int64 `json:"count"`
}

// TODO add server_id to connection hist
func (db *Store) QueryConnectionHistory(ctx context.Context, opts ConnectionHistoryQueryFilter) ([]QueryConnectionHistoryResult, int64, error) {
	builder := db.sb.
		Select("c.person_connection_id", "c.steam_id",
			"c.ip_addr", "c.persona_name", "c.created_on").
		From("person_connections c").
		GroupBy("c.person_connection_id, c.ip_addr")

	var constraints sq.And

	if opts.SourceID != "" {
		sid, errSID := opts.SourceID.SID64(ctx)
		if errSID != nil {
			return nil, 0, errors.Wrap(steamid.ErrInvalidSID, "Invalid steam id in query")
		}

		constraints = append(constraints, sq.Eq{"c.steam_id": sid.Int64()})
	}

	if opts.Offset > 0 {
		builder = builder.Offset(opts.Offset)
	}

	if opts.Limit > 0 && opts.Limit <= 100 {
		builder = builder.Limit(opts.Limit)
	} else {
		builder = builder.Limit(25)
	}

	orderBy := "filter_id"
	if opts.OrderBy != "" {
		orderBy = opts.OrderBy
	}

	order := "ASC"
	if opts.Desc {
		order = "DESC"
	}

	builder = builder.OrderBy(fmt.Sprintf("c.%s %s", orderBy, order))

	var messages []QueryConnectionHistoryResult

	rowsQuery, rowsArgs, rowsQueryErr := builder.Where(constraints).ToSql()
	if rowsQueryErr != nil {
		return nil, 0, errors.Wrap(rowsQueryErr, "Failed to build rows query")
	}

	rows, errQuery := db.Query(ctx, rowsQuery, rowsArgs...)
	if errQuery != nil {
		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			connHistory QueryConnectionHistoryResult
			steamID     int64
			target      = []any{
				&connHistory.PersonConnectionID,
				&steamID,
				&connHistory.IPAddr,
				&connHistory.PersonaName,
				&connHistory.CreatedOn,
				&connHistory.Count,
			}
		)

		if errScan := rows.Scan(target...); errScan != nil {
			return nil, 0, Err(errScan)
		}

		connHistory.SteamID = steamid.New(steamID)

		messages = append(messages, connHistory)
	}

	if messages == nil {
		return []QueryConnectionHistoryResult{}, 0, nil
	}

	count, errCount := db.GetCount(ctx, db.sb.
		Select("count(c.person_connection_id)").
		From("person_connections c").
		Where(constraints))

	if errCount != nil {
		return nil, 0, errCount
	}

	return messages, count, nil
}

var errLimit = errors.New("Requested too many")

type ChatHistoryQueryFilter struct {
	QueryFilter
	Personaname   string     `json:"personaname,omitempty"`
	SteamID       string     `json:"steam_id,omitempty"`
	ServerID      int        `json:"server_id,omitempty"`
	DateStart     *time.Time `json:"date_start,omitempty"`
	DateEnd       *time.Time `json:"date_end,omitempty"`
	Unrestricted  bool       `json:"-"`
	DontCalcTotal bool       `json:"-"`
	FlaggedOnly   bool       `json:"flagged_only"`
}

type QueryChatHistoryResult struct {
	PersonMessage
	AutoFilterFlagged bool `json:"auto_filter_flagged"`
}

const minQueryLen = 2

func (db *Store) QueryChatHistory(ctx context.Context, query ChatHistoryQueryFilter) ([]QueryChatHistoryResult, int64, error) { //nolint:maintidx
	if query.Limit > 1000 && !query.Unrestricted {
		return nil, 0, errLimit
	}

	columns := []string{
		"m.person_message_id",
		"m.steam_id ",
		"m.server_id",
		"m.body",
		"m.team ",
		"m.created_on",
		"m.persona_name",
		"m.match_id",
		"s.short_name",
		"COUNT(f.person_message_id)::int::boolean as flagged",
		"p.avatarhash",
	}

	countCols := []string{"count(m.created_on) as count"}

	if query.Query != "" && len(query.Query) < minQueryLen {
		return nil, 0, errors.New("Query value too short")
	}

	if query.Personaname != "" && len(query.Personaname) < minQueryLen {
		return nil, 0, errors.New("Name value too short")
	}

	count := db.sb.
		Select(countCols...).
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id")

	builder := db.sb.
		Select(columns...).
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		LeftJoin("person_messages_filter f on m.person_message_id = f.person_message_id").
		LeftJoin("person p on p.steam_id = m.steam_id")

	if query.Offset > 0 {
		builder = builder.Offset(query.Offset)
	}

	if query.Limit > 0 {
		builder = builder.Limit(query.Limit)
	} else {
		builder = builder.Limit(50)
	}

	if query.OrderBy != "created_on" && query.OrderBy != "person_message_id" {
		return nil, 0, errors.New("Sort only allowed on created_on")
	}

	prefix := "m."

	query.OrderBy = prefix + query.OrderBy

	if query.OrderBy != "" {
		orderBy := []string{query.OrderBy}

		if query.Desc {
			builder = builder.OrderBy(strings.Join(orderBy, ",") + " DESC")
		} else {
			builder = builder.OrderBy(strings.Join(orderBy, ",") + " ASC")
		}

		groupBy := []string{query.OrderBy, "m.created_on", "m.person_message_id", "s.short_name", "f.person_message_id", "p.avatarhash"}

		if query.Query != "" {
			groupBy = append(groupBy, "m.message_search")
		}

		if query.Personaname != "" {
			groupBy = append(groupBy, "m.name_search")
		}

		builder = builder.GroupBy(groupBy...)
	}

	var ands sq.And

	now := time.Now()

	if !query.Unrestricted {
		unrTime := now.AddDate(0, 0, -14)
		if query.DateStart != nil && query.DateStart.Before(unrTime) {
			return nil, 0, consts.ErrInvalidDuration
		}
	}

	switch {
	case query.DateStart != nil && query.DateEnd != nil:
		ands = append(ands, sq.Expr("m.created_on BETWEEN ? AND ?", query.DateStart, query.DateEnd))
	case query.DateStart != nil:
		ands = append(ands, sq.Expr("? > m.created_on", query.DateStart))
	case query.DateEnd != nil:
		ands = append(ands, sq.Expr("? < m.created_on", query.DateEnd))
	}

	if query.ServerID > 0 {
		ands = append(ands, sq.Eq{"m.server_id": query.ServerID})
	}

	if query.SteamID != "" {
		sid := steamid.New(query.SteamID)
		if !sid.Valid() {
			return nil, 0, errors.Wrap(steamid.ErrInvalidSID, "Invalid steam id in query")
		}

		ands = append(ands, sq.Eq{"m.steam_id": sid.Int64()})
	}

	if query.Personaname != "" {
		ands = append(ands, sq.Expr(`name_search @@ websearch_to_tsquery('simple', ?)`, query.Personaname))
	}

	if query.Query != "" {
		ands = append(ands, sq.Expr(`message_search @@ websearch_to_tsquery('simple', ?)`, query.Query))
	}

	if query.FlaggedOnly {
		ands = append(ands, sq.Eq{"flagged": true})
	}

	count = count.Where(ands)
	builder = builder.Where(ands)

	var totalRows int64

	if !query.DontCalcTotal {
		countQuery, countQueryArgs, countQueryErr := count.ToSql()
		if countQueryErr != nil {
			return nil, 0, errors.Wrap(countQueryErr, "Failed to build count query")
		}

		if errCount := db.QueryRow(ctx, countQuery, countQueryArgs...).Scan(&totalRows); errCount != nil {
			return nil, 0, errors.Wrap(errCount, "Failed to perform count query")
		}
	}

	var messages []QueryChatHistoryResult

	if totalRows == 0 && !query.DontCalcTotal {
		return messages, 0, nil
	}

	rowsQuery, rowsArgs, rowsQueryErr := builder.ToSql()
	if rowsQueryErr != nil {
		return nil, 0, errors.Wrap(rowsQueryErr, "Failed to build rows query")
	}

	rows, errQuery := db.Query(ctx, rowsQuery, rowsArgs...)
	if errQuery != nil {
		return nil, totalRows, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			message QueryChatHistoryResult
			steamID int64
			matchID []byte
			target  = []any{
				&message.PersonMessageID,
				&steamID,
				&message.ServerID,
				&message.Body,
				&message.Team,
				&message.CreatedOn,
				&message.PersonaName,
				&matchID,
				&message.ServerName,
				&message.AutoFilterFlagged,
				&message.AvatarHash,
			}
		)

		if errScan := rows.Scan(target...); errScan != nil {
			return nil, 0, Err(errScan)
		}

		if matchID != nil {
			message.MatchID = uuid.FromBytesOrNil(matchID)
		}

		message.SteamID = steamid.New(steamID)

		messages = append(messages, message)
	}

	if messages == nil {
		// Return empty list instead of null
		messages = []QueryChatHistoryResult{}
	}

	return messages, totalRows, nil
}

func (db *Store) GetPersonMessage(ctx context.Context, messageID int64, msg *QueryChatHistoryResult) error {
	const query = `
			SELECT m.person_message_id, m.steam_id,	m.server_id, m.body, m.team, m.created_on, 
			       m.persona_name, m.match_id, s.short_name, COUNT(f.person_message_id)::int::boolean as flagged
			FROM person_messages m 
			LEFT JOIN server s on m.server_id = s.server_id
			LEFT JOIN person_messages_filter f on m.person_message_id = f.person_message_id
		 	WHERE m.person_message_id = $1
			GROUP BY m.person_message_id, s.short_name
		`

	if errScan := db.conn.
		QueryRow(ctx, query, messageID).
		Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
			&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged); errScan != nil {
		return errors.Wrap(errScan, "Failed to scan message result")
	}

	return nil
}

func (db *Store) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error) {
	const query = `
		(
			SELECT m.person_message_id, m.steam_id,	m.server_id, m.body, m.team, m.created_on, 
			       m.persona_name,  m.match_id, s.short_name, COUNT(f.person_message_id)::int::boolean as flagged
			FROM person_messages m 
			LEFT JOIN server s on m.server_id = s.server_id
			LEFT JOIN person_messages_filter f on m.person_message_id = f.person_message_id
		 	WHERE m.server_id = $3 AND m.person_message_id >= $1 
		 	GROUP BY m.person_message_id, s.short_name 
		 	ORDER BY m.person_message_id ASC
		 	
		 	LIMIT $2+1
		)
		UNION
		(
			SELECT m.person_message_id, m.steam_id, m.server_id, m.body, m.team, m.created_on, 
			       m.persona_name,  m.match_id, s.short_name, COUNT(f.person_message_id)::int::boolean as flagged
		 	FROM person_messages m 
		 	    LEFT JOIN server s on m.server_id = s.server_id 
		 	LEFT JOIN person_messages_filter f on m.person_message_id = f.person_message_id
		 	WHERE m.server_id = $3 AND  m.person_message_id < $1
		 	GROUP BY m.person_message_id, s.short_name
		 	ORDER BY m.person_message_id DESC
		 	LIMIT $2
		)
		ORDER BY person_message_id DESC`

	if paddedMessageCount > 1000 {
		paddedMessageCount = 1000
	}

	if paddedMessageCount <= 0 {
		paddedMessageCount = 5
	}

	rows, errRows := db.conn.Query(ctx, query, messageID, paddedMessageCount, serverID)
	if errRows != nil {
		return nil, errors.Wrap(errRows, "Failed to fetch message context")
	}
	defer rows.Close()

	var messages []QueryChatHistoryResult

	for rows.Next() {
		var msg QueryChatHistoryResult

		if errScan := rows.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
			&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged); errScan != nil {
			return nil, errors.Wrap(errRows, "Failed to scan message result")
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (db *Store) GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (PersonConnections, error) {
	builder := db.sb.
		Select(
			"DISTINCT on (pn, pc.ip_addr) coalesce(pc.persona_name, pc.steam_id::text) as pn",
			"pc.person_connection_id",
			"pc.steam_id",
			"pc.ip_addr",
			"pc.created_on").
		From("person_connections pc").
		LeftJoin("net_location loc ON pc.ip_addr <@ loc.ip_range").
		// Join("LEFT JOIN net_proxy proxy ON pc.ip_addr <@ proxy.ip_range").
		OrderBy("1").
		Limit(limit)
	builder = builder.Where(sq.Eq{"pc.steam_id": sid64.Int64()})

	query, args, errCreateQuery := builder.ToSql()
	if errCreateQuery != nil {
		return nil, errors.Wrap(errCreateQuery, "Failed to build query")
	}

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	var connections PersonConnections

	for rows.Next() {
		var (
			conn    PersonConnection
			steamID int64
		)

		if errScan := rows.Scan(&conn.PersonaName, &conn.PersonConnectionID, &steamID, &conn.IPAddr, &conn.CreatedOn); errScan != nil {
			return nil, Err(errScan)
		}

		conn.SteamID = steamid.New(steamID)

		connections = append(connections, conn)
	}

	return connections, nil
}

func (db *Store) AddConnectionHistory(ctx context.Context, conn *PersonConnection) error {
	const query = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on) 
		VALUES ($1, $2, $3, $4) RETURNING person_connection_id`

	if errQuery := db.
		QueryRow(ctx, query, conn.SteamID.Int64(), conn.IPAddr, conn.PersonaName, conn.CreatedOn).
		Scan(&conn.PersonConnectionID); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

var personAuthColumns = []string{"person_auth_id", "steam_id", "ip_addr", "refresh_token", "created_on"} //nolint:gochecknoglobals

func (db *Store) GetPersonAuth(ctx context.Context, sid64 steamid.SID64, ipAddr net.IP, auth *PersonAuth) error {
	query, args, errQuery := db.sb.
		Select(personAuthColumns...).
		From("person_auth").
		Where(sq.And{sq.Eq{"steam_id": sid64.Int64()}, sq.Eq{"ip_addr": ipAddr.String()}}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	var steamID int64
	errRow := db.QueryRow(ctx, query, args...).
		Scan(&auth.PersonAuthID, &steamID, &auth.IPAddr, &auth.RefreshToken, &auth.CreatedOn)

	if errRow != nil {
		return Err(errRow)
	}

	auth.SteamID = steamid.New(steamID)

	return nil
}

func (db *Store) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *PersonAuth) error {
	query, args, errQuery := db.sb.
		Select(personAuthColumns...).
		From("person_auth").
		Where(sq.And{sq.Eq{"refresh_token": token}}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	var steamID int64

	errRow := db.
		QueryRow(ctx, query, args...).
		Scan(&auth.PersonAuthID, &steamID, &auth.IPAddr, &auth.RefreshToken, &auth.CreatedOn)
	if errRow != nil {
		return Err(errRow)
	}

	auth.SteamID = steamid.New(steamID)

	return nil
}

func (db *Store) SavePersonAuth(ctx context.Context, auth *PersonAuth) error {
	query, args, errQuery := db.sb.
		Insert("person_auth").
		Columns("steam_id", "ip_addr", "refresh_token", "created_on").
		Values(auth.SteamID.Int64(), auth.IPAddr.String(), auth.RefreshToken, auth.CreatedOn).
		Suffix("RETURNING \"person_auth_id\"").
		ToSql()

	if errQuery != nil {
		return Err(errQuery)
	}

	return Err(db.QueryRow(ctx, query, args...).Scan(&auth.PersonAuthID))
}

func (db *Store) DeletePersonAuth(ctx context.Context, authID int64) error {
	query, args, errQuery := db.sb.
		Delete("person_auth").
		Where(sq.Eq{"person_auth_id": authID}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	return Err(db.Exec(ctx, query, args...))
}

func (db *Store) PrunePersonAuth(ctx context.Context) error {
	query, args, errQuery := db.sb.
		Delete("person_auth").
		Where(sq.Gt{"created_on + interval '1 month'": time.Now()}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	return Err(db.Exec(ctx, query, args...))
}

func (db *Store) SendNotification(ctx context.Context, targetID steamid.SID64, severity consts.NotificationSeverity, message string, link string) error {
	query, args, errQuery := db.sb.
		Insert("person_notification").
		Columns("steam_id", "severity", "message", "link", "created_on").
		Values(targetID.Int64(), severity, message, link, time.Now()).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	return nil
}

type NotificationQuery struct {
	QueryFilter
	SteamID steamid.SID64 `json:"steam_id"`
}

func (db *Store) GetPersonNotifications(ctx context.Context, steamID steamid.SID64) ([]UserNotification, error) {
	var notifications []UserNotification

	query, args, errQuery := db.sb.
		Select("person_notification_id", "steam_id", "read", "deleted", "severity", "message", "link", "count", "created_on").
		From("person_notification").
		Where(sq.And{sq.Eq{"steam_id": steamID}, sq.Eq{"deleted": false}}).
		OrderBy("person_notification_id desc").
		ToSql()
	if errQuery != nil {
		return []UserNotification{}, Err(errQuery)
	}

	rows, errRows := db.Query(ctx, query, args...)
	if errRows != nil {
		return []UserNotification{}, errRows
	}

	defer rows.Close()

	for rows.Next() {
		var (
			notif      UserNotification
			outSteamID int64
		)

		if errScan := rows.Scan(&notif.PersonNotificationID, &outSteamID, &notif.Read, &notif.Deleted,
			&notif.Severity, &notif.Message, &notif.Link, &notif.Count, &notif.CreatedOn); errScan != nil {
			return []UserNotification{}, errors.Wrapf(errScan, "Failed to scan notification")
		}

		notif.SteamID = steamid.New(outSteamID)

		notifications = append(notifications, notif)
	}

	if notifications == nil {
		return []UserNotification{}, nil
	}

	return notifications, nil
}

func (db *Store) SetNotificationsRead(ctx context.Context, notificationIds []int64) error {
	query, args, errQuery := db.sb.
		Update("person_notification").
		Set("deleted", true).
		Where(sq.Eq{"person_notification_id": notificationIds}).
		ToSql()
	if errQuery != nil {
		return errors.Wrapf(errQuery, "Failed to create query")
	}

	return Err(db.Exec(ctx, query, args...))
}

func (db *Store) GetSteamIdsAbove(ctx context.Context, privilege consts.Privilege) (steamid.Collection, error) {
	query, args, errQuery := db.sb.
		Select("steam_id").
		From("person").
		Where(sq.GtOrEq{"permission_level": privilege}).
		ToSql()
	if errQuery != nil {
		return nil, errors.Wrapf(errQuery, "Failed to create query")
	}

	rows, errRows := db.Query(ctx, query, args...)
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var ids steamid.Collection

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, errors.Wrapf(errScan, "Failed to scan steam id")
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}
