package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func (app *App) showReportMeta(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)
	updateChan := make(chan any)

	go func() {
		time.Sleep(time.Second * 5)
		updateChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			reports, _, errReports := app.db.GetReports(ctx, domain.ReportQueryFilter{
				QueryFilter: domain.QueryFilter{
					Limit: 0,
				},
			})
			if errReports != nil {
				app.log.Error("failed to fetch reports for report metadata", zap.Error(errReports))

				continue
			}

			var (
				now  = time.Now()
				meta domain.ReportMeta
			)

			for _, report := range reports {
				if report.ReportStatus == domain.ClosedWithAction || report.ReportStatus == domain.ClosedWithoutAction {
					meta.TotalClosed++

					continue
				}

				meta.TotalOpen++

				if report.ReportStatus == domain.NeedMoreInfo {
					meta.NeedInfo++
				} else {
					meta.Open++
				}

				switch {
				case now.Sub(report.CreatedOn) > time.Hour*24*7:
					meta.OpenWeek++
				case now.Sub(report.CreatedOn) > time.Hour*24*3:
					meta.Open3Days++
				case now.Sub(report.CreatedOn) > time.Hour*24:
					meta.Open1Day++
				default:
					meta.OpenNew++
				}
			}

			conf := app.Config()

			app.SendPayload(conf.Discord.LogChannelID, discord.ReportStatsMessage(meta, conf.ExtURLRaw("/admin/reports")))
		case <-ctx.Done():
			app.log.Debug("showReportMeta shutting down")

			return
		}
	}
}

func cleanupTasks(ctx context.Context, database database.Stores, logger *zap.Logger) {
	var (
		log    = logger.Named("cleanupTasks")
		ticker = time.NewTicker(time.Hour * 24)
	)

	for {
		select {
		case <-ticker.C:
			if err := database.PrunePersonAuth(ctx); err != nil && !errors.Is(err, errs.ErrNoResult) {
				log.Error("Error pruning expired refresh tokens", zap.Error(err))
			}
		case <-ctx.Done():
			log.Debug("cleanupTasks shutting down")

			return
		}
	}
}

