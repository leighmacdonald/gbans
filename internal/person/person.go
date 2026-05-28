package person

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/sync/errgroup"
)

var (
	ErrPlayerDoesNotExist   = errors.New("player does not exist")
	ErrDiscordAlreadyLinked = errors.New("discord account is already linked")
	ErrSteamAPIArgLimit     = errors.New("steam api support a max of 100 steam ids")
	ErrFetchSteamBans       = errors.New("failed to fetch ban status from steam api")
	ErrSteamAPISummaries    = errors.New("failed to fetch player summaries")
	ErrSteamAPI             = errors.New("steam api requests have errors")
	ErrUpdatePerson         = errors.New("failed to save updated person profile")
)

type SteamMember interface {
	IsMember(steamID steamid.SteamID) (int64, bool)
}

type Query struct {
	query.Filter

	PersonaName          string
	WithPermissions      []permission.Privilege
	DiscordID            string
	SteamUpdateOlderThan time.Time
	SteamIDs             []string
	VacBans              int32
	GameBans             int32
	AvatarHash           string
	CommunityBanned      *bool
	TimeCreatedAfter     *time.Time
	TimeCreatedBefore    *time.Time
}

type RequestPermissionLevelUpdate struct {
	PermissionLevel permission.Privilege
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
	SteamID           steamid.SteamID
	CreatedOn         time.Time
	UpdatedOn         time.Time
	PermissionLevel   permission.Privilege
	Muted             bool
	isNew             bool
	DiscordID         string
	PatreonID         string
	IPAddr            netip.Addr
	CommunityBanned   bool
	VACBans           int32
	GameBans          int32
	EconomyBan        EconBanState
	DaysSinceLastBan  int32
	UpdatedOnSteam    time.Time
	AvatarHash        string
	CommentPermission int32
	LastLogoff        *time.Time
	LocCityID         int32
	LocCountryCode    string
	LocStateCode      string
	PersonaName       string
	PersonaState      int32
	PersonaStateFlags int32
	PrimaryClanID     string
	ProfileState      int32
	ProfileURL        string
	RealName          string
	TimeCreated       int64
	VisibilityState   int32
	BanID             int32
}

func (p Person) GetPrivilege() permission.Privilege {
	return p.PermissionLevel
}

func (p Person) ApplySteamInfo(summary thirdparty.PlayerSummaryResponse, steamBan thirdparty.SteamBan) Person {
	p.PersonaName = summary.PersonaName
	p.AvatarHash = summary.AvatarHash
	p.LocCityID = int32(summary.LocCityId) //nolint:gosec
	p.LocCountryCode = summary.LocCountryCode
	p.LastLogoff = new(time.Unix(summary.LastLogoff, 0))
	p.LocStateCode = summary.LocStateCode
	p.VisibilityState = int32(summary.VisibilityState)     //nolint:gosec
	p.PersonaState = int32(summary.PersonaState)           //nolint:gosec
	p.PersonaStateFlags = int32(summary.PersonaStateFlags) //nolint:gosec
	p.PrimaryClanID = summary.PrimaryClanId
	p.ProfileState = int32(summary.ProfileState) //nolint:gosec
	p.RealName = summary.RealName
	p.TimeCreated = summary.TimeCreated
	p.CommentPermission = int32(summary.CommentPermission) //nolint:gosec
	p.VACBans = int32(steamBan.NumberOfVacBans)            //nolint:gosec
	p.GameBans = int32(steamBan.NumberOfGameBans)          //nolint:gosec
	p.DaysSinceLastBan = int32(steamBan.DaysSinceLastBan)  //nolint:gosec
	p.CommunityBanned = steamBan.CommunityBanned
	p.EconomyBan = EconBanState(steamBan.EconomyBan) //nolint:gosec
	p.UpdatedOn = time.Now()
	p.UpdatedOnSteam = time.Now()

	return p
}

func (p Person) GetVACBans() int32 {
	return p.VACBans
}

func (p Person) GetGameBans() int32 {
	return p.GameBans
}

