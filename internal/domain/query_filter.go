package domain

import (
	"context"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type SteamIDField struct {
	SteamIDValue string `json:"steam_id"  url:"steam_id"` //nolint:tagliatelle
}

func (f SteamIDField) SteamID(ctx context.Context) (steamid.SteamID, bool) {
	if f.SteamIDValue == "" {
		return steamid.SteamID{}, false
	}

	sid, err := steamid.Resolve(ctx, f.SteamIDValue)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}

type SourceIDField struct {
	SourceID string `json:"source_id"  url:"source_id"`
}

func (f SourceIDField) SourceSteamID(ctx context.Context) (steamid.SteamID, bool) {
	if f.SourceID == "" {
		return steamid.SteamID{}, false
	}

	sid, err := steamid.Resolve(ctx, f.SourceID)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}

type TargetIDField struct {
	TargetID string `json:"target_id" url:"target_id"`
}

func (f TargetIDField) TargetSteamID(ctx context.Context) (steamid.SteamID, bool) {
	if f.TargetID == "" {
		return steamid.SteamID{}, false
	}

	sid, err := steamid.Resolve(ctx, f.TargetID)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}
