package action

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestActions(t *testing.T) {
	ctx := context.Background()
	c := make(chan *Action, 5)
	go func(inChan chan *Action) {
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			case a := <-inChan:
				switch i {
				case 0:
					a.Result <- Result{Err: nil, Message: "test", Value: 100}
				case 1:
					a.Result <- Result{Err: consts.ErrInvalidSID}
				}
			}
			i++
		}
	}(c)
	Register(c)
	req := NewKick(Core, "76561199040918801", "76561197992870439", "test")
	req2 := NewKick(Core, "76561199040918802", "76561197992870440", "test2")
	result := <-req.Enqueue().Done()
	result2 := <-req2.Enqueue().Done()

	require.NoError(t, result.Err)
	require.Equal(t, "test", result.Message)
	require.Equal(t, 100, result.Value.(int))
	require.Equal(t, consts.ErrInvalidSID, result2.Err)
}
