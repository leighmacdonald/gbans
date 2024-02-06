package blocklist

import (
	"context"
	"errors"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/network"
	"net"
	"net/url"
	"strings"
)

type blocklistUsecase struct {
	blocklistRepo domain.BlocklistRepository
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

func (b blocklistUsecase) GetCIDRBlockWhitelists(ctx context.Context) ([]domain.CIDRBlockWhitelist, error) {
	return b.blocklistRepo.GetCIDRBlockWhitelists(ctx)
}

func (b blocklistUsecase) GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *domain.CIDRBlockWhitelist) error {
	return b.blocklistRepo.GetCIDRBlockWhitelist(ctx, whitelistID, whitelist)
}

func (b blocklistUsecase) CreateCIDRBlockWhitelist(ctx context.Context, address string) (domain.CIDRBlockWhitelist, error) {
	if !strings.Contains(address, "/") {
		address += "/32"
	}

	_, cidr, errParse := net.ParseCIDR(address)
	if errParse != nil {
		return domain.CIDRBlockWhitelist{}, domain.ErrInvalidCIDR
	}

	whitelist := domain.CIDRBlockWhitelist{
		Address:     cidr,
		TimeStamped: domain.NewTimeStamped(),
	}

	if errSave := b.blocklistRepo.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
		return domain.CIDRBlockWhitelist{}, errSave
	}

	return whitelist, nil
}
func (b blocklistUsecase) UpdateCIDRBlockWhitelist(ctx context.Context, whitelistID int, address string) (domain.CIDRBlockWhitelist, error) {
	_, cidr, errParse := net.ParseCIDR(address)
	if errParse != nil {
		return domain.CIDRBlockWhitelist{}, domain.ErrInvalidCIDR
	}

	var whitelist domain.CIDRBlockWhitelist
	if errGet := b.GetCIDRBlockWhitelist(ctx, whitelistID, &whitelist); errGet != nil {
		return domain.CIDRBlockWhitelist{}, errGet
	}

	whitelist.Address = cidr

	return whitelist, nil
}

func (b blocklistUsecase) DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error {
	return b.blocklistRepo.DeleteCIDRBlockWhitelist(ctx, whitelistID)
}
