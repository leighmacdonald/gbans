package chat

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type chatUsecase struct {
	cr           domain.ChatRepository
	wfu          domain.WordFilterUsecase
	su           domain.StateUsecase
	bu           domain.BanUsecase
	pu           domain.PersonUsecase
	du           domain.DiscordUsecase
	cu           domain.ConfigUsecase
	log          *zap.Logger
	st           domain.StateUsecase
	warningMu    *sync.RWMutex
	dry          bool
	maxWeight    int
	warnings     map[steamid.SID64][]domain.UserWarning
	WarningChan  chan domain.NewUserWarning
	owner        steamid.SID64
	matchTimeout time.Duration
	checkTimeout time.Duration

	pingDiscord bool
}

func NewChatUsecase(log *zap.Logger, cu domain.ConfigUsecase, cr domain.ChatRepository,
	wfu domain.WordFilterUsecase, su domain.StateUsecase, bu domain.BanUsecase, pu domain.PersonUsecase,
	du domain.DiscordUsecase, st domain.StateUsecase,
) domain.ChatUsecase {
	conf := cu.Config()

	uc := &chatUsecase{
		cr:           cr,
		wfu:          wfu,
		log:          log,
		su:           su,
		bu:           bu,
		pu:           pu,
		du:           du,
		st:           st,
		pingDiscord:  conf.Filter.PingDiscord,
		warnings:     make(map[steamid.SID64][]domain.UserWarning),
		warningMu:    &sync.RWMutex{},
		WarningChan:  make(chan domain.NewUserWarning),
		matchTimeout: conf.Filter.MatchTimeout,
		dry:          conf.Filter.Dry,
		maxWeight:    conf.Filter.MaxWeight,
		checkTimeout: conf.Filter.CheckTimeout,
	}

	return uc
}

func (u chatUsecase) onWarningExceeded(ctx context.Context, newWarning domain.NewUserWarning) error {
	var (
		errBan   error
		banSteam domain.BanSteam
	)

	if newWarning.MatchedFilter.Action == domain.Ban || newWarning.MatchedFilter.Action == domain.Mute {
		duration, errDuration := util.ParseDuration(newWarning.MatchedFilter.Duration)
		if errDuration != nil {
			return fmt.Errorf("invalid duration: %w", errDuration)
		}

		if errNewBan := domain.NewBanSteam(ctx, domain.StringSID(u.owner),
			domain.StringSID(newWarning.UserMessage.SteamID.String()),
			duration,
			newWarning.WarnReason,
			"",
			"Automatic warning ban",
			domain.System,
			0,
			domain.NoComm,
			false,
			&banSteam); errNewBan != nil {
			return errors.Join(errNewBan, domain.ErrFailedToBan)
		}
	}

	switch newWarning.MatchedFilter.Action {
	case domain.Mute:
		banSteam.BanType = domain.NoComm
		errBan = u.bu.BanSteam(ctx, &banSteam)
	case domain.Ban:
		banSteam.BanType = domain.Banned
		errBan = u.bu.BanSteam(ctx, &banSteam)
	case domain.Kick:
		errBan = u.st.Kick(ctx, newWarning.UserMessage.SteamID, newWarning.WarnReason)
	}

	if errBan != nil {
		return errors.Join(errBan, domain.ErrWarnActionApply)
	}

	var person domain.Person
	if personErr := u.pu.GetPersonBySteamID(ctx, newWarning.UserMessage.SteamID, &person); personErr != nil {
		return personErr
	}

	newWarning.MatchedFilter.TriggerCount++
	if errSave := u.wfu.SaveFilter(ctx, newWarning.MatchedFilter); errSave != nil {
		u.log.Error("Failed to update filter trigger count", zap.Error(errSave))
	}

	if !u.pingDiscord {
		return nil
	}

	u.du.SendPayload(domain.ChannelModLog, discord.WarningMessage(newWarning, banSteam, person))

	return nil
}

func (u chatUsecase) onWarningHandler(ctx context.Context, newWarning domain.NewUserWarning) error {
	msg := fmt.Sprintf("[WARN] Please refrain from using slurs/toxicity (see: rules & MOTD). " +
		"Further offenses will result in mutes/bans")

	newWarning.MatchedFilter.TriggerCount++
	if errSave := u.wfu.SaveFilter(ctx, newWarning.MatchedFilter); errSave != nil {
		u.log.Error("Failed to update filter trigger count", zap.Error(errSave))
	}

	if !newWarning.MatchedFilter.IsEnabled {
		return nil
	}

	if errPSay := u.st.PSay(ctx, newWarning.UserMessage.SteamID, msg); errPSay != nil {
		return errors.Join(errPSay, state.ErrRCONCommand)
	}

	return nil
}

