package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var ErrInvalidCIDR = errors.New("failed to parse CIDR address")

const maskSingleHost = "/32"

type CIDRBlockSource struct {
	CIDRBlockSourceID int       `json:"cidr_block_source_id"`
	Name              string    `json:"name"`
	URL               string    `json:"url"`
	Enabled           bool      `json:"enabled"`
	CreatedOn         time.Time `json:"created_on"`
	UpdatedOn         time.Time `json:"updated_on"`
}

type WhitelistIP struct {
	CIDRBlockWhitelistID int        `json:"cidr_block_whitelist_id"`
	Address              *net.IPNet `json:"address"`
	CreatedOn            time.Time  `json:"created_on"`
	UpdatedOn            time.Time  `json:"updated_on"`
}

type WhitelistSteam struct {
	httphelper.SteamIDField

	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	Personaname string    `json:"personaname"`
	AvatarHash  string    `json:"avatar_hash"`
}

type CacheUpdater interface {
	UpdateCache(ctx context.Context) error
}

type Blocklists struct {
	repository BlocklistRepository
	bans       CacheUpdater
	cidrRx     *regexp.Regexp
}

func NewBlocklists(br BlocklistRepository, banUsecase CacheUpdater) Blocklists {
	return Blocklists{
		repository: br,
		bans:       banUsecase,
		cidrRx:     regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(/(3[0-2]|2[0-9]|1[0-9]|[0-9]))?$`),
	}
}

func (b *Blocklists) Sync(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}

	waitGroup.Go(func() {
		if err := b.bans.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned group members", slog.String("error", err.Error()))

			return
		}

		slog.Debug("Banned group members updated")
	})

	waitGroup.Go(func() {
		if err := b.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned CIDR ranges", slog.String("error", err.Error()))

			return
		}

		slog.Debug("Banned CIDR ranges updated")
	})

	waitGroup.Wait()
}

func (b *Blocklists) UpdateCache(ctx context.Context) error {
	lists, errLists := b.GetCIDRBlockSources(ctx)
	if errLists != nil {
		return errLists
	}

	for _, list := range lists {
		if err := b.updateSource(ctx, list); err != nil {
			slog.Error("Failed to update cidr block source", slog.String("error", err.Error()))
		}
	}

	return nil
}

func (b *Blocklists) updateSource(ctx context.Context, list CIDRBlockSource) error {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if errReq != nil {
		return errors.Join(errReq, httphelper.ErrRequestCreate)
	}

	client := httphelper.NewClient()

	resp, errResp := client.Do(req)
	if errResp != nil {
		return errors.Join(errResp, httphelper.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", httphelper.ErrRequestInvalidCode, resp.StatusCode)
	}

	bodyBytes, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return errors.Join(errRead, httphelper.ErrResponseBody)
	}

	var blocks []netip.Prefix //nolint:prealloc

	for line := range strings.SplitSeq(string(bodyBytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if !b.cidrRx.MatchString(trimmed) {
			continue
		}

		prefix, errBlock := netip.ParsePrefix(trimmed)
		if errBlock != nil {
			continue
		}

		blocks = append(blocks, prefix)
	}

	if err := b.repository.TruncateCachedEntries(ctx); err != nil {
		return err
	}

	if err := b.repository.InsertCache(ctx, list, blocks); err != nil {
		return err
	}

	return nil
}

func (b *Blocklists) CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (WhitelistSteam, error) {
	whitelist, err := b.repository.CreateSteamBlockWhitelists(ctx, steamID)
	if err != nil {
		return WhitelistSteam{}, err
	}

	slog.Info("Created steam block whitelist", slog.String("steam_id", steamID.String()))

	return whitelist, nil
}

func (b *Blocklists) GetSteamBlockWhitelists(ctx context.Context) ([]WhitelistSteam, error) {
	return b.repository.GetSteamBlockWhitelists(ctx)
}

func (b *Blocklists) DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error {
	if err := b.repository.DeleteSteamBlockWhitelists(ctx, steamID); err != nil {
		return err
	}

	slog.Info("Deleted steam whitelist", slog.String("steam_id", steamID.String()))

	return nil
}

func (b *Blocklists) GetCIDRBlockSources(ctx context.Context) ([]CIDRBlockSource, error) {
	return b.repository.GetCIDRBlockSources(ctx)
}

func (b *Blocklists) GetCIDRBlockSource(ctx context.Context, sourceID int, block *CIDRBlockSource) error {
	return b.repository.GetCIDRBlockSource(ctx, sourceID, block)
}

func (b *Blocklists) CreateCIDRBlockSources(ctx context.Context, name string, listURL string, enabled bool) (CIDRBlockSource, error) {
	if name == "" {
		return CIDRBlockSource{}, httphelper.ErrBadRequest // TODO better error
	}

	parsedURL, errURL := url.Parse(listURL)
	if errURL != nil {
		return CIDRBlockSource{}, httphelper.ErrBadRequest
	}

	blockList := CIDRBlockSource{
		Name:      name,
		URL:       parsedURL.String(),
		Enabled:   enabled,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}

	if err := b.repository.SaveCIDRBlockSources(ctx, &blockList); err != nil {
		return CIDRBlockSource{}, httphelper.ErrInternal
	}

	slog.Info("Created blocklist", slog.String("name", blockList.Name))

	return blockList, nil
}

func (b *Blocklists) UpdateCIDRBlockSource(ctx context.Context, sourceID int, name string, url string, enabled bool) (CIDRBlockSource, error) {
	var blockSource CIDRBlockSource

	if errSource := b.GetCIDRBlockSource(ctx, sourceID, &blockSource); errSource != nil {
		if errors.Is(errSource, database.ErrNoResult) {
			return blockSource, httphelper.ErrNotFound
		}

		return blockSource, httphelper.ErrBadRequest // TODO better errro
	}

	blockSource.Enabled = enabled
	blockSource.Name = name
	blockSource.URL = url

	if err := b.repository.SaveCIDRBlockSources(ctx, &blockSource); err != nil {
		return blockSource, err
	}

	slog.Info("Updated blocklist", slog.String("name", blockSource.Name))

	return blockSource, nil
}

func (b *Blocklists) DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error {
	if err := b.repository.DeleteCIDRBlockSources(ctx, blockSourceID); err != nil {
		return err
	}

	slog.Info("Deleted blocklist", slog.Int("cidr_block_source_id", blockSourceID))

	return nil
}

func (b *Blocklists) GetCIDRBlockWhitelists(ctx context.Context) ([]WhitelistIP, error) {
	return b.repository.GetCIDRBlockWhitelists(ctx)
}

func (b *Blocklists) GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *WhitelistIP) error {
	return b.repository.GetCIDRBlockWhitelist(ctx, whitelistID, whitelist)
}

func (b *Blocklists) CreateCIDRBlockWhitelist(ctx context.Context, address string) (WhitelistIP, error) {
	if !strings.Contains(address, "/") {
		address += maskSingleHost
	}

	_, cidr, errParse := net.ParseCIDR(address)
	if errParse != nil {
		return WhitelistIP{}, ErrInvalidCIDR
	}

	whitelist := WhitelistIP{
		Address:   cidr,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}

	if errSave := b.repository.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
		return WhitelistIP{}, errSave
	}

	slog.Info("Created ip whitelist", slog.String("addr", address))

	return whitelist, nil
}

func (b *Blocklists) UpdateCIDRBlockWhitelist(ctx context.Context, whitelistID int, address string) (WhitelistIP, error) {
	_, cidr, errParse := net.ParseCIDR(address)
	if errParse != nil {
		return WhitelistIP{}, ErrInvalidCIDR
	}

	var whitelist WhitelistIP
	if errGet := b.GetCIDRBlockWhitelist(ctx, whitelistID, &whitelist); errGet != nil {
		return WhitelistIP{}, errGet
	}

	whitelist.Address = cidr

	if errSave := b.repository.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
		return WhitelistIP{}, errSave
	}

	slog.Info("Updated ip whitelist", slog.String("addr", address), slog.Int("whitelist_id", whitelistID))

	return whitelist, nil
}

func (b *Blocklists) DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error {
	if err := b.repository.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
		return err
	}

	slog.Info("Blocklist deleted", slog.Int("cidr_block_source_id", whitelistID))

	return nil
}