func (p Person) GetTimeCreated() time.Time {
	return time.Unix(p.TimeCreated, 0)
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

func (p Person) GetAvatar() person.Avatar {
	return person.Avatar(p.AvatarHash)
}

func (p Person) GetSteamID() steamid.SteamID {
	return p.SteamID
}

func (p Person) GetSteamIDString() string {
	return p.SteamID.String()
}

func (p Person) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

// LoggedIn checks for a valid steamID.
func (p Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

func (p Person) Core() person.Core {
	return person.Core{
		SteamID:         p.SteamID,
		PermissionLevel: p.PermissionLevel,
		Name:            p.GetName(),
		Avatarhash:      p.AvatarHash,
		DiscordID:       p.DiscordID,
		PatreonID:       p.PatreonID,
		VacBans:         p.VACBans,
		GameBans:        p.GameBans,
		TimeCreated:     p.CreatedOn,
	}
}

type ProfileResponse struct {
	Player   *Person
	Friends  []thirdparty.SteamFriend
	Settings Settings
}

// New allocates a new default person instance.
func New(sid64 steamid.SteamID) Person {
	curTime := time.Now()

	return Person{
		SteamID:          sid64,
		CreatedOn:        curTime,
		UpdatedOn:        curTime,
		PermissionLevel:  permission.User,
		Muted:            false,
		isNew:            true,
		DiscordID:        "",
		CommunityBanned:  false,
		VACBans:          0,
		GameBans:         0,
		EconomyBan:       "none",
		DaysSinceLastBan: 0,
		UpdatedOnSteam:   time.Unix(0, 0),
	}
}

type People []Person

func (p People) ToSteamIDCollection() steamid.Collection {
	collection := make(steamid.Collection, len(p))
	for idx, player := range p {
		collection[idx] = player.SteamID
	}

	return collection
}

func (p People) AsMap() map[steamid.SteamID]Person {
	m := map[steamid.SteamID]Person{}
	for _, player := range p {
		m[player.SteamID] = player
	}

	return m
}

type Settings struct {
	PersonSettingsID     int64
	SteamID              steamid.SteamID
	ForumSignature       string
	ForumProfileMessages bool
	StatsHidden          bool

	// This key will be absent to indicate that this feature
	// is disabled (and UI should not be shown to the user).
	CenterProjectiles *bool

	CreatedOn time.Time
	UpdatedOn time.Time
}

type SettingsUpdate struct {
	ForumSignature       string
	ForumProfileMessages bool
	StatsHidden          bool
	CenterProjectiles    *bool
}

type Persons struct {
	owner          steamid.SteamID
	repo           Repository
	tfAPI          thirdparty.APIProvider
	knownPersonsMu *sync.RWMutex
	knownPersons   map[steamid.SteamID]bool
}

func NewPersons(repository Repository, owner steamid.SteamID, tfAPI thirdparty.APIProvider) *Persons {
	return &Persons{repo: repository, owner: owner, tfAPI: tfAPI, knownPersons: map[steamid.SteamID]bool{}, knownPersonsMu: &sync.RWMutex{}}
}

func (u *Persons) CanAlter(ctx context.Context, sourceID steamid.SteamID, targetID steamid.SteamID) (bool, error) {
	source, errSource := u.GetOrCreatePersonBySteamID(ctx, sourceID)
	if errSource != nil {
		return false, errSource
	}

	target, errGetProfile := u.GetOrCreatePersonBySteamID(ctx, targetID)
	if errGetProfile != nil {
		return false, errGetProfile
	}

	return source.PermissionLevel > target.PermissionLevel, nil
}

func (u *Persons) QueryProfile(ctx context.Context, query string) (ProfileResponse, error) {
	var resp ProfileResponse

	sid, errResolveSID64 := steamid.Resolve(ctx, query)
	if errResolveSID64 != nil {
		return resp, steamid.ErrInvalidSID
	}

	if err := u.EnsurePerson(ctx, sid); err != nil {
		return resp, err
	}

	player, errGetProfile := u.BySteamID(ctx, sid)
	if errGetProfile != nil {
		return resp, errGetProfile
	}

	if player.Expired() {
		if err := UpdatePlayerSummary(ctx, &player, u.tfAPI); err != nil {
			slog.Warn("Failed to update player summary", slog.String("error", err.Error()))
			player.UpdatedOnSteam = time.Now()
		}
		if errSave := u.Save(ctx, &player); errSave != nil {
			slog.Error("Failed to save person summary", slog.String("error", errSave.Error()))
		}
	}

	// friendList, errFetchFriends := u.tfAPI.Friends(ctx, player.SteamID)
	// if errFetchFriends == nil {
	// 	resp.Friends = friendList
	// }

	resp.Player = &player

	settings, err := u.GetPersonSettings(ctx, sid)
	if err != nil {
		return resp, err
	}

	resp.Settings = settings

	return resp, nil
}

func (u *Persons) UpdateProfiles(ctx context.Context, _ pgx.Tx, people People) (int, error) {
	if len(people) > 100 {
		return 0, ErrSteamAPIArgLimit
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
			return errors.Join(errBans, ErrFetchSteamBans)
		}

		banStates = newBanStates

		return nil
	})

	errGroup.Go(func() error {
		newSummaries, errSummaries := u.tfAPI.Summaries(ctx, steamIDs)
		if errSummaries != nil {
			return errors.Join(errSummaries, ErrSteamAPISummaries)
		}

		summaries = newSummaries

		return nil
	})

	if errFetch := errGroup.Wait(); errFetch != nil {
		return 0, errors.Join(errFetch, ErrSteamAPI)
	}

	for _, player := range people {
		player.isNew = false
		player.UpdatedOnSteam = time.Now()

		for _, newSummary := range summaries {
			summary := newSummary
			if player.SteamID.String() != summary.SteamId {
				continue
			}

			player.AvatarHash = summary.AvatarHash
			player.CommentPermission = int32(summary.CommentPermission) //nolint:gosec
			player.LastLogoff = new(time.Unix(summary.LastLogoff, 0))
			player.LocCityID = int32(summary.LocCityId) //nolint:gosec
			player.LocCountryCode = summary.LocCountryCode
			player.LocStateCode = summary.LocStateCode
			player.PersonaName = summary.PersonaName
			player.PersonaState = int32(summary.PersonaState)           //nolint:gosec
			player.PersonaStateFlags = int32(summary.PersonaStateFlags) //nolint:gosec
			player.PrimaryClanID = summary.PrimaryClanId
			player.ProfileState = int32(summary.ProfileState) //nolint:gosec
			player.ProfileURL = summary.ProfileUrl
			player.RealName = summary.RealName
			player.TimeCreated = summary.TimeCreated
			player.VisibilityState = int32(summary.VisibilityState) //nolint:gosec

			break
		}

		for _, banState := range banStates {
			if player.SteamID.String() != banState.SteamId {
				continue
			}

			player.CommunityBanned = banState.CommunityBanned
			player.VACBans = int32(banState.NumberOfVacBans)   //nolint:gosec
			player.GameBans = int32(banState.NumberOfGameBans) //nolint:gosec
			player.EconomyBan = EconBanState(banState.EconomyBan)
			player.CommunityBanned = banState.CommunityBanned
			player.DaysSinceLastBan = int32(banState.DaysSinceLastBan) //nolint:gosec
		}

		if errSavePerson := u.repo.Save(ctx, &player); errSavePerson != nil {
			return 0, errors.Join(errSavePerson, ErrUpdatePerson)
		}
	}

	return len(people), nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
func (u *Persons) SetSteam(ctx context.Context, sid64 steamid.SteamID, discordID string) error {
	if !sid64.Valid() {
		return steamid.ErrInvalidSID
	}

	newPerson, errGetPerson := u.BySteamID(ctx, sid64)
	if errGetPerson != nil {
		return errGetPerson
	}

	if (newPerson.DiscordID) != "" {
		return ErrDiscordAlreadyLinked
	}

	newPerson.DiscordID = discordID
	if errSavePerson := u.Save(ctx, &newPerson); errSavePerson != nil {
		return errors.Join(errSavePerson, database.ErrSaveChanges)
	}

	slog.Info("Discord steamid set", slog.Int64("sid64", sid64.Int64()), slog.String("discordId", discordID))

	return nil
}

func (u *Persons) BySteamID(ctx context.Context, steamID steamid.SteamID) (Person, error) {
	return u.getFirst(ctx, Query{SteamIDs: []string{steamID.String()}})
}

func (u *Persons) Drop(ctx context.Context, steamID steamid.SteamID) error {
	return u.repo.DropPerson(ctx, steamID)
}

func (u *Persons) Save(ctx context.Context, person *Person) error {
	if person == nil {
		return permission.ErrDenied
	}
	// Don't let owner un-admin themselves.
	if person.SteamID == u.owner && person.PermissionLevel != permission.Admin {
		return permission.ErrDenied
	}

	return u.repo.Save(ctx, person)
}

func (u *Persons) BySteamIDs(ctx context.Context, steamIDs steamid.Collection) (People, error) {
	people, _, err := u.repo.Query(ctx, Query{SteamIDs: steamIDs.ToStringSlice()})
	if err != nil {
		return nil, err
	}

	return people, nil
}

func (u *Persons) GetPeople(ctx context.Context, filter Query) (People, int64, error) {
	if (filter.TimeCreatedAfter != nil && filter.TimeCreatedBefore != nil) &&
		(filter.TimeCreatedAfter.Before(*filter.TimeCreatedBefore) || filter.TimeCreatedBefore.After(*filter.TimeCreatedAfter)) {
		filter.TimeCreatedAfter, filter.TimeCreatedBefore = filter.TimeCreatedBefore, filter.TimeCreatedAfter
	}

	return u.repo.Query(ctx, filter)
}

func (u *Persons) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SteamID) (person.Core, error) {
	fetchedPerson, errGetPerson := u.BySteamID(ctx, sid64)
	if errGetPerson != nil && errors.Is(errGetPerson, ErrPlayerDoesNotExist) {
		fetchedPerson = New(sid64)
		if err := u.repo.Save(ctx, &fetchedPerson); err != nil {
			return person.Core{}, err
		}
	}

	if fetchedPerson.Expired() {
		if errUpdate := u.updatePerson(ctx, &fetchedPerson); errUpdate != nil {
			slog.Error("Failed to update steam profile data", slog.String("steamid", sid64.String()),
				slog.String("error", errUpdate.Error()))
		}
	}

	return person.Core{
		SteamID:         fetchedPerson.SteamID,
		PermissionLevel: fetchedPerson.PermissionLevel,
		Name:            fetchedPerson.PersonaName,
		Avatarhash:      fetchedPerson.AvatarHash,
		DiscordID:       fetchedPerson.DiscordID,
		PatreonID:       fetchedPerson.PatreonID,
		GameBans:        fetchedPerson.GameBans,
		VacBans:         fetchedPerson.VACBans,
		TimeCreated:     time.Unix(fetchedPerson.TimeCreated, 0),
	}, nil
}

