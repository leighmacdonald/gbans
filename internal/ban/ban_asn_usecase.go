package ban

import (
	"context"
	"errors"
	"strconv"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/datetime"
)

type banASN struct {
	repository domain.BanASNRepository
	discord    domain.DiscordUsecase
	networks   domain.NetworkUsecase
	config     domain.ConfigUsecase
	person     domain.PersonUsecase
}

func NewBanASNUsecase(repository domain.BanASNRepository, discord domain.DiscordUsecase,
	network domain.NetworkUsecase, config domain.ConfigUsecase, person domain.PersonUsecase,
) domain.BanASNUsecase {
	return banASN{
		repository: repository,
		discord:    discord,
		networks:   network,
		config:     config,
		person:     person,
	}
}

func (s banASN) Expired(ctx context.Context) ([]domain.BanASN, error) {
	return s.repository.Expired(ctx)
}

func (s banASN) Ban(ctx context.Context, req domain.RequestBanASNCreate) (domain.BannedASNPerson, error) {
	var ban domain.BannedASNPerson
	var existing domain.BanASN
	if errGetExistingBan := s.repository.GetByASN(ctx, req.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, domain.ErrNoResult) {
			return ban, errors.Join(errGetExistingBan, domain.ErrFailedFetchBan)
		}
	}

	sourceID, valid := req.SourceSteamID(ctx)
	if !valid {
		return ban, domain.ErrInvalidAuthorSID
	}

	targetID, targetValid := req.TargetSteamID(ctx)
	if !targetValid {
		return ban, domain.ErrInvalidTargetSID
	}

	author, errAuthor := s.person.GetOrCreatePersonBySteamID(ctx, sourceID)
	if errAuthor != nil {
		return ban, errors.Join(errAuthor, domain.ErrGetPerson)
	}

	target, errTarget := s.person.GetOrCreatePersonBySteamID(ctx, targetID)
	if errTarget != nil {
		return ban, errors.Join(errTarget, domain.ErrGetPerson)
	}

	duration, errDuration := datetime.CalcDuration(req.Duration, req.ValidUntil)
	if errDuration != nil {
		return ban, errDuration
	}

	var newBan domain.BanASN
	if err := domain.NewBanASN(author.SteamID, target.SteamID, duration, req.Reason, req.ReasonText,
		req.Note, domain.System, req.ASNum, domain.Banned, &newBan); err != nil {
		return ban, err
	}

	bannedPerson, errSave := s.repository.Save(ctx, &newBan)
	if errSave != nil {
		return ban, errors.Join(errSave, domain.ErrSaveBan)
	}

	s.discord.SendPayload(domain.ChannelBanLog, discord.BanASNMessage(bannedPerson, s.config.Config()))

	return bannedPerson, nil
}

func (s banASN) Unban(ctx context.Context, asnNum string, reasonText string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errors.Join(errConv, domain.ErrParseASN)
	}

	var ban domain.BanASN
	if errGetBanASN := s.repository.GetByASN(ctx, asNum, &ban); errGetBanASN != nil {
		return false, errors.Join(errGetBanASN, domain.ErrFetchASNBan)
	}

	if errDrop := s.Delete(ctx, ban.BanASNId, domain.RequestUnban{UnbanReasonText: reasonText}); errDrop != nil {
		return false, errors.Join(errDrop, domain.ErrDropASNBan)
	}

	s.discord.SendPayload(domain.ChannelBanLog, discord.UnbanASNMessage(asNum))

	return true, nil
}

func (s banASN) GetByID(ctx context.Context, banID int64) (domain.BannedASNPerson, error) {
	return s.repository.GetByID(ctx, banID)
}

func (s banASN) GetByASN(ctx context.Context, asNum int64, banASN *domain.BanASN) error {
	return s.repository.GetByASN(ctx, asNum, banASN)
}

func (s banASN) Get(ctx context.Context, filter domain.ASNBansQueryFilter) ([]domain.BannedASNPerson, error) {
	return s.repository.Get(ctx, filter)
}

func (s banASN) Update(ctx context.Context, asnID int64, req domain.RequestBanASNUpdate) (domain.BannedASNPerson, error) {
	ban, errBan := s.GetByID(ctx, asnID)
	if errBan != nil {
		return ban, errBan
	}

	if ban.Reason == domain.Custom && req.ReasonText == "" {
		return ban, domain.ErrInvalidParameter
	}

	targetID, targetIDOK := req.TargetSteamID(ctx)
	if !targetIDOK {
		return ban, domain.ErrInvalidParameter
	}

	ban.Note = req.Note
	ban.ASNum = req.ASNum
	ban.ValidUntil = req.ValidUntil
	ban.TargetID = targetID
	ban.Reason = req.Reason
	ban.ReasonText = req.ReasonText

	return s.repository.Save(ctx, &ban.BanASN)
}

func (s banASN) Delete(ctx context.Context, asnID int64, req domain.RequestUnban) error {
	asn, errFetch := s.GetByID(ctx, asnID)
	if errFetch != nil {
		return errFetch
	}

	asn.UnbanReasonText = req.UnbanReasonText
	asn.Deleted = true

	if _, err := s.repository.Save(ctx, &asn.BanASN); err != nil {
		return err
	}

	return nil
}
