package anticheat

import (
	"context"
	"io"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type antiCheatUsecase struct {
	parser logparse.StacParser
	repo   domain.AntiCheatRepository
	person domain.PersonUsecase
}

func NewAntiCheatUsecase(repo domain.AntiCheatRepository, person domain.PersonUsecase) domain.AntiCheatUsecase {
	return &antiCheatUsecase{
		parser: logparse.NewStacParser(),
		repo:   repo,
		person: person,
	}
}

func (a antiCheatUsecase) DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error) {
	if !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	return a.repo.DetectionsBySteamID(ctx, steamID)
}

func (a antiCheatUsecase) DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error) {
	return a.repo.DetectionsByType(ctx, detectionType)
}

func (a antiCheatUsecase) Import(ctx context.Context, fileName string, reader io.ReadCloser, serverID int) error {
	entries, errEntries := a.parser.Parse(fileName, reader)
	if errEntries != nil {
		return errEntries
	}

	if len(entries) == 0 {
		return nil
	}

	for i := range entries {
		entries[i].ServerID = serverID
	}

	for _, entry := range entries {
		player, err := a.person.GetOrCreatePersonBySteamID(ctx, nil, entry.SteamID)
		if err != nil {
			return err
		}
		if player.PersonaName == "" && entry.Name != "" {
			player.PersonaName = entry.Name
			if errSave := a.person.SavePerson(ctx, nil, &player); errSave != nil {
				return errSave
			}
		}
	}

	return a.repo.SaveEntries(ctx, entries)
}

func (a antiCheatUsecase) SyncDemoIDs(ctx context.Context, limit uint64) error {
	if limit == 0 {
		limit = 100
	}

	return a.repo.SyncDemoIDs(ctx, limit)
}

func (a antiCheatUsecase) Query(ctx context.Context, query domain.AnticheatQuery) ([]domain.AnticheatEntry, error) {
	if query.SteamID != "" {
		sid := steamid.New(query.SteamID)
		if !sid.Valid() {
			return nil, domain.ErrInvalidSID
		}
	}

	return a.repo.Query(ctx, query)
}
