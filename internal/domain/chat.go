package domain

import "context"

type ChatRepository interface {
	GetPersonMessage(ctx context.Context, messageID int64, msg *QueryChatHistoryResult) error
	TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error)
	AddChatHistory(ctx context.Context, message *PersonMessage) error
	QueryChatHistory(ctx context.Context, filters ChatHistoryQueryFilter) ([]QueryChatHistoryResult, int64, error)
	GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error)
}

type ChatUsecase interface {
	WarningState() map[string][]UserWarning
	GetPersonMessage(ctx context.Context, messageID int64, msg *QueryChatHistoryResult) error
	AddChatHistory(ctx context.Context, message *PersonMessage) error
	QueryChatHistory(ctx context.Context, filters ChatHistoryQueryFilter) ([]QueryChatHistoryResult, int64, error)
	GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error)
	TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error)
}
