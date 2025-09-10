package match

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/pkg/log"
)

// DataUpdater handles periodically updating a data source and caching the results via user supplied func.
type DataUpdater[T any] struct {
	data       T
	update     func() (T, error)
	updateRate time.Duration
	dataMu     *sync.RWMutex
}

func NewDataUpdater[T any](updateRate time.Duration, updateFn func() (T, error)) *DataUpdater[T] {
	return &DataUpdater[T]{
		update:     updateFn,
		dataMu:     &sync.RWMutex{},
		updateRate: updateRate,
	}
}

func (c *DataUpdater[T]) Data() T { //nolint:ireturn
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	return c.data
}

func (c *DataUpdater[T]) execUpdate() {
	newData, errUpdate := c.update()
	if errUpdate != nil && !errors.Is(errUpdate, database.ErrNoResult) {
		slog.Error("Failed to update data source", log.ErrAttr(errUpdate))

		return
	}

	c.dataMu.Lock()
	c.data = newData
	c.dataMu.Unlock()
}

func (c *DataUpdater[T]) Start(ctx context.Context) {
	refreshTimer := time.NewTicker(c.updateRate)

	for {
		select {
		case <-refreshTimer.C:
			c.execUpdate()
		case <-ctx.Done():
			return
		}
	}
}
