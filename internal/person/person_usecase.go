package person

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/sync/errgroup"
)

type PersonUsecase struct {
	config *config.ConfigUsecase
	repo   *PersonRepository
	tfAPI  *thirdparty.TFAPI
}

func NewPersonUsecase(repository *PersonRepository, config *config.ConfigUsecase, tfAPI *thirdparty.TFAPI) *PersonUsecase {
	return &PersonUsecase{
		repo:   repository,
		config: config,
		tfAPI:  tfAPI,
	}
}

func (u PersonUsecase) CanAlter(ctx context.Context, sourceID steamid.SteamID, targetID steamid.SteamID) (bool, error) {
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

func (u PersonUsecase) QueryProfile(ctx context.Context, query string) (ProfileResponse, error) {
	var resp ProfileResponse

	sid, errResolveSID64 := steamid.Resolve(ctx, query)
	if errResolveSID64 != nil {
		return resp, domain.ErrInvalidSID
	}

	person, errGetProfile := u.GetOrCreatePersonBySteamID(ctx, nil, sid)
	if errGetProfile != nil {
		return resp, errGetProfile
	}

	if person.Expired() {
		if err := UpdatePlayerSummary(ctx, &person, u.tfAPI); err != nil {
			slog.Error("Failed to update player summary", log.ErrAttr(err))
		} else {
			if errSave := u.SavePerson(ctx, nil, &person); errSave != nil {
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

func (u PersonUsecase) UpdateProfiles(ctx context.Context, transaction pgx.Tx, people People) (int, error) {
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
		person.IsNew = false
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

		if errSavePerson := u.repo.SavePerson(ctx, transaction, &person); errSavePerson != nil {
			return 0, errors.Join(errSavePerson, domain.ErrUpdatePerson)
		}
	}

	return len(people), nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
func (u PersonUsecase) SetSteam(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID, discordID string) error {
	newPerson, errGetPerson := u.GetOrCreatePersonBySteamID(ctx, transaction, sid64)
	if errGetPerson != nil || !sid64.Valid() {
		return domain.ErrInvalidSID
	}

	if (newPerson.DiscordID) != "" {
		return domain.ErrDiscordAlreadyLinked
	}

	newPerson.DiscordID = discordID
	if errSavePerson := u.SavePerson(ctx, transaction, &newPerson); errSavePerson != nil {
		return errors.Join(errSavePerson, domain.ErrSaveChanges)
	}

	slog.Info("Discord steamid set", slog.Int64("sid64", sid64.Int64()), slog.String("discordId", discordID))

	return nil
}

func (u PersonUsecase) GetPersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (Person, error) {
	return u.repo.GetPersonBySteamID(ctx, transaction, sid64)
}

func (u PersonUsecase) DropPerson(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID) error {
	return u.repo.DropPerson(ctx, transaction, steamID)
}

func (u PersonUsecase) SavePerson(ctx context.Context, transaction pgx.Tx, person *Person) error {
	return u.repo.SavePerson(ctx, transaction, person)
}

func (u PersonUsecase) GetPeopleBySteamID(ctx context.Context, transaction pgx.Tx, steamIDs steamid.Collection) (People, error) {
	return u.repo.GetPeopleBySteamID(ctx, transaction, steamIDs)
}

func (u PersonUsecase) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	return u.repo.GetSteamsAtAddress(ctx, addr)
}

func (u PersonUsecase) GetPeople(ctx context.Context, transaction pgx.Tx, filter PlayerQuery) (People, int64, error) {
	return u.repo.GetPeople(ctx, transaction, filter)
}

func (u PersonUsecase) GetOrCreatePersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (Person, error) {
	person, errGetPerson := u.repo.GetPersonBySteamID(ctx, transaction, sid64)
	if errGetPerson != nil && errors.Is(errGetPerson, database.ErrNoResult) {
		person = NewPerson(sid64)

		if err := u.repo.SavePerson(ctx, transaction, &person); err != nil {
			return person, err
		}
	}

	return person, nil
}

func (u PersonUsecase) GetPersonByDiscordID(ctx context.Context, discordID string) (Person, error) {
	return u.repo.GetPersonByDiscordID(ctx, discordID)
}

func (u PersonUsecase) GetExpiredProfiles(ctx context.Context, limit uint64) ([]Person, error) {
	return u.repo.GetExpiredProfiles(ctx, nil, limit)
}

func (u PersonUsecase) GetPersonMessageByID(ctx context.Context, personMessageID int64) (chat.PersonMessage, error) {
	return u.repo.GetPersonMessageByID(ctx, personMessageID)
}

func (u PersonUsecase) GetSteamIDsAbove(ctx context.Context, privilege permission.Privilege) (steamid.Collection, error) {
	return u.repo.GetSteamIDsAbove(ctx, privilege)
}

func (u PersonUsecase) GetSteamIDsByGroups(ctx context.Context, privileges []permission.Privilege) (steamid.Collection, error) {
	return u.repo.GetSteamIDsByGroups(ctx, privileges)
}

func (u PersonUsecase) GetPersonSettings(ctx context.Context, steamID steamid.SteamID) (PersonSettings, error) {
	return u.repo.GetPersonSettings(ctx, steamID)
}

func (u PersonUsecase) SavePersonSettings(ctx context.Context, user domain.PersonInfo, update PersonSettingsUpdate) (PersonSettings, error) {
	settings, err := u.GetPersonSettings(ctx, user.GetSteamID())
	if err != nil {
		return settings, err
	}

	settings.ForumProfileMessages = update.ForumProfileMessages
	settings.StatsHidden = update.StatsHidden
	settings.ForumSignature = stringutil.SanitizeUGC(update.ForumSignature)
	settings.CenterProjectiles = update.CenterProjectiles

	if errSave := u.repo.SavePersonSettings(ctx, &settings); errSave != nil {
		return settings, errSave
	}

	return settings, nil
}

func (u PersonUsecase) SetPermissionLevel(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID, level permission.Privilege) error {
	person, errGet := u.GetPersonBySteamID(ctx, transaction, steamID)
	if errGet != nil {
		return errGet
	}

	// Don't let admins un-admin themselves.
	if steamID == steamid.New(u.config.Config().Owner) {
		return permission.ErrPermissionDenied
	}

	person.PermissionLevel = level

	if errSave := u.SavePerson(ctx, transaction, &person); errSave != nil {
		return errSave
	}

	return u.repo.SavePerson(ctx, transaction, &person)
}
