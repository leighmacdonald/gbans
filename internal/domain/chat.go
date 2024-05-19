package domain

import "context"

type ChatRepository interface {
	GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error)
	Start(ctx context.Context)
	TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error)
	AddChatHistory(ctx context.Context, message *PersonMessage) error
	QueryChatHistory(ctx context.Context, filters ChatHistoryQueryFilter) ([]QueryChatHistoryResult, error)
	GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error)
	GetWarningChan() chan NewUserWarning
}

type ChatUsecase interface {
	Start(ctx context.Context)
	WarningState() map[string][]UserWarning
	GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error)
	AddChatHistory(ctx context.Context, message *PersonMessage) error
	QueryChatHistory(ctx context.Context, user PersonInfo, filters ChatHistoryQueryFilter) ([]QueryChatHistoryResult, error)
	GetPersonMessageContext(ctx context.Context, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error)
	TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error)
}