func (u *Persons) updatePerson(ctx context.Context, person *Person) error {
	if u.tfAPI == nil {
		return nil
	}
	summaries, errSummaries := u.tfAPI.Summaries(ctx, []steamid.SteamID{person.SteamID})
	if errSummaries != nil {
		return errSummaries
	}

	if len(summaries) != 1 {
		return ErrSteamUpdate
	}

	vacInfo, errVacInfo := u.tfAPI.SteamBans(ctx, []steamid.SteamID{person.SteamID})
	if errVacInfo != nil {
		return errVacInfo
	}

	if len(vacInfo) != 1 {
		return ErrSteamUpdate
	}

	person.ApplySteamInfo(summaries[0], vacInfo[0])

	return u.Save(ctx, person)
}

func (u *Persons) getFirst(ctx context.Context, query Query) (Person, error) {
	people, _, errPeople := u.repo.Query(ctx, query)
	if errPeople != nil {
		return Person{}, errPeople
	}

	if len(people) != 1 {
		return Person{}, ErrPlayerDoesNotExist
	}

	player := people[0]
	if player.Expired() {
		if errGetProfile := UpdatePlayerSummary(ctx, &player, u.tfAPI); errGetProfile != nil {
			slog.Warn("Failed to update player summary for expired player", slog.String("error", errGetProfile.Error()))
		}
	}

	return player, nil
}

