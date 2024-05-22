package blocklist

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"strings"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type blocklistUsecase struct {
	blocklistRepo domain.BlocklistRepository
}

func (b blocklistUsecase) SyncBlocklists(ctx context.Context) error {
	lists, errLists := b.GetCIDRBlockSources(ctx)
	if errLists != nil {
		return errLists
	}

	blocker := network.NewBlocker()

	for _, list := range lists {
		if !list.Enabled {
			continue
		}

		count, errAdd := blocker.AddRemoteSource(ctx, list.Name, list.URL)
		if errAdd != nil {
			slog.Error("Failed to load source data", slog.String("name", list.Name), slog.String("url", list.URL))
			continue
		}
	}

	if err := b.blocklistRepo.TruncateCachedEntries(ctx); err != nil {
		return err
	}

	for k, v := range blocker.Blocks {

	}
}

func (b blocklistUsecase) CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (domain.WhitelistSteam, error) {
	return b.blocklistRepo.CreateSteamBlockWhitelists(ctx, steamID)
}

func (b blocklistUsecase) GetSteamBlockWhitelists(ctx context.Context) ([]domain.WhitelistSteam, error) {
	return b.blocklistRepo.GetSteamBlockWhitelists(ctx)
}

func (b blocklistUsecase) DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error {
	return b.blocklistRepo.DeleteSteamBlockWhitelists(ctx, steamID)
}

func NewBlocklistUsecase(br domain.BlocklistRepository) domain.BlocklistUsecase {
	return &blocklistUsecase{blocklistRepo: br}
}

func (b blocklistUsecase) GetCIDRBlockSources(ctx context.Context) ([]domain.CIDRBlockSource, error) {
	return b.blocklistRepo.GetCIDRBlockSources(ctx)
}

func (b blocklistUsecase) GetCIDRBlockSource(ctx context.Context, sourceID int, block *domain.CIDRBlockSource) error {
	return b.blocklistRepo.GetCIDRBlockSource(ctx, sourceID, block)
}

func (b blocklistUsecase) CreateCIDRBlockSources(ctx context.Context, name string, listURL string, enabled bool) (domain.CIDRBlockSource, error) {
	if name == "" {
		return domain.CIDRBlockSource{}, domain.ErrBadRequest
	}

	parsedURL, errURL := url.Parse(listURL)
	if errURL != nil {
		return domain.CIDRBlockSource{}, domain.ErrBadRequest
	}

	blockList := domain.CIDRBlockSource{
		Name:        name,
		URL:         parsedURL.String(),
		Enabled:     enabled,
		TimeStamped: domain.NewTimeStamped(),
	}

	if err := b.blocklistRepo.SaveCIDRBlockSources(ctx, &blockList); err != nil {
		return domain.CIDRBlockSource{}, domain.ErrInternal
	}

	return blockList, nil
}

func (b blocklistUsecase) UpdateCIDRBlockSource(ctx context.Context, sourceID int, name string, url string, enabled bool) (domain.CIDRBlockSource, error) {
	var blockSource domain.CIDRBlockSource

	if errSource := b.GetCIDRBlockSource(ctx, sourceID, &blockSource); errSource != nil {
		if errors.Is(errSource, domain.ErrNoResult) {
			return blockSource, domain.ErrNotFound
		}

		return blockSource, domain.ErrBadRequest
	}

	testBlocker := network.NewBlocker()
	if count, errTest := testBlocker.AddRemoteSource(ctx, name, url); errTest != nil || count == 0 {
		return blockSource, domain.ErrValidateURL
	}

	blockSource.Enabled = enabled
	blockSource.Name = name
	blockSource.URL = url

	if err := b.blocklistRepo.SaveCIDRBlockSources(ctx, &blockSource); err != nil {
		return blockSource, err
	}

	return blockSource, nil
}

func (b blocklistUsecase) DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error {
	return b.blocklistRepo.DeleteCIDRBlockSources(ctx, blockSourceID)
}

func (b blocklistUsecase) GetCIDRBlockWhitelists(ctx context.Context) ([]domain.WhitelistIP, error) {
	return b.blocklistRepo.GetCIDRBlockWhitelists(ctx)
}

func (b blocklistUsecase) GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *domain.WhitelistIP) error {
	return b.blocklistRepo.GetCIDRBlockWhitelist(ctx, whitelistID, whitelist)
}

func (b blocklistUsecase) CreateCIDRBlockWhitelist(ctx context.Context, address string) (domain.WhitelistIP, error) {
	if !strings.Contains(address, "/") {
		address += "/32"
	}

	_, cidr, errParse := net.ParseCIDR(address)
	if errParse != nil {
		return domain.WhitelistIP{}, domain.ErrInvalidCIDR
	}

	whitelist := domain.WhitelistIP{
		Address:     cidr,
		TimeStamped: domain.NewTimeStamped(),
	}

	if errSave := b.blocklistRepo.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
		return domain.WhitelistIP{}, errSave
	}

	return whitelist, nil
}

func (b blocklistUsecase) UpdateCIDRBlockWhitelist(ctx context.Context, whitelistID int, address string) (domain.WhitelistIP, error) {
	_, cidr, errParse := net.ParseCIDR(address)
	if errParse != nil {
		return domain.WhitelistIP{}, domain.ErrInvalidCIDR
	}

	var whitelist domain.WhitelistIP
	if errGet := b.GetCIDRBlockWhitelist(ctx, whitelistID, &whitelist); errGet != nil {
		return domain.WhitelistIP{}, errGet
	}

	whitelist.Address = cidr

	if errSave := b.blocklistRepo.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
		return domain.WhitelistIP{}, errSave
	}

	return whitelist, nil
}

func (b blocklistUsecase) DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error {
	return b.blocklistRepo.DeleteCIDRBlockWhitelist(ctx, whitelistID)
}
