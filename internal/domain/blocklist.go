package domain

import (
	"context"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type BlocklistUsecase interface {
	GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error
	CreateCIDRBlockSources(ctx context.Context, name string, url string, enabled bool) (CIDRBlockSource, error)
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]WhitelistIP, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *WhitelistIP) error
	CreateCIDRBlockWhitelist(ctx context.Context, address string) (WhitelistIP, error)
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
	UpdateCIDRBlockSource(ctx context.Context, sourceID int, name string, url string, enabled bool) (CIDRBlockSource, error)
	UpdateCIDRBlockWhitelist(ctx context.Context, whitelistID int, address string) (WhitelistIP, error)

	CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (WhitelistSteam, error)
	GetSteamBlockWhitelists(ctx context.Context) ([]WhitelistSteam, error)
	DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error
}

type BlocklistRepository interface {
	GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error
	SaveCIDRBlockSources(ctx context.Context, block *CIDRBlockSource) error
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]WhitelistIP, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *WhitelistIP) error
	SaveCIDRBlockWhitelist(ctx context.Context, whitelist *WhitelistIP) error
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
	TruncateCachedEntries(ctx context.Context) error
	CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (WhitelistSteam, error)
	GetSteamBlockWhitelists(ctx context.Context) ([]WhitelistSteam, error)
	DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error
}
