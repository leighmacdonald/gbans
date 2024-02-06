package domain

import "context"

type BuildInfo struct {
	BuildVersion string
	Commit       string
	Date         string
}

type ServiceStarter interface {
	Start(ctx context.Context)
}
