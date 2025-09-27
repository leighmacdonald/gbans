package chat

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

type HistoryQueryFilter struct {
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

func (f HistoryQueryFilter) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SourceID)

	return sid, sid.Valid()
}

type TopChatterResult struct {
	Name    string
	SteamID steamid.SteamID
	Count   int
}

type Message struct {
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

type PersonMessages []Message

type QueryChatHistoryResult struct {
	Message
	Pattern string `json:"pattern"`
}

type Chat struct {
	repository    Repository
	wordFilters   WordFilters
	bans          ban.Bans
	config        *config.Configuration
	persons       domain.PersonProvider
	notifications notification.Notifier
	state         *servers.State
	warningMu     *sync.RWMutex
	dry           bool
	maxWeight     int
	warnings      map[steamid.SteamID][]UserWarning
	owner         steamid.SteamID
	matchTimeout  time.Duration
	checkTimeout  time.Duration
	pingDiscord   bool

	WarningChan chan NewUserWarning
}

func NewChat(repo Repository, config *config.Configuration, filters WordFilters,
	bans ban.Bans, persons domain.PersonProvider, notifications notification.Notifier,
) *Chat {
	conf := config.Config()

	return &Chat{
		repository:    repo,
		wordFilters:   filters,
		bans:          bans,
		notifications: notifications,
		persons:       persons,
		pingDiscord:   conf.Filters.PingDiscord,
		warnings:      make(map[steamid.SteamID][]UserWarning),
		warningMu:     &sync.RWMutex{},
		matchTimeout:  time.Duration(conf.Filters.MatchTimeout) * time.Second,
		dry:           conf.Filters.Dry,
		maxWeight:     conf.Filters.MaxWeight,
		owner:         steamid.New(conf.Owner),
		checkTimeout:  time.Duration(conf.Filters.CheckTimeout) * time.Second,
		config:        config,
	}
}

func (u Chat) Start(ctx context.Context, events *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]) {
	cleanupTicker := time.NewTicker(u.checkTimeout)
	eventChan := make(chan logparse.ServerEvent)
	if errRegister := events.Consume(eventChan, logparse.Connected, logparse.Say, logparse.SayTeam); errRegister != nil {
		slog.Warn("logWriter Tried to register duplicate reader channel", log.ErrAttr(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTicker.C:
			u.cleanupExpired()
		case evt := <-eventChan:
			if errEvent := u.handleEvent(ctx, evt); errEvent != nil {
				slog.Error("Failed to handle chat event", slog.String("error", errEvent.Error()))
			}
		}
	}
}

func (u *Chat) handleEvent(ctx context.Context, evt logparse.ServerEvent) error {
	switch evt.EventType {
	case logparse.Connected:
		connectEvent, ok := evt.Event.(logparse.ConnectedEvt)
		if !ok {
			return nil
		}

		connectMsg := "Player connected with username: " + connectEvent.Name

		return u.handleMessage(ctx, evt, connectEvent.SourcePlayer, connectMsg, false, connectEvent.CreatedOn, banDomain.Username)
	case logparse.Say:
		fallthrough
	case logparse.SayTeam:
		sayEvent, ok := evt.Event.(logparse.SayEvt)
		if !ok {
			return nil
		}

		return u.handleMessage(ctx, evt, sayEvent.SourcePlayer, sayEvent.Msg, sayEvent.Team, sayEvent.CreatedOn, banDomain.Language)
	}

	return nil
}

func (u *Chat) cleanupExpired() {
	u.warningMu.Lock()
	defer u.warningMu.Unlock()

	for steamID := range u.warnings {
		for warnIdx, warning := range u.warnings[steamID] {
			if time.Since(warning.CreatedOn) > u.matchTimeout {
				if len(u.warnings[steamID]) > 1 {
					u.warnings[steamID] = append(u.warnings[steamID][:warnIdx], u.warnings[steamID][warnIdx+1])
				} else {
					delete(u.warnings, steamID)
				}
			}
		}
	}
}

func (u Chat) handleMessage(ctx context.Context, evt logparse.ServerEvent, person logparse.SourcePlayer, msg string, team bool, created time.Time, reason banDomain.Reason) error {
	if msg == "" {
		return nil
	}

	_, errPerson := u.persons.GetOrCreatePersonBySteamID(ctx, nil, person.SID)
	if errPerson != nil && !errors.Is(errPerson, database.ErrDuplicate) {
		return errPerson
	}

	userMsg := Message{
		SteamID:     person.SID,
		PersonaName: strings.ToValidUTF8(person.Name, "_"),
		ServerName:  evt.ServerName,
		ServerID:    evt.ServerID,
		Body:        strings.ToValidUTF8(msg, "_"),
		Team:        team,
		CreatedOn:   created,
	}

	if errChat := u.AddChatHistory(ctx, &userMsg); errChat != nil {
		return errChat
	}

	matchedFilter := u.wordFilters.Check(userMsg.Body)
	if len(matchedFilter) == 0 {
		return nil
	}

	if errSaveMatch := u.wordFilters.AddMessageFilterMatch(ctx, userMsg.PersonMessageID, matchedFilter[0].FilterID); errSaveMatch != nil {
		slog.Error("Failed to save message findMatch status", log.ErrAttr(errSaveMatch))
	}

	matchResult := matchedFilter[0]

	u.WarningChan <- NewUserWarning{
		UserMessage: userMsg,
		PlayerID:    person.PID,
		UserWarning: UserWarning{
			WarnReason: reason,
			Message:    userMsg.Body,
			// todo
			// Matched:       matchResult,
			MatchedFilter: matchResult,
			CreatedOn:     time.Now(),
			Personaname:   userMsg.PersonaName,
			Avatar:        userMsg.AvatarHash,
			ServerName:    userMsg.ServerName,
			ServerID:      userMsg.ServerID,
			SteamID:       userMsg.SteamID.String(),
		},
	}

	return nil
}

func (u Chat) onWarningExceeded(ctx context.Context, newWarning NewUserWarning) error {
	var (
		errBan error
		req    ban.Opts
		newBan ban.Ban
	)

	if newWarning.MatchedFilter.Action == FilterActionBan || newWarning.MatchedFilter.Action == FilterActionMute {
		dur, errDur := duration.Parse(newWarning.MatchedFilter.Duration)
		if errDur != nil {
			return errors.Join(errDur, banDomain.ErrDuration)
		}
		req = ban.Opts{
			TargetID:   newWarning.UserMessage.SteamID,
			Reason:     newWarning.WarnReason,
			ReasonText: "",
			Note:       "Automatic warning ban",
			Duration:   dur,
		}
	}

	admin, errAdmin := u.persons.GetOrCreatePersonBySteamID(ctx, nil, u.owner)
	if errAdmin != nil {
		return errAdmin
	}

	switch newWarning.MatchedFilter.Action {
	case FilterActionMute:
		req.BanType = banDomain.NoComm
		newBan, errBan = u.bans.Create(ctx, req)
	case FilterActionBan:
		req.BanType = banDomain.Banned
		newBan, errBan = u.bans.Create(ctx, req)
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

	u.notifications.Send(notification.NewDiscord(
		u.config.Config().Discord.WordFilterLogChannelID,
		WarningMessage(newWarning, newBan)))

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

func (u Chat) GetPersonMessageByID(ctx context.Context, personMessageID int64) (Message, error) {
	return u.repository.GetPersonMessageByID(ctx, personMessageID)
}

func (u Chat) WarningState() map[string][]UserWarning {
	return u.State()
}

func (u Chat) GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error) {
	return u.repository.GetPersonMessage(ctx, messageID)
}

func (u Chat) AddChatHistory(ctx context.Context, message *Message) error {
	return u.repository.AddChatHistory(ctx, message)
}

func (u Chat) QueryChatHistory(ctx context.Context, user domain.PersonInfo, req HistoryQueryFilter) ([]QueryChatHistoryResult, error) {
	if req.Limit <= 0 || (req.Limit > 100 && !user.HasPermission(permission.Moderator)) {
		req.Limit = 100
	}

	if !user.HasPermission(permission.Moderator) {
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
