package domain

import "context"

type BlocklistUsecase interface {
	GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error
	CreateCIDRBlockSources(ctx context.Context, name string, url string, enabled bool) (CIDRBlockSource, error)
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]CIDRBlockWhitelist, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *CIDRBlockWhitelist) error
	CreateCIDRBlockWhitelist(ctx context.Context, address string) (CIDRBlockWhitelist, error)
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
	UpdateCIDRBlockSource(ctx context.Context, sourceID int, name string, url string, enabled bool) (CIDRBlockSource, error)
	UpdateCIDRBlockWhitelist(ctx context.Context, whitelistID int, address string) (CIDRBlockWhitelist, error)
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
