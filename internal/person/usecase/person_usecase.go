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

func NewPersonUsecase(pr domain.PersonRepository, timeout time.Duration) domain.PersonUsecase {
	return &personUsecase{
		personRepo:     pr,
		contextTimeout: timeout,
	}
}

func (p *personUsecase) GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	return p.personRepo.GetPersonBySteamID(ctx, sid64, person)
}

func (p *personUsecase) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	return p.personRepo.DropPerson(ctx, steamID)
}

func (p *personUsecase) SavePerson(ctx context.Context, person *domain.Person) error {
	return p.personRepo.SavePerson(ctx, person)
}

func (p *personUsecase) GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (domain.People, error) {
	return p.personRepo.GetPeopleBySteamID(ctx, steamIds)
}

func (p *personUsecase) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	return p.personRepo.GetSteamsAtAddress(ctx, addr)
}

func (p *personUsecase) GetPeople(ctx context.Context, filter domain.PlayerQuery) (domain.People, int64, error) {
	return p.personRepo.GetPeople(ctx, filter)
}

func (p *personUsecase) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	return p.personRepo.GetOrCreatePersonBySteamID(ctx, sid64, person)
}

func (p *personUsecase) GetPersonByDiscordID(ctx context.Context, discordID string, person *domain.Person) error {
	return p.GetPersonByDiscordID(ctx, discordID, person)
}

func (p *personUsecase) GetExpiredProfiles(ctx context.Context, limit uint64) ([]domain.Person, error) {
	return p.personRepo.GetExpiredProfiles(ctx, limit)
}

func (p *personUsecase) GetPersonMessageByID(ctx context.Context, personMessageID int64, msg *domain.PersonMessage) error {
	return p.personRepo.GetPersonMessageByID(ctx, personMessageID, msg)
}

func (p *personUsecase) QueryConnectionHistory(ctx context.Context, opts domain.ConnectionHistoryQueryFilter) ([]domain.PersonConnection, int64, error) {
	return p.personRepo.QueryConnectionHistory(ctx, opts)
}

func (p *personUsecase) QueryChatHistory(ctx context.Context, filters domain.ChatHistoryQueryFilter) ([]domain.QueryChatHistoryResult, int64, error) {
	return p.personRepo.QueryChatHistory(ctx, filters)
}

func (p *personUsecase) GetPersonMessage(ctx context.Context, messageID int64, msg *domain.QueryChatHistoryResult) error {
	return p.personRepo.GetPersonMessage(ctx, messageID, msg)
}

func (p *personUsecase) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]domain.QueryChatHistoryResult, error) {
	return p.personRepo.GetPersonMessageContext(ctx, serverID, messageID, paddedMessageCount)
}

func (p *personUsecase) GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (domain.PersonConnections, error) {
	return p.personRepo.GetPersonIPHistory(ctx, sid64, limit)
}

func (p *personUsecase) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	return p.personRepo.AddConnectionHistory(ctx, conn)
}

func (p *personUsecase) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *domain.PersonAuth) error {
	return p.personRepo.GetPersonAuthByRefreshToken(ctx, token, auth)
}

func (p *personUsecase) SavePersonAuth(ctx context.Context, auth *domain.PersonAuth) error {
	return p.personRepo.SavePersonAuth(ctx, auth)
}

func (p *personUsecase) DeletePersonAuth(ctx context.Context, authID int64) error {
	return p.personRepo.DeletePersonAuth(ctx, authID)
}

func (p *personUsecase) PrunePersonAuth(ctx context.Context) error {
	return p.personRepo.PrunePersonAuth(ctx)
}

func (p *personUsecase) SendNotification(ctx context.Context, targetID steamid.SID64, severity domain.NotificationSeverity, message string, link string) error {
	return p.personRepo.SendNotification(ctx, targetID, severity, message, link)
}

func (p *personUsecase) GetPersonNotifications(ctx context.Context, filters domain.NotificationQuery) ([]domain.UserNotification, int64, error) {
	return p.personRepo.GetPersonNotifications(ctx, filters)
}

func (p *personUsecase) GetSteamIdsAbove(ctx context.Context, privilege domain.Privilege) (steamid.Collection, error) {
	return p.personRepo.GetSteamIdsAbove(ctx, privilege)
}

func (p *personUsecase) GetPersonSettings(ctx context.Context, steamID steamid.SID64, settings *domain.PersonSettings) error {
	return p.personRepo.GetPersonSettings(ctx, steamID, settings)
}

func (p *personUsecase) SavePersonSettings(ctx context.Context, settings *domain.PersonSettings) error {
	return p.personRepo.SavePersonSettings(ctx, settings)
}
