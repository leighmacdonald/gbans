package person

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/playerqueue"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/sync/errgroup"
)

var ErrPlayerDoesNotExist = errors.New("player does not exist")

type SteamMember interface {
	IsMember(steamID steamid.SteamID) (int64, bool)
}

type Query struct {
	query.Filter
	Personaname          string               `json:"personaname"`
	IP                   string               `json:"ip"`
	StaffOnly            bool                 `json:"staff_only"`
	WithPermissions      permission.Privilege `json:"with_permissions"`
	DiscordID            string               `json:"discord_id"`
	SteamUpdateOlderThan time.Time            `json:"steam_update_older_than"`
	Addr                 net.IP               `json:"addr"`
	SteamIDs             steamid.Collection   `json:"steam_ids"`
}

type RequestPermissionLevelUpdate struct {
	PermissionLevel permission.Privilege `json:"permission_level"`
}

// EconBanState  holds the users current economy ban status.
type EconBanState string

// EconBanState values
//
//goland:noinspection ALL
const (
	EconBanNone      EconBanState = "none"
	EconBanProbation EconBanState = "probation"
	EconBanBanned    EconBanState = "banned"
)

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID               steamid.SteamID        `json:"steam_id"`
	CreatedOn             time.Time              `json:"created_on"`
	UpdatedOn             time.Time              `json:"updated_on"`
	PermissionLevel       permission.Privilege   `json:"permission_level"`
	Muted                 bool                   `json:"muted"`
	isNew                 bool                   `json:"-"`
	DiscordID             string                 `json:"discord_id"`
	PatreonID             string                 `json:"patreon_id"`
	IPAddr                netip.Addr             `json:"-"` // TODO Allow json for admins endpoints
	CommunityBanned       bool                   `json:"community_banned"`
	VACBans               int                    `json:"vac_bans"`
	GameBans              int                    `json:"game_bans"`
	EconomyBan            EconBanState           `json:"economy_ban"`
	DaysSinceLastBan      int                    `json:"days_since_last_ban"`
	UpdatedOnSteam        time.Time              `json:"updated_on_steam"`
	PlayerqueueChatStatus playerqueue.ChatStatus `json:"playerqueue_chat_status"`
	PlayerqueueChatReason string                 `json:"playerqueue_chat_reason"`
	AvatarHash            string                 `json:"avatar_hash"`
	CommentPermission     int64                  `json:"comment_permission"`
	LastLogoff            int64                  `json:"last_logoff"`
	LocCityID             int64                  `json:"loc_city_id"`
	LocCountryCode        string                 `json:"loc_country_code"`
	LocStateCode          string                 `json:"loc_state_code"`
	PersonaName           string                 `json:"persona_name"`
	PersonaState          int64                  `json:"persona_state"`
	PersonaStateFlags     int64                  `json:"persona_state_flags"`
	PrimaryClanID         string                 `json:"primary_clan_id"`
	ProfileState          int64                  `json:"profile_state"`
	ProfileURL            string                 `json:"profile_url"`
	RealName              string                 `json:"real_name"`
	TimeCreated           int64                  `json:"time_created"`
	VisibilityState       int64                  `json:"visibility_state"`
}

func (p Person) Avatar() string {
	return p.GetAvatar().Small()
}

func (p Person) AvatarMedium() string {
	return p.GetAvatar().Medium()
}

func (p Person) AvatarFull() string {
	return p.GetAvatar().Full()
}

func (p Person) Profile() string {
	return "https://steamcommunity.com/profiles/" + p.SteamID.String()
}

func (p Person) Expired() bool {
	return p.isNew || time.Since(p.UpdatedOnSteam) > time.Hour*24*30
}

func (p Person) GetDiscordID() string {
	return p.DiscordID
}

func (p Person) GetName() string {
	if p.PersonaName == "" {
		return p.SteamID.String()
	}

	return p.PersonaName
}

func (p Person) Permissions() permission.Privilege {
	return p.PermissionLevel
}

func (p Person) HasPermission(privilege permission.Privilege) bool {
	return p.PermissionLevel >= privilege
}

func (p Person) GetAvatar() domain.Avatar {
	return domain.NewAvatar(p.AvatarHash)
}