func (app *App) notificationSender(ctx context.Context) {
	log := app.log.Named("notificationSender")

	for {
		select {
		case notification := <-app.notificationChan:
			go func() {
				if errSend := app.SendNotification(ctx, notification); errSend != nil {
					log.Error("Failed to send user notification", zap.Error(errSend))
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

func (app *App) updateProfiles(ctx context.Context, people domain.People) error {
	if len(people) > 100 {
		return errSteamAPIArgLimit
	}

	var (
		banStates           []steamweb.PlayerBanState
		summaries           []steamweb.PlayerSummary
		steamIDs            = people.ToSteamIDCollection()
		errGroup, cancelCtx = errgroup.WithContext(ctx)
	)

	errGroup.Go(func() error {
		newBanStates, errBans := thirdparty.FetchPlayerBans(cancelCtx, steamIDs)
		if errBans != nil {
			return errors.Join(errBans, errFetchSteamBans)
		}

		banStates = newBanStates

		return nil
	})

	errGroup.Go(func() error {
		newSummaries, errSummaries := steamweb.PlayerSummaries(cancelCtx, steamIDs)
		if errSummaries != nil {
			return errors.Join(errSummaries, errSteamAPISummaries)
		}

		summaries = newSummaries

		return nil
	})

	if errFetch := errGroup.Wait(); errFetch != nil {
		return errors.Join(errFetch, errSteamAPI)
	}

	for _, curPerson := range people {
		person := curPerson
		person.IsNew = false
		person.UpdatedOnSteam = time.Now()

		for _, newSummary := range summaries {
			summary := newSummary
			if person.SteamID != summary.SteamID {
				continue
			}

			person.PlayerSummary = &summary

			break
		}

		for _, banState := range banStates {
			if person.SteamID != banState.SteamID {
				continue
			}

			person.CommunityBanned = banState.CommunityBanned
			person.VACBans = banState.NumberOfVACBans
			person.GameBans = banState.NumberOfGameBans
			person.EconomyBan = banState.EconomyBan
			person.CommunityBanned = banState.CommunityBanned
			person.DaysSinceLastBan = banState.DaysSinceLastBan
		}

		if errSavePerson := app.db.SavePerson(ctx, &person); errSavePerson != nil {
			return errors.Join(errSavePerson, errUpdatePerson)
		}
	}

	app.log.Debug("Updated steam profiles and vac data", zap.Int("count", len(people)))

	return nil
}

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution.
func (app *App) profileUpdater(ctx context.Context) {
	var (
		log    = app.log.Named("profileUpdate")
		run    = make(chan any)
		ticker = time.NewTicker(time.Second * 300)
	)

	go func() {
		run <- true
	}()

	for {
		select {
		case <-ticker.C:
			run <- true
		case <-run:
			localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
			people, errGetExpired := app.db.GetExpiredProfiles(localCtx, 100)

			if errGetExpired != nil || len(people) == 0 {
				cancel()

				continue
			}

			if errUpdate := app.updateProfiles(localCtx, people); errUpdate != nil {
				log.Error("Failed to update profiles", zap.Error(errUpdate))
			}

			cancel()
		case <-ctx.Done():
			log.Debug("profileUpdater shutting down")

			return
		}
	}
}

// banSweeper periodically will query the database for expired bans and remove them.
func (app *App) banSweeper(ctx context.Context) {
	var (
		log    = app.log.Named("banSweeper")
		ticker = time.NewTicker(time.Minute)
	)

	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)

			go func() {
				defer waitGroup.Done()

				expiredBans, errExpiredBans := app.db.GetExpiredBans(ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, errs.ErrNoResult) {
					log.Error("Failed to get expired expiredBans", zap.Error(errExpiredBans))
				} else {
					for _, expiredBan := range expiredBans {
						ban := expiredBan
						if errDrop := app.db.DropBan(ctx, &ban, false); errDrop != nil {
							log.Error("Failed to drop expired expiredBan", zap.Error(errDrop))
						} else {
							var person domain.Person
							if errPerson := app.db.GetPersonBySteamID(ctx, ban.TargetID, &person); errPerson != nil {
								log.Error("Failed to get expired Person", zap.Error(errPerson))

								continue
							}

							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}

							conf := app.Config()

							app.discord.SendPayload(conf.Discord.LogChannelID, discord.BanExpiresMessage(ban, person, conf.ExtURL(ban)))

							log.Info("Ban expired",
								zap.String("reason", ban.Reason.String()),
								zap.Int64("sid64", ban.TargetID.Int64()), zap.String("name", name))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredNetBans, errExpiredNetBans := app.db.GetExpiredNetBans(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, errs.ErrNoResult) {
					log.Warn("Failed to get expired network bans", zap.Error(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						expiredBan := expiredNetBan
						if errDropBanNet := app.db.DropBanNet(ctx, &expiredBan); errDropBanNet != nil {
							log.Error("Failed to drop expired network expiredNetBan", zap.Error(errDropBanNet))
						} else {
							log.Info("IP ban expired", zap.String("cidr", expiredBan.String()))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredASNBans, errExpiredASNBans := app.db.GetExpiredASNBans(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, errs.ErrNoResult) {
					log.Error("Failed to get expired asn bans", zap.Error(errExpiredASNBans))
				} else {
					for _, expiredASNBan := range expiredASNBans {
						expired := expiredASNBan
						if errDropASN := app.db.DropBanASN(ctx, &expired); errDropASN != nil {
							log.Error("Failed to drop expired asn ban", zap.Error(errDropASN))
						} else {
							log.Info("ASN ban expired", zap.Int64("ban_id", expired.BanASNId))
						}
					}
				}
			}()

			waitGroup.Wait()
		case <-ctx.Done():
			log.Debug("banSweeper shutting down")

			return
		}
	}
}
