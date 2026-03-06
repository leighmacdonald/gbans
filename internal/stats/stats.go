package stats

import (
	"context"
	"errors"
	"fmt"

	"github.com/leighmacdonald/gbans/pkg/demoparse"
)

var (
	ErrInvalidState = errors.New("Invalid demo state")
)

type Stats struct {
	repo Repository
}

func New(repo Repository) Stats {
	return Stats{repo: repo}
}

func (s Stats) ImportDemo(ctx context.Context, demo demoparse.Demo) error {
	if demo.DemoType != demoparse.HL2Demo {
		return fmt.Errorf("%w: Invalid demo type", ErrInvalidState)
	}

	if demo.Server == "" {
		return fmt.Errorf("%w: Invalid server name", ErrInvalidState)
	}

	if demo.Filename == "" {
		return fmt.Errorf("%w: Invalid file name", ErrInvalidState)
	}

	if len(demo.SteamIDs()) < 4 {
		return fmt.Errorf("%w: Not enough players", ErrInvalidState)
	}

	return nil
}
