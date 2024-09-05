package chat

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type chatUsecase struct {
	repository    domain.ChatRepository
	wordFilters   domain.WordFilterUsecase
	bansSteam     domain.BanSteamUsecase
	persons       domain.PersonUsecase
	notifications domain.NotificationUsecase
	state         domain.StateUsecase
	warningMu     *sync.RWMutex
	dry           bool
	maxWeight     int
	warnings      map[steamid.SteamID][]domain.UserWarning
	owner         steamid.SteamID
	matchTimeout  time.Duration
	checkTimeout  time.Duration

	pingDiscord bool
}

func NewChatUsecase(config domain.ConfigUsecase, chatRepository domain.ChatRepository,
	filters domain.WordFilterUsecase, stateUsecase domain.StateUsecase, bans domain.BanSteamUsecase,
	persons domain.PersonUsecase, notifications domain.NotificationUsecase,
) domain.ChatUsecase {
	conf := config.Config()

	return &chatUsecase{
		repository:    chatRepository,
		wordFilters:   filters,
		bansSteam:     bans,
		persons:       persons,
		notifications: notifications,
		state:         stateUsecase,
		pingDiscord:   conf.Filters.PingDiscord,
		warnings:      make(map[steamid.SteamID][]domain.UserWarning),
		warningMu:     &sync.RWMutex{},
		matchTimeout:  time.Duration(conf.Filters.MatchTimeout) * time.Second,
		dry:           conf.Filters.Dry,
		maxWeight:     conf.Filters.MaxWeight,
		owner:         steamid.New(conf.Owner),
		checkTimeout:  time.Duration(conf.Filters.CheckTimeout) * time.Second,
	}
}

func (u chatUsecase) onWarningExceeded(ctx context.Context, newWarning domain.NewUserWarning) error {
	var (
		ban    domain.BannedSteamPerson
		errBan error
		req    domain.RequestBanSteamCreate
	)

	if newWarning.MatchedFilter.Action == domain.Ban || newWarning.MatchedFilter.Action == domain.Mute {
		req = domain.RequestBanSteamCreate{
			SourceIDField: domain.SourceIDField{},
			TargetIDField: domain.TargetIDField{TargetID: newWarning.UserMessage.SteamID.String()},
			Duration:      newWarning.MatchedFilter.Duration,
			Reason:        newWarning.WarnReason,
			ReasonText:    "",
			Note:          "Automatic warning ban",
		}
	}

	admin, errAdmin := u.persons.GetPersonBySteamID(ctx, u.owner)
	if errAdmin != nil {
		return errAdmin
	}

	switch newWarning.MatchedFilter.Action {
	case domain.Mute:
		req.BanType = domain.NoComm
		ban, errBan = u.bansSteam.Ban(ctx, admin.ToUserProfile(), domain.System, req)
	case domain.Ban:
		req.BanType = domain.Banned
		ban, errBan = u.bansSteam.Ban(ctx, admin.ToUserProfile(), domain.System, req)
	case domain.Kick:
		// Kicks are temporary, so should be done by Player ID to avoid
		// missing players who weren't in the latest state update
		// (otherwise, kicking players very shortly after they connect
		// will usually fail).
		errBan = u.state.KickPlayerID(ctx, newWarning.PlayerID, newWarning.ServerID, newWarning.WarnReason)
	}

	if errBan != nil {
		return errors.Join(errBan, domain.ErrWarnActionApply)
	}

	newWarning.MatchedFilter.TriggerCount++

	_, errSave := u.wordFilters.Edit(ctx, admin, newWarning.MatchedFilter.FilterID, newWarning.MatchedFilter)
	if errSave != nil {
		return errSave
	}

	if !u.pingDiscord {
		return nil
	}

	u.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		domain.ChannelWordFilterLog,
		discord.WarningMessage(newWarning, ban)))

	return nil
}

func (u chatUsecase) onWarningHandler(ctx context.Context, newWarning domain.NewUserWarning) error {
	msg := "[WARN] Please refrain from using slurs/toxicity (see: rules & MOTD). " +
		"Further offenses will result in mutes/bans"

	newWarning.MatchedFilter.TriggerCount++

	admin, errAdmin := u.persons.GetPersonBySteamID(ctx, u.owner)
	if errAdmin != nil {
		return errAdmin
	}

	_, errSave := u.wordFilters.Edit(ctx, admin, newWarning.MatchedFilter.FilterID, newWarning.MatchedFilter)
	if errSave != nil {
		return errSave
	}

	if !newWarning.MatchedFilter.IsEnabled {
		return nil
	}

	if errPSay := u.state.PSay(ctx, newWarning.UserMessage.SteamID, msg); errPSay != nil {
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

	if u.dry {
		return
	}

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
		slog.Info("Warn limit exceeded",
			slog.String("sid64", newWarn.UserMessage.SteamID.String()),
			slog.Int("count", count),
			slog.Int("weight", currentWeight))

		if err := u.onWarningExceeded(ctx, newWarn); err != nil {
			slog.Error("Failed to execute warning exceeded handler", log.ErrAttr(err))
		}
	} else {
		if err := u.onWarningHandler(ctx, newWarn); err != nil {
			slog.Error("Failed to execute warning handler", log.ErrAttr(err))
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
		case newWarn := <-u.repository.GetWarningChan():
			u.trigger(ctx, newWarn)
		case <-ctx.Done():
			return
		}
	}
}

func (u chatUsecase) WarningState() map[string][]domain.UserWarning {
	return u.State()
}

func (u chatUsecase) GetPersonMessage(ctx context.Context, messageID int64) (domain.QueryChatHistoryResult, error) {
	return u.repository.GetPersonMessage(ctx, messageID)
}

func (u chatUsecase) AddChatHistory(ctx context.Context, message *domain.PersonMessage) error {
	return u.repository.AddChatHistory(ctx, message)
}

func (u chatUsecase) QueryChatHistory(ctx context.Context, user domain.PersonInfo, req domain.ChatHistoryQueryFilter) ([]domain.QueryChatHistoryResult, error) {
	if req.Limit <= 0 || (req.Limit > 100 && !user.HasPermission(domain.PModerator)) {
		req.Limit = 100
	}

	if !user.HasPermission(domain.PModerator) {
		req.Unrestricted = false
	} else {
		req.Unrestricted = true
	}

	return u.repository.QueryChatHistory(ctx, req)
}

func (u chatUsecase) GetPersonMessageContext(ctx context.Context, messageID int64, paddedMessageCount int) ([]domain.QueryChatHistoryResult, error) {
	if paddedMessageCount > 100 || paddedMessageCount <= 0 {
		paddedMessageCount = 100
	}

	msg, errMsg := u.GetPersonMessage(ctx, messageID)
	if errMsg != nil {
		return nil, errMsg
	}

	return u.repository.GetPersonMessageContext(ctx, msg.ServerID, messageID, paddedMessageCount)
}

func (u chatUsecase) TopChatters(ctx context.Context, count uint64) ([]domain.TopChatterResult, error) {
	return u.repository.TopChatters(ctx, count)
}
