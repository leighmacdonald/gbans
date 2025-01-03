package person

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"golang.org/x/sync/errgroup"
)

type personUsecase struct {
	config domain.ConfigUsecase
	repo   domain.PersonRepository
}

func NewPersonUsecase(repository domain.PersonRepository, config domain.ConfigUsecase) domain.PersonUsecase {
	return &personUsecase{
		repo:   repository,
		config: config,
	}
}

func (u personUsecase) QueryProfile(ctx context.Context, query string) (domain.ProfileResponse, error) {
	var resp domain.ProfileResponse

	sid, errResolveSID64 := steamid.Resolve(ctx, query)
	if errResolveSID64 != nil {
		return resp, domain.ErrInvalidSID
	}

	person, errGetProfile := u.GetOrCreatePersonBySteamID(ctx, nil, sid)
	if errGetProfile != nil {
		return resp, errGetProfile
	}

	if person.Expired() {
		if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
			slog.Error("Failed to update player summary", log.ErrAttr(err))
		} else {
			if errSave := u.SavePerson(ctx, nil, &person); errSave != nil {
				slog.Error("Failed to save person summary", log.ErrAttr(errSave))
			}
		}
	}

	friendList, errFetchFriends := steamweb.GetFriendList(ctx, httphelper.NewHTTPClient(), person.SteamID)
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

func (u personUsecase) UpdateProfiles(ctx context.Context, transaction pgx.Tx, people domain.People) (int, error) {
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
		newSummaries, errSummaries := steamweb.PlayerSummaries(cancelCtx, httphelper.NewHTTPClient(), steamIDs)
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

		if errSavePerson := u.repo.SavePerson(ctx, transaction, &person); errSavePerson != nil {
			return 0, errors.Join(errSavePerson, domain.ErrUpdatePerson)
		}
	}

	return len(people), nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
func (u personUsecase) SetSteam(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID, discordID string) error {
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

func (u personUsecase) GetPersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (domain.Person, error) {
	return u.repo.GetPersonBySteamID(ctx, transaction, sid64)
}

func (u personUsecase) DropPerson(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID) error {
	return u.repo.DropPerson(ctx, transaction, steamID)
}

func (u personUsecase) SavePerson(ctx context.Context, transaction pgx.Tx, person *domain.Person) error {
	return u.repo.SavePerson(ctx, transaction, person)
}

func (u personUsecase) GetPeopleBySteamID(ctx context.Context, transaction pgx.Tx, steamIDs steamid.Collection) (domain.People, error) {
	return u.repo.GetPeopleBySteamID(ctx, transaction, steamIDs)
}

func (u personUsecase) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	return u.repo.GetSteamsAtAddress(ctx, addr)
}

func (u personUsecase) GetPeople(ctx context.Context, transaction pgx.Tx, filter domain.PlayerQuery) (domain.People, int64, error) {
	return u.repo.GetPeople(ctx, transaction, filter)
}

func (u personUsecase) GetOrCreatePersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (domain.Person, error) {
	person, errGetPerson := u.repo.GetPersonBySteamID(ctx, transaction, sid64)
	if errGetPerson != nil && errors.Is(errGetPerson, domain.ErrNoResult) {
		person = domain.NewPerson(sid64)

		if err := u.repo.SavePerson(ctx, transaction, &person); err != nil {
			return person, err
		}
	}

	return person, nil
}

func (u personUsecase) GetPersonByDiscordID(ctx context.Context, discordID string) (domain.Person, error) {
	return u.repo.GetPersonByDiscordID(ctx, discordID)
}

func (u personUsecase) GetExpiredProfiles(ctx context.Context, limit uint64) ([]domain.Person, error) {
	return u.repo.GetExpiredProfiles(ctx, nil, limit)
}

func (u personUsecase) GetPersonMessageByID(ctx context.Context, personMessageID int64) (domain.PersonMessage, error) {
	return u.repo.GetPersonMessageByID(ctx, personMessageID)
}

func (u personUsecase) GetSteamIDsAbove(ctx context.Context, privilege domain.Privilege) (steamid.Collection, error) {
	return u.repo.GetSteamIDsAbove(ctx, privilege)
}

func (u personUsecase) GetSteamIDsByGroups(ctx context.Context, privileges []domain.Privilege) (steamid.Collection, error) {
	return u.repo.GetSteamIDsByGroups(ctx, privileges)
}

func (u personUsecase) GetPersonSettings(ctx context.Context, steamID steamid.SteamID) (domain.PersonSettings, error) {
	return u.repo.GetPersonSettings(ctx, steamID)
}

func (u personUsecase) SavePersonSettings(ctx context.Context, user domain.PersonInfo, update domain.PersonSettingsUpdate) (domain.PersonSettings, error) {
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

func (u personUsecase) SetPermissionLevel(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID, level domain.Privilege) error {
	person, errGet := u.GetPersonBySteamID(ctx, transaction, steamID)
	if errGet != nil {
		return errGet
	}

	// Don't let admins un-admin themselves.
	if steamID == steamid.New(u.config.Config().Owner) {
		return domain.ErrPermissionDenied
	}

	person.PermissionLevel = level

	if errSave := u.SavePerson(ctx, transaction, &person); errSave != nil {
		return errSave
	}

	return u.repo.SavePerson(ctx, transaction, &person)
}
