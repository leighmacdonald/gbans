package external

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"sync"
)

type CompHist struct {
	RGLDiv    string
	RGLTeam   string
	LogsCount int
}

func FetchCompHist(ctx context.Context, sid steamid.SID64, hist *CompHist) error {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ltf, err := LogsTFOverview(sid)
		if err != nil {
			return
		}
		hist.LogsCount = ltf.Total
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var rglProf RGLProfile
		if err := GetRGLProfile(ctx, sid, &rglProf); err != nil {
			return
		}
		hist.RGLDiv = rglProf.Division
		hist.RGLTeam = rglProf.Team
	}()
	wg.Wait()
	return nil
}
