package updater

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/pkg/log"
)

// Updater handles periodically updating a data source and caching the results via user supplied func.
type Updater[T any] struct {
	data       T
	updateFn   func() (T, error)
	updateRate time.Duration
	dataMu     *sync.RWMutex
}

func New[T any](updateInterval time.Duration, updateFn func() (T, error)) *Updater[T] {
	return &Updater[T]{
		updateFn:   updateFn,
		dataMu:     &sync.RWMutex{},
		updateRate: updateInterval,
	}
}

func (c *Updater[T]) Data() T { //nolint:ireturn
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	return c.data
}

func (c *Updater[T]) update() {
	newData, errUpdate := c.updateFn()
	if errUpdate != nil && !errors.Is(errUpdate, database.ErrNoResult) {
		slog.Error("Failed to update data source", log.ErrAttr(errUpdate))

		return
	}

	c.dataMu.Lock()
	c.data = newData
	c.dataMu.Unlock()
}

func (c *Updater[T]) Start(ctx context.Context) {
	refreshTimer := time.NewTicker(c.updateRate)

	for {
		select {
		case <-refreshTimer.C:
			c.update()
		case <-ctx.Done():
			return
		}
	}
}
