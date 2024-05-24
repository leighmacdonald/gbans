package blocklist

import (
	"context"
	"errors"
	"fmt"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type blocklistUsecase struct {
	blocklistRepo   domain.BlocklistRepository
	banUsecase      domain.BanSteamUsecase
	banGroupUsecase domain.BanGroupUsecase
	cidrRx          *regexp.Regexp
}

func NewBlocklistUsecase(br domain.BlocklistRepository, banUsecase domain.BanSteamUsecase, banGroupUsecase domain.BanGroupUsecase) domain.BlocklistUsecase {
	return &blocklistUsecase{
		blocklistRepo:   br,
		banUsecase:      banUsecase,
		banGroupUsecase: banGroupUsecase,
		cidrRx:          regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(/(3[0-2]|2[0-9]|1[0-9]|[0-9]))?$`),
	}
}

func (b blocklistUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 12)

	update := func() {
		b.syncBlocklists(ctx)
	}

	update()

	for {
		select {
		case <-ticker.C:
			update()
		case <-ctx.Done():
			return
		}
	}
}

func (b blocklistUsecase) syncBlocklists(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if err := b.banGroupUsecase.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned group members", log.ErrAttr(err))
		}
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if err := b.banUsecase.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned friends", log.ErrAttr(err))
		}
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if err := b.UpdateCache(ctx); err != nil {
			slog.Error("failed to update banned friends", log.ErrAttr(err))
		}
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
		return errReq
	}

	client := util.NewHTTPClient()
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

	var blocks []*net.IPNet //nolint:prealloc

	for _, line := range strings.Split(string(bodyBytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if !b.cidrRx.MatchString(trimmed) {
			continue
		}

		_, cidrBlock, errBlock := net.ParseCIDR(trimmed)
		if errBlock != nil {
			continue
		}

		blocks = append(blocks, cidrBlock)
	}

	if err := b.blocklistRepo.InsertCache(ctx, list, blocks); err != nil {
		return err
	}

	return nil
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
