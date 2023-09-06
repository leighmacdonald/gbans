package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DataUpdater handles periodically updating a data source and caching the results via user supplied func.
type DataUpdater[T any] struct {
	data       T
	update     func() (T, error)
	updateChan chan any
	updateRate time.Duration
	dataMu     *sync.RWMutex
	log        *zap.Logger
}

func NewDataUpdater[T any](log *zap.Logger, updateRate time.Duration, updateFn func() (T, error)) *DataUpdater[T] {
	return &DataUpdater[T]{
		log:        log.Named("cache"),
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
			if errUpdate != nil && !errors.Is(errUpdate, ErrNoResult) {
				c.log.Error("Failed to update data source", zap.Error(errUpdate))

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
