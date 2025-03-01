package domain

import "context"

type ServiceStarter interface {
	Start(ctx context.Context)
}
