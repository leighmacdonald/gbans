package person

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"golang.org/x/sync/errgroup"
)

type personUsecase struct {
	configUsecase domain.ConfigUsecase
	personRepo    domain.PersonRepository
}

func NewPersonUsecase(repository domain.PersonRepository, configUsecase domain.ConfigUsecase) domain.PersonUsecase {
	return &personUsecase{
		personRepo:    repository,
		configUsecase: configUsecase,
	}
}

func (u personUsecase) QueryProfile(ctx context.Context, query string) (domain.ProfileResponse, error) {
	var resp domain.ProfileResponse

	sid, errResolveSID64 := steamid.Resolve(ctx, query)
	if errResolveSID64 != nil {
		return resp, domain.ErrInvalidSID
	}

	person, errGetProfile := u.GetOrCreatePersonBySteamID(ctx, sid)
	if errGetProfile != nil {
		return resp, errGetProfile
	}

	if person.Expired() {
		if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
			slog.Error("Failed to update player summary", log.ErrAttr(err))
		} else {
			if errSave := u.SavePerson(ctx, &person); errSave != nil {
				slog.Error("Failed to save person summary", log.ErrAttr(errSave))
			}
		}
	}

	friendList, errFetchFriends := steamweb.GetFriendList(ctx, person.SteamID)
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

func (u personUsecase) updateProfiles(ctx context.Context, people domain.People) (int, error) {
	if len(people) > 100 {
		return 0, domain.ErrSteamAPIArgLimit
	}

	var (
		banStates           []steamweb.PlayerBanState
		summaries           []steamweb.PlayerSummary
		steamIDs            = people.ToSteamIDCollection()
		errGroup, cancelCtx = errgroup.WithContext(ctx)
	)

	errGroup.Go(func() error {
		newBanStates, errBans := thirdparty.FetchPlayerBans(cancelCtx, steamIDs)
		if errBans != nil {
			return errors.Join(errBans, domain.ErrFetchSteamBans)
		}

		banStates = newBanStates

		return nil
	})

	errGroup.Go(func() error {
		newSummaries, errSummaries := steamweb.PlayerSummaries(cancelCtx, steamIDs)
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
			if person.SteamID != summary.SteamID {
				continue
			}

			person.PlayerSummary = &summary

			break
		}

		for _, banState := range banStates {
			if person.SteamID != banState.SteamID {
				continue
			}

			person.CommunityBanned = banState.CommunityBanned
			person.VACBans = banState.NumberOfVACBans
			person.GameBans = banState.NumberOfGameBans
			person.EconomyBan = banState.EconomyBan
			person.CommunityBanned = banState.CommunityBanned
			person.DaysSinceLastBan = banState.DaysSinceLastBan
		}

		if errSavePerson := u.personRepo.SavePerson(ctx, &person); errSavePerson != nil {
			return 0, errors.Join(errSavePerson, domain.ErrUpdatePerson)
		}
	}

	return len(people), nil
}

