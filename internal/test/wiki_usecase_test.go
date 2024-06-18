package test

import (
	"context"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	_, errStore := NewDB(testCtx)
	if errStore != nil {
		panic(errStore)
	}

	m.Run()
}
