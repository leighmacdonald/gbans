package domain

import "github.com/leighmacdonald/gbans/internal/config"

type ConfigRepository interface {
	Config() *config.Config
}

type ConfigUsecase interface {
	Config() *config.Config
}
