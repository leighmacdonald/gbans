package ban

import (
	"context"
	"errors"
	"net"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"go.uber.org/zap"
)

type banNetUsecase struct {
	banRepo        domain.BanNetRepository
	personUsecase  domain.PersonUsecase
	configUsecase  domain.ConfigUsecase
	discordUsecase domain.DiscordUsecase
	stateUsecase   domain.StateUsecase
	log            *zap.Logger
}

func NewBanNetUsecase(logger *zap.Logger, repository domain.BanNetRepository, personUsecase domain.PersonUsecase,
	configUsecase domain.ConfigUsecase, discordUsecase domain.DiscordUsecase, stateUsecase domain.StateUsecase,
) domain.BanNetUsecase {
	return &banNetUsecase{
		log:     logger.Named("ban_net"),
		banRepo: repository, personUsecase: personUsecase, configUsecase: configUsecase,
		discordUsecase: discordUsecase, stateUsecase: stateUsecase,
	}
}

// BanCIDR adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s *banNetUsecase) Ban(ctx context.Context, banNet *domain.BanCIDR) error {
	// TODO
	// _, err2 := db.GetBanNetByAddress(ctx, net.ParseIP(cidrStr))
	// if err2 != nil && err2 != db.ErrNoResult {
	//	return "", errCommandFailed
	// }
	// if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	// }
	if banNet.CIDR == "" {
		return domain.ErrCIDRMissing
	}

	_, realCIDR, errCIDR := net.ParseCIDR(banNet.CIDR)
	if errCIDR != nil {
		return errors.Join(errCIDR, domain.ErrInvalidIP)
	}

	if errSaveBanNet := s.banRepo.Save(ctx, banNet); errSaveBanNet != nil {
		return errors.Join(errSaveBanNet, domain.ErrSaveBan)
	}

	var author domain.Person
	if err := s.personUsecase.GetOrCreatePersonBySteamID(ctx, banNet.SourceID, &author); err != nil {
		return errors.Join(err, domain.ErrFetchSource)
	}

	var target domain.Person
	if err := s.personUsecase.GetOrCreatePersonBySteamID(ctx, banNet.TargetID, &target); err != nil {
		return errors.Join(err, domain.ErrFetchTarget)
	}

	conf := s.configUsecase.Config()

	s.discordUsecase.SendPayload(domain.ChannelModLog, discord.BanCIDRResponse(realCIDR, author, conf.ExtURL(author), target, banNet))

	go func(_ *net.IPNet, reason domain.Reason) {
		foundPlayers := s.stateUsecase.Find("", "", nil, realCIDR)

		if len(foundPlayers) == 0 {
			return
		}

		for _, player := range foundPlayers {
			if errKick := s.stateUsecase.Kick(ctx, player.Player.SID, reason); errKick != nil {
				s.log.Error("Failed to kick player", zap.Error(errKick))
			}
		}
	}(realCIDR, banNet.Reason)

	return nil
}

func (s *banNetUsecase) GetByAddress(ctx context.Context, ipAddr net.IP) ([]domain.BanCIDR, error) {
	return s.banRepo.GetByAddress(ctx, ipAddr)
}

func (s *banNetUsecase) GetByID(ctx context.Context, netID int64, banNet *domain.BanCIDR) error {
	return s.banRepo.GetByID(ctx, netID, banNet)
}

func (s *banNetUsecase) Get(ctx context.Context, filter domain.CIDRBansQueryFilter) ([]domain.BannedCIDRPerson, int64, error) {
	return s.banRepo.Get(ctx, filter)
}

func (s *banNetUsecase) Save(ctx context.Context, banNet *domain.BanCIDR) error {
	return s.banRepo.Save(ctx, banNet)
}

func (s *banNetUsecase) Delete(ctx context.Context, banNet *domain.BanCIDR) error {
	return s.banRepo.Delete(ctx, banNet)
}

func (s *banNetUsecase) Expired(ctx context.Context) ([]domain.BanCIDR, error) {
	return s.banRepo.Expired(ctx)
}
