package domain

import "context"

type SRCDSRepository interface{}

type SRCDSUsecase interface {
	ServerAuth(ctx context.Context, req ServerAuthReq) (string, error)
	Report(ctx context.Context, currentUser UserProfile, req CreateReportReq) (*Report, error)
}

type ServerAuthReq struct {
	Key string `json:"key"`
}