// Start takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution.
func (u personUsecase) Start(ctx context.Context) {
	var (
		run    = make(chan any)
		ticker = time.NewTicker(time.Second * 300)
	)

	go func() {
		run <- true
	}()

	for {
		select {
		case <-ticker.C:
			run <- true
		case <-run:
			localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
			people, errGetExpired := u.personRepo.GetExpiredProfiles(localCtx, 100)

			if errGetExpired != nil || len(people) == 0 {
				cancel()

				continue
			}

			count, errUpdate := u.updateProfiles(localCtx, people)
			if errUpdate != nil {
				slog.Error("Failed to update profiles", log.ErrAttr(errUpdate))
			}

			slog.Debug("Updated steam profiles and vac data", slog.Int("count", count))

			cancel()
		case <-ctx.Done():
			slog.Debug("profileUpdater shutting down")

			return
		}
	}
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
func (u personUsecase) SetSteam(ctx context.Context, sid64 steamid.SteamID, discordID string) error {
	newPerson, errGetPerson := u.GetOrCreatePersonBySteamID(ctx, sid64)
	if errGetPerson != nil || !sid64.Valid() {
		return domain.ErrInvalidSID
	}

	if (newPerson.DiscordID) != "" {
		return domain.ErrDiscordAlreadyLinked
	}

	newPerson.DiscordID = discordID
	if errSavePerson := u.SavePerson(ctx, &newPerson); errSavePerson != nil {
		return errors.Join(errSavePerson, domain.ErrSaveChanges)
	}

	slog.Info("Discord steamid set", slog.Int64("sid64", sid64.Int64()), slog.String("discordId", discordID))

	return nil
}

func (u personUsecase) GetPersonBySteamID(ctx context.Context, sid64 steamid.SteamID) (domain.Person, error) {
	return u.personRepo.GetPersonBySteamID(ctx, sid64)
}

func (u personUsecase) DropPerson(ctx context.Context, steamID steamid.SteamID) error {
	return u.personRepo.DropPerson(ctx, steamID)
}

func (u personUsecase) SavePerson(ctx context.Context, person *domain.Person) error {
	return u.personRepo.SavePerson(ctx, person)
}

func (u personUsecase) GetPeopleBySteamID(ctx context.Context, steamIDs steamid.Collection) (domain.People, error) {
	return u.personRepo.GetPeopleBySteamID(ctx, steamIDs)
}

func (u personUsecase) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	return u.personRepo.GetSteamsAtAddress(ctx, addr)
}

func (u personUsecase) GetPeople(ctx context.Context, filter domain.PlayerQuery) (domain.People, int64, error) {
	return u.personRepo.GetPeople(ctx, filter)
}

func (u personUsecase) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SteamID) (domain.Person, error) {
	person, errGetPerson := u.personRepo.GetPersonBySteamID(ctx, sid64)
	if errGetPerson != nil && errors.Is(errGetPerson, domain.ErrNoResult) {
		// FIXME
		person = domain.NewPerson(sid64)

		if err := u.personRepo.SavePerson(ctx, &person); err != nil {
			return person, err
		}
	}

	return person, nil
}

func (u personUsecase) GetPersonByDiscordID(ctx context.Context, discordID string) (domain.Person, error) {
	return u.personRepo.GetPersonByDiscordID(ctx, discordID)
}

func (u personUsecase) GetExpiredProfiles(ctx context.Context, limit uint64) ([]domain.Person, error) {
	return u.personRepo.GetExpiredProfiles(ctx, limit)
}

func (u personUsecase) GetPersonMessageByID(ctx context.Context, personMessageID int64) (domain.PersonMessage, error) {
	return u.personRepo.GetPersonMessageByID(ctx, personMessageID)
}

func (u personUsecase) GetSteamIDsAbove(ctx context.Context, privilege domain.Privilege) (steamid.Collection, error) {
	return u.personRepo.GetSteamIDsAbove(ctx, privilege)
}

func (u personUsecase) GetPersonSettings(ctx context.Context, steamID steamid.SteamID) (domain.PersonSettings, error) {
	return u.personRepo.GetPersonSettings(ctx, steamID)
}

func (u personUsecase) SavePersonSettings(ctx context.Context, user domain.PersonInfo, update domain.PersonSettingsUpdate) (domain.PersonSettings, error) {
	settings, err := u.GetPersonSettings(ctx, user.GetSteamID())
	if err != nil {
		return settings, err
	}

	settings.ForumProfileMessages = update.ForumProfileMessages
	settings.StatsHidden = update.StatsHidden
	settings.ForumSignature = util.SanitizeUGC(update.ForumSignature)

	if errSave := u.personRepo.SavePersonSettings(ctx, &settings); errSave != nil {
		return settings, errSave
	}

	return settings, nil
}

func (u personUsecase) SetPermissionLevel(ctx context.Context, steamID steamid.SteamID, level domain.Privilege) error {
	person, errGet := u.GetPersonBySteamID(ctx, steamID)
	if errGet != nil {
		return errGet
	}

	// Don't let admins un-admin themselves.
	if steamID == steamid.New(u.configUsecase.Config().General.Owner) {
		return domain.ErrPermissionDenied
	}

	person.PermissionLevel = level

	if errSave := u.SavePerson(ctx, &person); errSave != nil {
		return errSave
	}

	return u.personRepo.SavePerson(ctx, &person)
}
