package ban

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type banNetUsecase struct {
	banRepo        domain.BanNetRepository
	personUsecase  domain.PersonUsecase
	configUsecase  domain.ConfigUsecase
	discordUsecase domain.DiscordUsecase
	stateUsecase   domain.StateUsecase
}

func NewBanNetUsecase(repository domain.BanNetRepository, personUsecase domain.PersonUsecase,
	configUsecase domain.ConfigUsecase, discordUsecase domain.DiscordUsecase, stateUsecase domain.StateUsecase,
) domain.BanNetUsecase {
	return &banNetUsecase{
		banRepo: repository, personUsecase: personUsecase, configUsecase: configUsecase,
		discordUsecase: discordUsecase, stateUsecase: stateUsecase,
	}
}

// Ban adds a new network to the banned network list. It will accept any Valid CIDR format.
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

	author, errAuthor := s.personUsecase.GetOrCreatePersonBySteamID(ctx, banNet.SourceID)
	if errAuthor != nil {
		return errors.Join(errAuthor, domain.ErrFetchSource)
	}

	target, errTarget := s.personUsecase.GetOrCreatePersonBySteamID(ctx, banNet.TargetID)
	if errTarget != nil {
		return errors.Join(errTarget, domain.ErrFetchTarget)
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
				slog.Error("Failed to kick player", log.ErrAttr(errKick))
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
