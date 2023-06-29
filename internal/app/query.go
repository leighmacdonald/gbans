package app

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date.
func (app *App) PersonBySID(ctx context.Context, sid steamid.SID64, person *store.Person) error {
	if errGetPerson := app.db.GetOrCreatePersonBySteamID(ctx, sid, person); errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get person instance: %s", sid)
	}
	if person.IsNew || time.Since(person.UpdatedOnSteam) > time.Hour*24 {
		summaries, errSummaries := steamweb.PlayerSummaries(ctx, steamid.Collection{sid})
		if errSummaries != nil {
			return errors.Wrapf(errSummaries, "Failed to get Player summary: %v", errSummaries)
		}
		if len(summaries) > 0 {
			s := summaries[0]
			person.PlayerSummary = &s
		} else {
			app.log.Warn("Failed to update profile summary", zap.Error(errSummaries), zap.Int64("sid", sid.Int64()))
			// return errors.Errorf("Failed to fetch Player summary for %d", sid)
		}
		vac, errBans := thirdparty.FetchPlayerBans(ctx, steamid.Collection{sid})
		if errBans != nil || len(vac) != 1 {
			app.log.Warn("Failed to update ban status", zap.Error(errBans), zap.Int64("sid", sid.Int64()))
			// return errors.Wrapf(errBans, "Failed to get Player ban state: %v", errBans)
		} else {
			person.CommunityBanned = vac[0].CommunityBanned
			person.VACBans = vac[0].NumberOfVACBans
			person.GameBans = vac[0].NumberOfGameBans
			person.EconomyBan = steamweb.EconBanNone
			person.CommunityBanned = vac[0].CommunityBanned
			person.DaysSinceLastBan = vac[0].DaysSinceLastBan
		}
		person.UpdatedOnSteam = config.Now()
	}
	person.SteamID = sid
	if errSavePerson := app.db.SavePerson(ctx, person); errSavePerson != nil {
		return errors.Wrapf(errSavePerson, "Failed to save person")
	}
	return nil
}
