package person

import (
	"context"
	"errors"
	"net"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type personUsecase struct {
	personRepo domain.PersonRepository
}

func NewPersonUsecase(pr domain.PersonRepository) domain.PersonUsecase {
	return &personUsecase{
		personRepo: pr,
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

	// env.Log().Info("Discord steamid set", zap.Int64("sid64", sid64.Int64()), zap.String("discordId", discordID))

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
	return p.GetPersonByDiscordID(ctx, discordID, person)
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
