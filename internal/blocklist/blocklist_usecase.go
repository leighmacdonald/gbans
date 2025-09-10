package blocklist

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

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type blocklistUsecase struct {
	repository domain.BlocklistRepository
	bans       ban.BanUsecase
	cidrRx     *regexp.Regexp
}

func NewBlocklistUsecase(br domain.BlocklistRepository, banUsecase ban.BanUsecase) domain.BlocklistUsecase {
	return &blocklistUsecase{
		repository: br,
		bans:       banUsecase,
		cidrRx:     regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(/(3[0-2]|2[0-9]|1[0-9]|[0-9]))?$`),
	}
}

func (b blocklistUsecase) Sync(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if err := b.bans.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned group members", log.ErrAttr(err))

			return
		}

		slog.Debug("Banned group members updated")
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if err := b.bans.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned friends", log.ErrAttr(err))

			return
		}

		slog.Debug("Banned friends updated")
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if err := b.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned friends", log.ErrAttr(err))

			return
		}

		slog.Debug("Banned CIDR ranges updated")
	}()

	waitGroup.Wait()
}

func (b blocklistUsecase) UpdateCache(ctx context.Context) error {
	lists, errLists := b.GetCIDRBlockSources(ctx)
	if errLists != nil {
		return errLists
	}

	for _, list := range lists {
		if err := b.updateSource(ctx, list); err != nil {
			slog.Error("Failed to update cidr block source", log.ErrAttr(err))
		}
	}

	return nil
}

func (b blocklistUsecase) updateSource(ctx context.Context, list domain.CIDRBlockSource) error {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if errReq != nil {
		return errors.Join(errReq, domain.ErrRequestCreate)
	}

	client := httphelper.NewHTTPClient()

	resp, errResp := client.Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", domain.ErrRequestInvalidCode, resp.StatusCode)
	}

	bodyBytes, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return errors.Join(errRead, domain.ErrResponseBody)
	}

	var blocks []netip.Prefix //nolint:prealloc

	for _, line := range strings.Split(string(bodyBytes), "\n") {
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

	blocks = append(blocks, netip.MustParsePrefix("192.168.0.0/24"))

	if err := b.repository.TruncateCachedEntries(ctx); err != nil {
		return err
	}

	if err := b.repository.InsertCache(ctx, list, blocks); err != nil {
		return err
	}

	return nil
}

func (b blocklistUsecase) CreateSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) (domain.WhitelistSteam, error) {
	whitelist, err := b.repository.CreateSteamBlockWhitelists(ctx, steamID)
	if err != nil {
		return domain.WhitelistSteam{}, err
	}

	slog.Info("Created steam block whitelist", slog.String("steam_id", steamID.String()))

	return whitelist, nil
}

func (b blocklistUsecase) GetSteamBlockWhitelists(ctx context.Context) ([]domain.WhitelistSteam, error) {
	return b.repository.GetSteamBlockWhitelists(ctx)
}

func (b blocklistUsecase) DeleteSteamBlockWhitelists(ctx context.Context, steamID steamid.SteamID) error {
	if err := b.repository.DeleteSteamBlockWhitelists(ctx, steamID); err != nil {
		return err
	}

	slog.Info("Deleted steam whitelist", slog.String("steam_id", steamID.String()))

	return nil
}

func (b blocklistUsecase) GetCIDRBlockSources(ctx context.Context) ([]domain.CIDRBlockSource, error) {
	return b.repository.GetCIDRBlockSources(ctx)
}

func (b blocklistUsecase) GetCIDRBlockSource(ctx context.Context, sourceID int, block *domain.CIDRBlockSource) error {
	return b.repository.GetCIDRBlockSource(ctx, sourceID, block)
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

	if err := b.repository.SaveCIDRBlockSources(ctx, &blockList); err != nil {
		return domain.CIDRBlockSource{}, domain.ErrInternal
	}

	slog.Info("Created blocklist", slog.String("name", blockList.Name))

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

	blockSource.Enabled = enabled
	blockSource.Name = name
	blockSource.URL = url

	if err := b.repository.SaveCIDRBlockSources(ctx, &blockSource); err != nil {
		return blockSource, err
	}

	slog.Info("Updated blocklist", slog.String("name", blockSource.Name))

	return blockSource, nil
}

func (b blocklistUsecase) DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error {
	if err := b.repository.DeleteCIDRBlockSources(ctx, blockSourceID); err != nil {
		return err
	}

	slog.Info("Deleted blocklist", slog.Int("cidr_block_source_id", blockSourceID))

	return nil
}

func (b blocklistUsecase) GetCIDRBlockWhitelists(ctx context.Context) ([]domain.WhitelistIP, error) {
	return b.repository.GetCIDRBlockWhitelists(ctx)
}

func (b blocklistUsecase) GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *domain.WhitelistIP) error {
	return b.repository.GetCIDRBlockWhitelist(ctx, whitelistID, whitelist)
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

	if errSave := b.repository.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
		return domain.WhitelistIP{}, errSave
	}

	slog.Info("Created ip whitelist", slog.String("addr", address))

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

	if errSave := b.repository.SaveCIDRBlockWhitelist(ctx, &whitelist); errSave != nil {
		return domain.WhitelistIP{}, errSave
	}

	slog.Info("Updated ip whitelist", slog.String("addr", address), slog.Int("whitelist_id", whitelistID))

	return whitelist, nil
}

func (b blocklistUsecase) DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error {
	if err := b.repository.DeleteCIDRBlockWhitelist(ctx, whitelistID); err != nil {
		return err
	}

	slog.Info("Blocklist deleted", slog.Int("cidr_block_source_id", whitelistID))

	return nil
}