func (u *Persons) GetPersonByDiscordID(ctx context.Context, discordID string) (person.Core, error) {
	fetchedPerson, errGetPerson := u.getFirst(ctx, Query{DiscordID: discordID})
	if errGetPerson != nil {
		return person.Core{}, errGetPerson
	}

	return person.Core{
		SteamID:         fetchedPerson.SteamID,
		PermissionLevel: fetchedPerson.PermissionLevel,
		Name:            fetchedPerson.PersonaName,
		Avatarhash:      fetchedPerson.AvatarHash,
		PatreonID:       fetchedPerson.PatreonID,
		GameBans:        fetchedPerson.GameBans,
		VacBans:         fetchedPerson.VACBans,
		TimeCreated:     time.Unix(fetchedPerson.TimeCreated, 0),
		DiscordID:       fetchedPerson.DiscordID,
	}, nil
}

func (u *Persons) GetExpiredProfiles(ctx context.Context, limit uint64) ([]Person, int64, error) {
	return u.repo.Query(ctx, Query{
		Filter: query.Filter{
			Limit: limit,
		},
		SteamUpdateOlderThan: time.Now().AddDate(0, 0, -30),
	})
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

func (u *Persons) SavePersonSettings(ctx context.Context, user person.BaseUser, update SettingsUpdate) (Settings, error) {
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

func (u *Persons) EnsurePerson(ctx context.Context, steamID steamid.SteamID) error {
	u.knownPersonsMu.RLock()
	_, exists := u.knownPersons[steamID]
	if exists {
		u.knownPersonsMu.RUnlock()

		return nil
	}
	u.knownPersonsMu.RUnlock()

	if _, err := u.GetOrCreatePersonBySteamID(ctx, steamID); err != nil {
		return err
	}

	u.knownPersonsMu.Lock()
	u.knownPersons[steamID] = true
	u.knownPersonsMu.Unlock()

	return nil
}