func (p Person) GetSteamID() steamid.SteamID {
	return p.SteamID
}

func (p Person) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

// LoggedIn checks for a valid steamID.
func (p Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

type ProfileResponse struct {
	Player   *Person                  `json:"player"`
	Friends  []thirdparty.SteamFriend `json:"friends"`
	Settings Settings                 `json:"settings"`
}

// New allocates a new default person instance.
func New(sid64 steamid.SteamID) Person {
	curTime := time.Now()

	return Person{
		SteamID:               sid64,
		CreatedOn:             curTime,
		UpdatedOn:             curTime,
		PermissionLevel:       permission.User,
		Muted:                 false,
		isNew:                 true,
		DiscordID:             "",
		CommunityBanned:       false,
		VACBans:               0,
		GameBans:              0,
		EconomyBan:            "none",
		DaysSinceLastBan:      0,
		UpdatedOnSteam:        time.Unix(0, 0),
		PlayerqueueChatStatus: "readwrite",
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

func (p People) AsMap() map[steamid.SteamID]Person {
	m := map[steamid.SteamID]Person{}
	for _, person := range p {
		m[person.SteamID] = person
	}

	return m
}

type Settings struct {
	PersonSettingsID     int64           `json:"person_settings_id"`
	SteamID              steamid.SteamID `json:"steam_id"`
	ForumSignature       string          `json:"forum_signature"`
	ForumProfileMessages bool            `json:"forum_profile_messages"`
	StatsHidden          bool            `json:"stats_hidden"`

	// This key will be absent to indicate that this feature
	// is disabled (and UI should not be shown to the user).
	CenterProjectiles *bool `json:"center_projectiles,omitempty"`

	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type SettingsUpdate struct {
	ForumSignature       string `json:"forum_signature"`
	ForumProfileMessages bool   `json:"forum_profile_messages"`
	StatsHidden          bool   `json:"stats_hidden"`
	CenterProjectiles    *bool  `json:"center_projectiles,omitempty"`
}

type Persons struct {
	owner steamid.SteamID
	repo  Repository
	tfAPI *thirdparty.TFAPI
}

func NewPersons(repository Repository, owner steamid.SteamID, tfAPI *thirdparty.TFAPI) *Persons {
	return &Persons{
		repo:  repository,
		owner: owner,
		tfAPI: tfAPI,
	}
}

func (u *Persons) CanAlter(ctx context.Context, sourceID steamid.SteamID, targetID steamid.SteamID) (bool, error) {
	source, errSource := u.GetOrCreatePersonBySteamID(ctx, nil, sourceID)
	if errSource != nil {
		return false, errSource
	}

	target, errGetProfile := u.GetOrCreatePersonBySteamID(ctx, nil, targetID)
	if errGetProfile != nil {
		return false, errGetProfile
	}

	return source.PermissionLevel > target.PermissionLevel, nil
}

func (u *Persons) QueryProfile(ctx context.Context, query string) (ProfileResponse, error) {
	var resp ProfileResponse

	sid, errResolveSID64 := steamid.Resolve(ctx, query)
	if errResolveSID64 != nil {
		return resp, domain.ErrInvalidSID
	}

	_, _ = u.GetOrCreatePersonBySteamID(ctx, nil, sid)

	person, errGetProfile := u.BySteamID(ctx, nil, sid)
	if errGetProfile != nil {
		return resp, errGetProfile
	}

	if person.Expired() {
		if err := UpdatePlayerSummary(ctx, &person, u.tfAPI); err != nil {
			slog.Error("Failed to update player summary", log.ErrAttr(err))
		} else {
			if errSave := u.Save(ctx, nil, &person); errSave != nil {
				slog.Error("Failed to save person summary", log.ErrAttr(errSave))
			}
		}
	}

	friendList, errFetchFriends := u.tfAPI.Friends(ctx, person.SteamID)
	if errFetchFriends == nil {
		resp.Friends = friendList
	}

	resp.Player = &person

	settings, err := u.GetPersonSettings(ctx, sid)
	if err != nil {
		return resp, err
	}

	resp.Settings = settings

	return resp, nil
}

func (u *Persons) UpdateProfiles(ctx context.Context, transaction pgx.Tx, people People) (int, error) {
	if len(people) > 100 {
		return 0, domain.ErrSteamAPIArgLimit
	}

	var (
		banStates           []thirdparty.SteamBan
		summaries           []thirdparty.PlayerSummaryResponse
		steamIDs            = people.ToSteamIDCollection()
		errGroup, cancelCtx = errgroup.WithContext(ctx)
	)

	errGroup.Go(func() error {
		newBanStates, errBans := FetchPlayerBans(cancelCtx, u.tfAPI, steamIDs)
		if errBans != nil {
			return errors.Join(errBans, domain.ErrFetchSteamBans)
		}

		banStates = newBanStates

		return nil
	})

	errGroup.Go(func() error {
		newSummaries, errSummaries := u.tfAPI.Summaries(ctx, steamIDs)
		if errSummaries != nil {
			return errors.Join(errSummaries, domain.ErrSteamAPISummaries)
		}

		summaries = newSummaries

		return nil
	})

	if errFetch := errGroup.Wait(); errFetch != nil {
		return 0, errors.Join(errFetch, domain.ErrSteamAPI)
	}

	for _, curPerson := range people {
		person := curPerson
		person.isNew = false
		person.UpdatedOnSteam = time.Now()

		for _, newSummary := range summaries {
			summary := newSummary
			if person.SteamID.String() != summary.SteamId {
				continue
			}

			person.AvatarHash = summary.AvatarHash
			person.CommentPermission = summary.CommentPermission
			person.LastLogoff = summary.LastLogoff
			person.LocCityID = summary.LocCityId
			person.LocCountryCode = summary.LocCountryCode
			person.LocStateCode = summary.LocStateCode
			person.PersonaName = summary.PersonaName
			person.PersonaState = summary.PersonaState
			person.PersonaStateFlags = summary.PersonaStateFlags
			person.PrimaryClanID = summary.PrimaryClanId
			person.ProfileState = summary.ProfileState
			person.ProfileURL = summary.ProfileUrl
			person.RealName = summary.RealName
			person.TimeCreated = summary.TimeCreated
			person.VisibilityState = summary.VisibilityState

			break
		}

		for _, banState := range banStates {
			if person.SteamID.String() != banState.SteamId {
				continue
			}

			person.CommunityBanned = banState.CommunityBanned
			person.VACBans = int(banState.NumberOfVacBans)
			person.GameBans = int(banState.NumberOfGameBans)
			person.EconomyBan = EconBanState(banState.EconomyBan)
			person.CommunityBanned = banState.CommunityBanned
			person.DaysSinceLastBan = int(banState.DaysSinceLastBan)
		}

		if errSavePerson := u.repo.Save(ctx, transaction, &person); errSavePerson != nil {
			return 0, errors.Join(errSavePerson, domain.ErrUpdatePerson)
		}
	}

	return len(people), nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
func (u *Persons) SetSteam(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID, discordID string) error {
	if !sid64.Valid() {
		return domain.ErrInvalidSID
	}

	newPerson, errGetPerson := u.BySteamID(ctx, transaction, sid64)
	if errGetPerson != nil {
		return errGetPerson
	}

	if (newPerson.DiscordID) != "" {
		return domain.ErrDiscordAlreadyLinked
	}

	newPerson.DiscordID = discordID
	if errSavePerson := u.Save(ctx, transaction, &newPerson); errSavePerson != nil {
		return errors.Join(errSavePerson, domain.ErrSaveChanges)
	}

	slog.Info("Discord steamid set", slog.Int64("sid64", sid64.Int64()), slog.String("discordId", discordID))

	return nil
}

func (u *Persons) BySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (Person, error) {
	results, errQuery := u.repo.Query(ctx, transaction, Query{SteamIDs: []steamid.SteamID{sid64}})
	if errQuery != nil {
		return Person{}, errQuery
	}

	if len(results) != 1 {
		return Person{}, ErrPlayerDoesNotExist
	}

	return results[0], nil
}

func (u *Persons) Drop(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID) error {
	return u.repo.DropPerson(ctx, transaction, steamID)
}

func (u *Persons) Save(ctx context.Context, transaction pgx.Tx, person *Person) error {
	// Don't let owner un-admin themselves.
	if person.SteamID == u.owner && person.PermissionLevel != permission.Admin {
		return permission.ErrDenied
	}

	return u.repo.Save(ctx, transaction, person)
}

func (u *Persons) BySteamIDs(ctx context.Context, transaction pgx.Tx, steamIDs steamid.Collection) (People, error) {
	return u.repo.Query(ctx, transaction, Query{SteamIDs: steamIDs})
}

func (u *Persons) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	people, errQuery := u.repo.Query(ctx, nil, Query{Addr: addr})
	if errQuery != nil {
		return nil, errQuery
	}

	var coll steamid.Collection
	for _, person := range people {
		coll = append(coll, person.SteamID)
	}

	return coll, nil
}

func (u *Persons) GetPeople(ctx context.Context, transaction pgx.Tx, filter Query) (People, error) {
	return u.repo.Query(ctx, transaction, filter)
}

func (u *Persons) GetOrCreatePersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (domain.PersonCore, error) {
	person, errGetPerson := u.BySteamID(ctx, transaction, sid64)
	if errGetPerson != nil && errors.Is(errGetPerson, ErrPlayerDoesNotExist) {
		person = New(sid64)

		if err := u.repo.Save(ctx, transaction, &person); err != nil {
			return domain.PersonCore{}, err
		}
	}

	return domain.PersonCore{
		SteamID:         person.SteamID,
		PermissionLevel: person.PermissionLevel,
		Name:            person.PersonaName,
		Avatarhash:      person.AvatarHash,
	}, nil
}

func (u *Persons) getFirst(ctx context.Context, query Query) (Person, error) {
	people, errPeople := u.repo.Query(ctx, nil, query)
	if errPeople != nil {
		return Person{}, errPeople
	}

	if len(people) != 1 {
		return Person{}, ErrPlayerDoesNotExist
	}

	return people[0], nil
}

func (u *Persons) GetPersonByDiscordID(ctx context.Context, discordID string) (Person, error) {
	return u.getFirst(ctx, Query{DiscordID: discordID})
}

func (u *Persons) GetExpiredProfiles(ctx context.Context, limit uint64) ([]Person, error) {
	return u.repo.Query(ctx, nil, Query{
		Filter: query.Filter{
			Limit: limit,
		},
		SteamUpdateOlderThan: time.Now().AddDate(0, 0, -30),
	})
}

func (u *Persons) GetSteamIDsAbove(ctx context.Context, privilege permission.Privilege) (steamid.Collection, error) {
	var steamIDs steamid.Collection
	players, errPlayers := u.repo.Query(ctx, nil, Query{
		WithPermissions: privilege,
	})

	if errPlayers != nil {
		return steamIDs, errPlayers
	}

	for _, player := range players {
		steamIDs = append(steamIDs, player.SteamID)
	}

	return steamIDs, nil
}

func (u *Persons) GetPersonSettings(ctx context.Context, steamID steamid.SteamID) (Settings, error) {
	settings, errSettings := u.repo.Settings(ctx, steamID)
	if errSettings != nil {
		if !errors.Is(errSettings, database.ErrNoResult) {
			return settings, errSettings
		}

		return Settings{
			SteamID:   steamID,
			CreatedOn: time.Now(),
			UpdatedOn: time.Now(),
		}, nil
	}

	return settings, nil
}

func (u *Persons) SavePersonSettings(ctx context.Context, user domain.PersonInfo, update SettingsUpdate) (Settings, error) {
	settings, err := u.GetPersonSettings(ctx, user.GetSteamID())
	if err != nil {
		return settings, err
	}

	settings.ForumProfileMessages = update.ForumProfileMessages
	settings.StatsHidden = update.StatsHidden
	settings.ForumSignature = stringutil.SanitizeUGC(update.ForumSignature)
	settings.CenterProjectiles = update.CenterProjectiles

	if errSave := u.repo.SaveSettings(ctx, &settings); errSave != nil {
		return settings, errSave
	}

	return settings, nil
}
