package domain

import "context"

type AppealRepository interface {
	SaveBanMessage(ctx context.Context, message *BanAppealMessage) error
	GetBanMessages(ctx context.Context, banID int64) ([]BanAppealMessage, error)
	GetBanMessageByID(ctx context.Context, banMessageID int, message *BanAppealMessage) error
	DropBanMessage(ctx context.Context, message *BanAppealMessage) error
	GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, int64, error)
}

type AppealUsecase interface {
	SaveBanMessage(ctx context.Context, curUserProfile UserProfile, req BanAppealMessage) (*BanAppealMessage, error)
	GetBanMessages(ctx context.Context, banID int64) ([]BanAppealMessage, error)
	GetBanMessageByID(ctx context.Context, banMessageID int, message *BanAppealMessage) error
	DropBanMessage(ctx context.Context, curUser UserProfile, message *BanAppealMessage) error
	GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, int64, error)
}

type NewBanMessage struct {
	Message string `json:"message"`
}
