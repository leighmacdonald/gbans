package person

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type personUsecase struct {
	personRepo domain.PersonRepository
	log        *zap.Logger
}

func NewPersonUsecase(log *zap.Logger, repository domain.PersonRepository) domain.PersonUsecase {
	return &personUsecase{
		log:        log,
		personRepo: repository,
	}
}

func (p *personUsecase) updateProfiles(ctx context.Context, people domain.People) (int, error) {
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

		if errSavePerson := p.personRepo.SavePerson(ctx, &person); errSavePerson != nil {
			return 0, errors.Join(errSavePerson, domain.ErrUpdatePerson)
		}
	}

	return len(people), nil
}

// Start takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution.
func (p *personUsecase) Start(ctx context.Context) {
	var (
		log    = p.log.Named("profileUpdate")
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
			people, errGetExpired := p.personRepo.GetExpiredProfiles(localCtx, 100)

			if errGetExpired != nil || len(people) == 0 {
				cancel()

				continue
			}

			count, errUpdate := p.updateProfiles(localCtx, people)
			if errUpdate != nil {
				log.Error("Failed to update profiles", zap.Error(errUpdate))
			}

			p.log.Debug("Updated steam profiles and vac data", zap.Int("count", count))

			cancel()
		case <-ctx.Done():
			log.Debug("profileUpdater shutting down")

			return
		}
	}
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
func (p *personUsecase) SetSteam(ctx context.Context, sid64 steamid.SID64, discordID string) error {
	newPerson := domain.NewPerson(sid64)
	if errGetPerson := p.GetOrCreatePersonBySteamID(ctx, sid64, &newPerson); errGetPerson != nil || !sid64.Valid() {
		return domain.ErrInvalidSID
	}

	if (newPerson.DiscordID) != "" {
		return domain.ErrDiscordAlreadyLinked
	}

	newPerson.DiscordID = discordID
	if errSavePerson := p.SavePerson(ctx, &newPerson); errSavePerson != nil {
		return errors.Join(errSavePerson, domain.ErrSaveChanges)
	}

	p.log.Info("Discord steamid set", zap.Int64("sid64", sid64.Int64()), zap.String("discordId", discordID))

	return nil
}

func (p *personUsecase) GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	return p.personRepo.GetPersonBySteamID(ctx, sid64, person)
}

func (p *personUsecase) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	return p.personRepo.DropPerson(ctx, steamID)
}

func (p *personUsecase) SavePerson(ctx context.Context, person *domain.Person) error {
	return p.personRepo.SavePerson(ctx, person)
}

func (p *personUsecase) GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (domain.People, error) {
	return p.personRepo.GetPeopleBySteamID(ctx, steamIds)
}

func (p *personUsecase) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	return p.personRepo.GetSteamsAtAddress(ctx, addr)
}

func (p *personUsecase) GetPeople(ctx context.Context, filter domain.PlayerQuery) (domain.People, int64, error) {
	return p.personRepo.GetPeople(ctx, filter)
}

func (p *personUsecase) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	errGetPerson := p.personRepo.GetPersonBySteamID(ctx, sid64, person)
	if errGetPerson != nil && errors.Is(errGetPerson, domain.ErrNoResult) {
		// FIXME
		newPerson := domain.NewPerson(sid64)
		*person = newPerson

		return p.personRepo.SavePerson(ctx, person)
	}

	return errGetPerson
}

func (p *personUsecase) GetPersonByDiscordID(ctx context.Context, discordID string, person *domain.Person) error {
	return p.personRepo.GetPersonByDiscordID(ctx, discordID, person)
}

func (p *personUsecase) GetExpiredProfiles(ctx context.Context, limit uint64) ([]domain.Person, error) {
	return p.personRepo.GetExpiredProfiles(ctx, limit)
}

func (p *personUsecase) GetPersonMessageByID(ctx context.Context, personMessageID int64, msg *domain.PersonMessage) error {
	return p.personRepo.GetPersonMessageByID(ctx, personMessageID, msg)
}

func (p *personUsecase) GetSteamIdsAbove(ctx context.Context, privilege domain.Privilege) (steamid.Collection, error) {
	return p.personRepo.GetSteamIdsAbove(ctx, privilege)
}

func (p *personUsecase) GetPersonSettings(ctx context.Context, steamID steamid.SID64, settings *domain.PersonSettings) error {
	return p.personRepo.GetPersonSettings(ctx, steamID, settings)
}

func (p *personUsecase) SavePersonSettings(ctx context.Context, settings *domain.PersonSettings) error {
	return p.personRepo.SavePersonSettings(ctx, settings)
}
