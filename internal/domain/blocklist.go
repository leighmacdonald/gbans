package domain

import "context"

type BlocklistUsecase interface {
	GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error
	SaveCIDRBlockSources(ctx context.Context, block *CIDRBlockSource) error
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]CIDRBlockWhitelist, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *CIDRBlockWhitelist) error
	SaveCIDRBlockWhitelist(ctx context.Context, whitelist *CIDRBlockWhitelist) error
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
}

type BlocklistRepository interface {
	GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error
	SaveCIDRBlockSources(ctx context.Context, block *CIDRBlockSource) error
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]CIDRBlockWhitelist, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *CIDRBlockWhitelist) error
	SaveCIDRBlockWhitelist(ctx context.Context, whitelist *CIDRBlockWhitelist) error
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
}
