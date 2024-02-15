package match

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
)

// DataUpdater handles periodically updating a data source and caching the results via user supplied func.
type DataUpdater[T any] struct {
	data       T
	update     func() (T, error)
	updateChan chan any
	updateRate time.Duration
	dataMu     *sync.RWMutex
}

func NewDataUpdater[T any](updateRate time.Duration, updateFn func() (T, error)) *DataUpdater[T] {
	return &DataUpdater[T]{
		update:     updateFn,
		updateChan: make(chan any),
		dataMu:     &sync.RWMutex{},
		updateRate: updateRate,
	}
}

func (c *DataUpdater[T]) Data() T { //nolint:ireturn
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	return c.data
}

func (c *DataUpdater[T]) Start(ctx context.Context) {
	go func() {
		c.updateChan <- true
	}()

	refreshTimer := time.NewTicker(c.updateRate)

	for {
		select {
		case <-c.updateChan:
			newData, errUpdate := c.update()
			if errUpdate != nil && !errors.Is(errUpdate, domain.ErrNoResult) {
				slog.Error("Failed to update data source", log.ErrAttr(errUpdate))

				return
			}

			c.dataMu.Lock()
			c.data = newData
			c.dataMu.Unlock()
		case <-refreshTimer.C:
			c.updateChan <- true
		case <-ctx.Done():
			return
		}
	}
}
