package chat

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ChatHistoryQueryFilter struct {
	query.Filter
	domain.SourceIDField
	Personaname   string     `json:"personaname,omitempty"`
	ServerID      int        `json:"server_id,omitempty"`
	DateStart     *time.Time `json:"date_start,omitempty"`
	DateEnd       *time.Time `json:"date_end,omitempty"`
	Unrestricted  bool       `json:"-"`
	DontCalcTotal bool       `json:"-"`
	FlaggedOnly   bool       `json:"flagged_only"`
}

func (f ChatHistoryQueryFilter) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SourceID)

	return sid, sid.Valid()
}

type TopChatterResult struct {
	Name    string
	SteamID steamid.SteamID
	Count   int
}

type PersonMessage struct {
	PersonMessageID   int64           `json:"person_message_id"`
	MatchID           uuid.UUID       `json:"match_id"`
	SteamID           steamid.SteamID `json:"steam_id"`
	AvatarHash        string          `json:"avatar_hash"`
	PersonaName       string          `json:"persona_name"`
	ServerName        string          `json:"server_name"`
	ServerID          int             `json:"server_id"`
	Body              string          `json:"body"`
	Team              bool            `json:"team"`
	CreatedOn         time.Time       `json:"created_on"`
	AutoFilterFlagged int64           `json:"auto_filter_flagged"`
}

type PersonMessages []PersonMessage

type QueryChatHistoryResult struct {
	PersonMessage
	Pattern string `json:"pattern"`
}

type Chat struct {
	repository    *ChatRepository
	wordFilters   WordFilters
	bans          ban.Bans
	persons       domain.PersonProvider
	notifications notification.Notifications
	state         *servers.State
	warningMu     *sync.RWMutex
	dry           bool
	maxWeight     int
	warnings      map[steamid.SteamID][]UserWarning
	owner         steamid.SteamID
	matchTimeout  time.Duration
	checkTimeout  time.Duration
	pingDiscord   bool
}

func NewChat(config *config.Configuration, repo *ChatRepository, filters WordFilters,
	state *servers.State, bans ban.Bans, persons domain.PersonProvider) *Chat {
	conf := config.Config()

	return &Chat{
		repository:   repo,
		wordFilters:  filters,
		bans:         bans,
		persons:      persons,
		state:        state,
		pingDiscord:  conf.Filters.PingDiscord,
		warnings:     make(map[steamid.SteamID][]UserWarning),
		warningMu:    &sync.RWMutex{},
		matchTimeout: time.Duration(conf.Filters.MatchTimeout) * time.Second,
		dry:          conf.Filters.Dry,
		maxWeight:    conf.Filters.MaxWeight,
		owner:        steamid.New(conf.Owner),
		checkTimeout: time.Duration(conf.Filters.CheckTimeout) * time.Second,
	}
}

func (u Chat) onWarningExceeded(ctx context.Context, newWarning NewUserWarning) error {
	var (
		errBan error
		req    ban.BanOpts
	)

	if newWarning.MatchedFilter.Action == FilterActionBan || newWarning.MatchedFilter.Action == FilterActionMute {
		req = ban.BanOpts{
			TargetID:   newWarning.UserMessage.SteamID,
			Reason:     newWarning.WarnReason,
			ReasonText: "",
			Note:       "Automatic warning ban",
		}
		req.SetDuration(newWarning.MatchedFilter.Duration, time.Now().AddDate(10, 0, 0))
	}

	admin, errAdmin := u.persons.GetOrCreatePersonBySteamID(ctx, nil, u.owner)
	if errAdmin != nil {
		return errAdmin
	}

	switch newWarning.MatchedFilter.Action {
	case FilterActionMute:
		req.BanType = banDomain.NoComm
		_, errBan = u.bans.Create(ctx, req)
	case FilterActionBan:
		req.BanType = banDomain.Banned
		_, errBan = u.bans.Create(ctx, req)
	case FilterActionKick:
		// Kicks are temporary, so should be done by Player ID to avoid
		// missing players who weren't in the latest state update
		// (otherwise, kicking players very shortly after they connect
		// will usually fail).
		errBan = u.state.KickPlayerID(ctx, newWarning.PlayerID, newWarning.ServerID, newWarning.WarnReason.String())
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

	// u.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelWordFilterLog,
	// 	discord.WarningMessage(newWarning, newBan)))

	return nil
}

func (u Chat) onWarningHandler(ctx context.Context, newWarning NewUserWarning) error {
	msg := "[WARN] Please refrain from using slurs/toxicity (see: rules & MOTD). " +
		"Further offenses will result in mutes/bans"

	newWarning.MatchedFilter.TriggerCount++

	admin, errAdmin := u.persons.GetOrCreatePersonBySteamID(ctx, nil, u.owner)
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
		return errPSay
	}

	return nil
}

// State returns a string key so its more easily portable to frontend js w/o using BigInt.
func (u Chat) State() map[string][]UserWarning {
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

func (u Chat) check(now time.Time) {
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

func (u Chat) trigger(ctx context.Context, newWarn NewUserWarning) {
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

func (u Chat) Start(ctx context.Context) {
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

func (u Chat) GetPersonMessageByID(ctx context.Context, personMessageID int64) (PersonMessage, error) {
	return u.repository.GetPersonMessageByID(ctx, personMessageID)
}

func (u Chat) WarningState() map[string][]UserWarning {
	return u.State()
}

func (u Chat) GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error) {
	return u.repository.GetPersonMessage(ctx, messageID)
}

func (u Chat) AddChatHistory(ctx context.Context, message *PersonMessage) error {
	return u.repository.AddChatHistory(ctx, message)
}

func (u Chat) QueryChatHistory(ctx context.Context, user domain.PersonInfo, req ChatHistoryQueryFilter) ([]QueryChatHistoryResult, error) {
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

func (u Chat) GetPersonMessageContext(ctx context.Context, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error) {
	if paddedMessageCount > 100 || paddedMessageCount <= 0 {
		paddedMessageCount = 100
	}

	msg, errMsg := u.GetPersonMessage(ctx, messageID)
	if errMsg != nil {
		return nil, errMsg
	}

	return u.repository.GetPersonMessageContext(ctx, msg.ServerID, messageID, paddedMessageCount)
}

func (u Chat) TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error) {
	return u.repository.TopChatters(ctx, count)
}
