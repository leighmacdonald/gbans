package chat

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/person/permission"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ChatUsecase struct {
	repository    chat.ChatRepository
	wordFilters   chat.WordFilterUsecase
	bans          ban.BanUsecase
	persons       person.PersonUsecase
	notifications notification.NotificationUsecase
	state         state.StateUsecase
	warningMu     *sync.RWMutex
	dry           bool
	maxWeight     int
	warnings      map[steamid.SteamID][]UserWarning
	owner         steamid.SteamID
	matchTimeout  time.Duration
	checkTimeout  time.Duration

	pingDiscord bool
}

func NewChatUsecase(config *config.ConfigUsecase, chatRepository chat.ChatRepository,
	filters chat.WordFilterUsecase, stateUsecase state.StateUsecase, bans ban.BanUsecase,
	persons person.PersonUsecase, notifications notification.NotificationUsecase,
) *ChatUsecase {
	conf := config.Config()

	return &ChatUsecase{
		repository:    chatRepository,
		wordFilters:   filters,
		bans:          bans,
		persons:       persons,
		notifications: notifications,
		state:         stateUsecase,
		pingDiscord:   conf.Filters.PingDiscord,
		warnings:      make(map[steamid.SteamID][]UserWarning),
		warningMu:     &sync.RWMutex{},
		matchTimeout:  time.Duration(conf.Filters.MatchTimeout) * time.Second,
		dry:           conf.Filters.Dry,
		maxWeight:     conf.Filters.MaxWeight,
		owner:         steamid.New(conf.Owner),
		checkTimeout:  time.Duration(conf.Filters.CheckTimeout) * time.Second,
	}
}

func (u ChatUsecase) onWarningExceeded(ctx context.Context, newWarning NewUserWarning) error {
	var (
		newBan ban.BannedPerson
		errBan error
		req    ban.BanOpts
	)

	if newWarning.MatchedFilter.Action == FilterActionBan || newWarning.MatchedFilter.Action == FilterActionMute {
		req = ban.BanOpts{
			TargetID:   newWarning.UserMessage.SteamID,
			Duration:   newWarning.MatchedFilter.Duration,
			Reason:     newWarning.WarnReason,
			ReasonText: "",
			Note:       "Automatic warning ban",
		}
	}

	admin, errAdmin := u.persons.GetPersonBySteamID(ctx, nil, u.owner)
	if errAdmin != nil {
		return errAdmin
	}

	switch newWarning.MatchedFilter.Action {
	case FilterActionMute:
		req.BanType = ban.NoComm
		newBan, errBan = u.bans.Ban(ctx, admin.ToUserProfile(), ban.System, req)
	case FilterActionBan:
		req.BanType = ban.Banned
		newBan, errBan = u.bans.Ban(ctx, admin.ToUserProfile(), ban.System, req)
	case FilterActionKick:
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

	u.notifications.Enqueue(ctx, notification.NewDiscordNotification(
		discord.ChannelWordFilterLog,
		discord.WarningMessage(newWarning, newBan)))

	return nil
}

func (u ChatUsecase) onWarningHandler(ctx context.Context, newWarning NewUserWarning) error {
	msg := "[WARN] Please refrain from using slurs/toxicity (see: rules & MOTD). " +
		"Further offenses will result in mutes/bans"

	newWarning.MatchedFilter.TriggerCount++

	admin, errAdmin := u.persons.GetPersonBySteamID(ctx, nil, u.owner)
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
func (u ChatUsecase) State() map[string][]UserWarning {
	u.warningMu.RLock()
	defer u.warningMu.RUnlock()

	out := make(map[string][]UserWarning)

	for steamID, v := range u.warnings {
		var warnings []UserWarning

		warnings = append(warnings, v...)

		out[steamID.String()] = warnings
	}

	return out
}

func (u ChatUsecase) check(now time.Time) {
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

func (u ChatUsecase) trigger(ctx context.Context, newWarn NewUserWarning) {
	if !newWarn.UserMessage.SteamID.Valid() {
		return
	}

	if u.dry {
		return
	}

	u.warningMu.Lock()

	_, found := u.warnings[newWarn.UserMessage.SteamID]
	if !found {
		u.warnings[newWarn.UserMessage.SteamID] = []UserWarning{}
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

func (u ChatUsecase) Start(ctx context.Context) {
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

func (u ChatUsecase) WarningState() map[string][]UserWarning {
	return u.State()
}

func (u ChatUsecase) GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error) {
	return u.repository.GetPersonMessage(ctx, messageID)
}

func (u ChatUsecase) AddChatHistory(ctx context.Context, message *PersonMessage) error {
	return u.repository.AddChatHistory(ctx, message)
}

func (u ChatUsecase) QueryChatHistory(ctx context.Context, user person.PersonInfo, req ChatHistoryQueryFilter) ([]QueryChatHistoryResult, error) {
	if req.Limit <= 0 || (req.Limit > 100 && !user.HasPermission(permission.PModerator)) {
		req.Limit = 100
	}

	if !user.HasPermission(permission.PModerator) {
		req.Unrestricted = false
	} else {
		req.Unrestricted = true
	}

	return u.repository.QueryChatHistory(ctx, req)
}

func (u ChatUsecase) GetPersonMessageContext(ctx context.Context, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error) {
	if paddedMessageCount > 100 || paddedMessageCount <= 0 {
		paddedMessageCount = 100
	}

	msg, errMsg := u.GetPersonMessage(ctx, messageID)
	if errMsg != nil {
		return nil, errMsg
	}

	return u.repository.GetPersonMessageContext(ctx, msg.ServerID, messageID, paddedMessageCount)
}

func (u ChatUsecase) TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error) {
	return u.repository.TopChatters(ctx, count)
}
