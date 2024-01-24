package usecase

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"net"
	"time"
)

type personUsecase struct {
	personRepo     domain.PersonRepository
	contextTimeout time.Duration
}

func (p *personUsecase) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) SavePerson(ctx context.Context, person *domain.Person) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (domain.People, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPeople(ctx context.Context, filter domain.PlayerQuery) (domain.People, int64, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonByDiscordID(ctx context.Context, discordID string, person *domain.Person) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetExpiredProfiles(ctx context.Context, limit uint64) ([]domain.Person, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonMessageByID(ctx context.Context, personMessageID int64, msg *domain.PersonMessage) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) QueryConnectionHistory(ctx context.Context, opts domain.ConnectionHistoryQueryFilter) ([]domain.PersonConnection, int64, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) QueryChatHistory(ctx context.Context, filters domain.ChatHistoryQueryFilter) ([]domain.QueryChatHistoryResult, int64, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonMessage(ctx context.Context, messageID int64, msg *domain.QueryChatHistoryResult) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]domain.QueryChatHistoryResult, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (domain.PersonConnections, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *domain.PersonAuth) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) SavePersonAuth(ctx context.Context, auth *domain.PersonAuth) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) DeletePersonAuth(ctx context.Context, authID int64) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) PrunePersonAuth(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) SendNotification(ctx context.Context, targetID steamid.SID64, severity domain.NotificationSeverity, message string, link string) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonNotifications(ctx context.Context, filters domain.NotificationQuery) ([]domain.UserNotification, int64, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetSteamIdsAbove(ctx context.Context, privilege domain.Privilege) (steamid.Collection, error) {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) GetPersonSettings(ctx context.Context, steamID steamid.SID64, settings *domain.PersonSettings) error {
	//TODO implement me
	panic("implement me")
}

func (p personUsecase) SavePersonSettings(ctx context.Context, settings *domain.PersonSettings) error {
	//TODO implement me
	panic("implement me")
}

func NewPersonUsecase(pr domain.PersonRepository, timeout time.Duration) domain.PersonUsecase {
	return &personUsecase{
		personRepo:     pr,
		contextTimeout: timeout,
	}
}