// State returns a string key so its more easily portable to frontend js w/o using BigInt.
func (u chatUsecase) State() map[string][]domain.UserWarning {
	u.warningMu.RLock()
	defer u.warningMu.RUnlock()

	out := make(map[string][]domain.UserWarning)

	for steamID, v := range u.warnings {
		var warnings []domain.UserWarning

		warnings = append(warnings, v...)

		out[steamID.String()] = warnings
	}

	return out
}

func (u chatUsecase) check(now time.Time) {
	u.warningMu.Lock()
	defer u.warningMu.Unlock()

	for steamID := range u.warnings {
		for warnIdx, warning := range u.warnings[steamID] {
			if now.Sub(warning.CreatedOn) > u.matchTimeout {
				if len(u.warnings[steamID]) > 1 {
					u.warnings[steamID] = append(u.warnings[steamID][:warnIdx], u.warnings[steamID][warnIdx+1])
				} else {
					delete(u.warnings, steamID)
				}
			}
		}

		var newSum int
		for idx := range u.warnings[steamID] {
			newSum += u.warnings[steamID][idx].MatchedFilter.Weight
			u.warnings[steamID][idx].CurrentTotal = newSum
		}
	}
}

func (u chatUsecase) trigger(ctx context.Context, newWarn domain.NewUserWarning) {
	if !newWarn.UserMessage.SteamID.Valid() {
		return
	}

	if !u.dry {
		u.warningMu.Lock()

		_, found := u.warnings[newWarn.UserMessage.SteamID]
		if !found {
			u.warnings[newWarn.UserMessage.SteamID] = []domain.UserWarning{}
		}

		var (
			currentWeight = newWarn.MatchedFilter.Weight
			count         int
		)

		for _, existing := range u.warnings[newWarn.UserMessage.SteamID] {
			currentWeight += existing.MatchedFilter.Weight
			count++
		}

		newWarn.CurrentTotal = currentWeight + newWarn.MatchedFilter.Weight

		u.warnings[newWarn.UserMessage.SteamID] = append(u.warnings[newWarn.UserMessage.SteamID], newWarn.UserWarning)

		u.warningMu.Unlock()

		if currentWeight > u.maxWeight {
			u.log.Info("Warn limit exceeded",
				zap.Int64("sid64", newWarn.UserMessage.SteamID.Int64()),
				zap.Int("count", count), zap.Int("weight", currentWeight))

			if err := u.onWarningExceeded(ctx, newWarn); err != nil {
				u.log.Error("Failed to execute warning exceeded handler", zap.Error(err))
			}
		} else {
			if err := u.onWarningHandler(ctx, newWarn); err != nil {
				u.log.Error("Failed to execute warning handler", zap.Error(err))
			}
		}
	}
}

func (u chatUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(u.checkTimeout)

	for {
		select {
		case now := <-ticker.C:
			u.check(now)
			ticker.Reset(u.checkTimeout)
		case newWarn := <-u.WarningChan:
			u.trigger(ctx, newWarn)
		case <-ctx.Done():
			return
		}
	}
}

func (u chatUsecase) WarningState() map[string][]domain.UserWarning {
	return u.State()
}

func (u chatUsecase) GetPersonMessage(ctx context.Context, messageID int64, msg *domain.QueryChatHistoryResult) error {
	return u.cr.GetPersonMessage(ctx, messageID, msg)
}

func (u chatUsecase) AddChatHistory(ctx context.Context, message *domain.PersonMessage) error {
	return u.AddChatHistory(ctx, message)
}

func (u chatUsecase) QueryChatHistory(ctx context.Context, filters domain.ChatHistoryQueryFilter) ([]domain.QueryChatHistoryResult, int64, error) {
	return u.cr.QueryChatHistory(ctx, filters)
}

func (u chatUsecase) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]domain.QueryChatHistoryResult, error) {
	return u.GetPersonMessageContext(ctx, serverID, messageID, paddedMessageCount)
}

func (u chatUsecase) TopChatters(ctx context.Context, count uint64) ([]domain.TopChatterResult, error) {
	return u.cr.TopChatters(ctx, count)
}
