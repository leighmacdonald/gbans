package domain

import "context"

type AppealRepository interface {
	SaveBanMessage(ctx context.Context, message *BanAppealMessage) error
	GetBanMessages(ctx context.Context, banID int64) ([]BanAppealMessage, error)
	GetBanMessageByID(ctx context.Context, banMessageID int, message *BanAppealMessage) error
	DropBanMessage(ctx context.Context, message *BanAppealMessage) error
}

type AppealUsecase interface {
	SaveBanMessage(ctx context.Context, message *BanAppealMessage) error
	GetBanMessages(ctx context.Context, banID int64) ([]BanAppealMessage, error)
	GetBanMessageByID(ctx context.Context, banMessageID int, message *BanAppealMessage) error
	DropBanMessage(ctx context.Context, message *BanAppealMessage) error
}
