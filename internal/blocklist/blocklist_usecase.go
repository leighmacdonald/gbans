package blocklist

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
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

func (b blocklistUsecase) SaveCIDRBlockSources(ctx context.Context, block *domain.CIDRBlockSource) error {
	return b.blocklistRepo.SaveCIDRBlockSources(ctx, block)
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

func (b blocklistUsecase) SaveCIDRBlockWhitelist(ctx context.Context, whitelist *domain.CIDRBlockWhitelist) error {
	return b.blocklistRepo.SaveCIDRBlockWhitelist(ctx, whitelist)
}

func (b blocklistUsecase) DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error {
	return b.blocklistRepo.DeleteCIDRBlockWhitelist(ctx, whitelistID)
}
