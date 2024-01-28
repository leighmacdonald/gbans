package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type chatUsecase struct {
	cr          domain.ChatRepository
	fu          domain.WordFilterUsecase
	su          domain.StateUsecase
	bu          domain.BanUsecase
	pu          domain.PersonUsecase
	du          domain.DiscordUsecase
	cu          domain.ConfigUsecase
	tracker     *chat.Tracker
	owner       steamid.SID64
	log         *zap.Logger
	pingDiscord bool
}

func NewChatUsecase(ctx context.Context, log *zap.Logger, cu domain.ConfigUsecase, cr domain.ChatRepository, fu domain.WordFilterUsecase, su domain.StateUsecase, bu domain.BanUsecase, pu domain.PersonUsecase,
	du domain.DiscordUsecase) domain.ChatUsecase {

	conf := cu.Config()
	uc := &chatUsecase{cr: cr, fu: fu, log: log, su: su, bu: bu, pu: pu, du: du, pingDiscord: conf.Filter.PingDiscord}

	uc.tracker = chat.NewTracker(log, conf.Filter.MatchTimeout, conf.Filter.Dry, conf.Filter.MaxWeight, uc.onWarningHandler, uc.onWarningExceeded)

	go uc.tracker.Start(ctx, conf.Filter.CheckTimeout)

	return uc
}

func (u chatUsecase) WarningState() map[string][]domain.UserWarning {
	return u.tracker.State()
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
		errBan = u.su.Kick(ctx, newWarning.UserMessage.SteamID, newWarning.WarnReason)
	}

	if errBan != nil {
		return errors.Join(errBan, domain.ErrWarnActionApply)
	}

	var person domain.Person
	if personErr := u.pu.GetPersonBySteamID(ctx, newWarning.UserMessage.SteamID, &person); personErr != nil {
		return personErr
	}

	newWarning.MatchedFilter.TriggerCount++
	if errSave := u.fu.SaveFilter(ctx, newWarning.MatchedFilter); errSave != nil {
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
	if errSave := u.fu.SaveFilter(ctx, newWarning.MatchedFilter); errSave != nil {
		u.log.Error("Failed to update filter trigger count", zap.Error(errSave))
	}

	if !newWarning.MatchedFilter.IsEnabled {
		return nil
	}

	if errPSay := u.su.PSay(ctx, newWarning.UserMessage.SteamID, msg); errPSay != nil {
		return errors.Join(errPSay, state.ErrRCONCommand)
	}

	return nil

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
