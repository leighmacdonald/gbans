package ban

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type banNet struct {
	repository domain.BanNetRepository
	persons    domain.PersonUsecase
	config     domain.ConfigUsecase
	discord    domain.DiscordUsecase
	state      domain.StateUsecase
}

func NewBanNetUsecase(repository domain.BanNetRepository, personUsecase domain.PersonUsecase,
	configUsecase domain.ConfigUsecase, discordUsecase domain.DiscordUsecase, stateUsecase domain.StateUsecase,
) domain.BanNetUsecase {
	return &banNet{
		repository: repository, persons: personUsecase, config: configUsecase,
		discord: discordUsecase, state: stateUsecase,
	}
}

// Ban adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s *banNet) Ban(ctx context.Context, banNet *domain.BanCIDR) error {
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
		return errors.Join(errCIDR, domain.ErrNetworkInvalidIP)
	}

	if errSaveBanNet := s.repository.Save(ctx, banNet); errSaveBanNet != nil {
		return errors.Join(errSaveBanNet, domain.ErrSaveBan)
	}

	author, errAuthor := s.persons.GetOrCreatePersonBySteamID(ctx, banNet.SourceID)
	if errAuthor != nil {
		return errors.Join(errAuthor, domain.ErrFetchSource)
	}

	target, errTarget := s.persons.GetOrCreatePersonBySteamID(ctx, banNet.TargetID)
	if errTarget != nil {
		return errors.Join(errTarget, domain.ErrFetchTarget)
	}

	conf := s.config.Config()

	s.discord.SendPayload(domain.ChannelBanLog, discord.BanCIDRResponse(realCIDR, author, conf.ExtURL(author), target, banNet))

	go func(_ *net.IPNet, reason domain.Reason) {
		foundPlayers := s.state.Find("", steamid.SteamID{}, nil, realCIDR)

		if len(foundPlayers) == 0 {
			return
		}

		for _, player := range foundPlayers {
			if errKick := s.state.Kick(ctx, player.Player.SID, reason); errKick != nil {
				slog.Error("Failed to kick player", log.ErrAttr(errKick))
			}
		}
	}(realCIDR, banNet.Reason)

	return nil
}

func (s *banNet) GetByAddress(ctx context.Context, ipAddr netip.Addr) ([]domain.BanCIDR, error) {
	return s.repository.GetByAddress(ctx, ipAddr)
}

func (s *banNet) GetByID(ctx context.Context, netID int64, banNet *domain.BanCIDR) error {
	return s.repository.GetByID(ctx, netID, banNet)
}

func (s *banNet) Get(ctx context.Context, filter domain.CIDRBansQueryFilter) ([]domain.BannedCIDRPerson, error) {
	return s.repository.Get(ctx, filter)
}

func (s *banNet) Save(ctx context.Context, banNet *domain.BanCIDR) error {
	return s.repository.Save(ctx, banNet)
}

func (s *banNet) Delete(ctx context.Context, banNet *domain.BanCIDR) error {
	return s.repository.Delete(ctx, banNet)
}

func (s *banNet) Expired(ctx context.Context) ([]domain.BanCIDR, error) {
	return s.repository.Expired(ctx)
}
