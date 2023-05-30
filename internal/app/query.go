package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"time"
)

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date
func PersonBySID(ctx context.Context, sid steamid.SID64, person *store.Person) error {
	if errGetPerson := store.GetOrCreatePersonBySteamID(ctx, sid, person); errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get person instance: %d", sid)
	}
	if person.IsNew || config.Now().Sub(person.UpdatedOnSteam) > time.Minute*60 {
		summaries, errSummaries := steamweb.PlayerSummaries(steamid.Collection{sid})
		if errSummaries != nil {
			return errors.Wrapf(errSummaries, "Failed to get Player summary: %v", errSummaries)
		}
		if len(summaries) > 0 {
			s := summaries[0]
			person.PlayerSummary = &s
		} else {
			return errors.Errorf("Failed to fetch Player summary for %d", sid)
		}
		vac, errBans := thirdparty.FetchPlayerBans(steamid.Collection{sid})
		if errBans != nil || len(vac) != 1 {
			return errors.Wrapf(errSummaries, "Failed to get Player ban state: %v", errSummaries)
		} else {
			person.CommunityBanned = vac[0].CommunityBanned
			person.VACBans = vac[0].NumberOfVACBans
			person.GameBans = vac[0].NumberOfGameBans
			person.EconomyBan = vac[0].EconomyBan
			person.CommunityBanned = vac[0].CommunityBanned
			person.DaysSinceLastBan = vac[0].DaysSinceLastBan
		}
		person.UpdatedOnSteam = config.Now()
	}
	person.SteamID = sid
	if errSavePerson := store.SavePerson(ctx, person); errSavePerson != nil {
		return errors.Wrapf(errSavePerson, "Failed to save person")
	}
	return nil
}
