package tests

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const DefaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

type FakeTFAPI struct{}

func (f FakeTFAPI) SteamGroup(_ context.Context, _ steamid.SteamID) (thirdparty.SteamGroup, error) {
	return thirdparty.SteamGroup{}, nil
}

func (f FakeTFAPI) LogsTFSummary(_ context.Context, _ steamid.SteamID) (thirdparty.LogsTFPlayerSummary, error) {
	return thirdparty.LogsTFPlayerSummary{}, nil
}

func (f FakeTFAPI) SteamBans(_ context.Context, steamIDs steamid.Collection) ([]thirdparty.SteamBan, error) {
	resp := make([]thirdparty.SteamBan, len(steamIDs))

	for index, steamID := range steamIDs {
		resp[index] = thirdparty.SteamBan{
			CommunityBanned:  false,
			DaysSinceLastBan: 0,
			EconomyBan:       "none",
			NumberOfGameBans: 0,
			NumberOfVacBans:  0,
			SteamId:          steamID.String(),
			VacBanned:        false,
		}
	}

	return resp, nil
}

func (f FakeTFAPI) Summaries(_ context.Context, steamIDs steamid.Collection) ([]thirdparty.PlayerSummaryResponse, error) {
	resp := make([]thirdparty.PlayerSummaryResponse, len(steamIDs))

	for index, steamID := range steamIDs {
		resp[index] = thirdparty.PlayerSummaryResponse{
			AvatarHash:      DefaultAvatarHash,
			PersonaName:     "name-" + steamID.String(),
			ProfileState:    1,
			RealName:        "randy marsh",
			SteamId:         steamID.String(),
			TimeCreated:     time.Now().Add(-500 * time.Hour).Unix(),
			VisibilityState: 5,
		}
	}

	return resp, nil
}

func (f FakeTFAPI) Friends(_ context.Context, _ steamid.SteamID) ([]thirdparty.SteamFriend, error) {
	return []thirdparty.SteamFriend{}, nil
}
