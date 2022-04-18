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
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		logsTFResult, errOverview := LogsTFOverview(sid)
		if errOverview != nil {
			return
		}
		hist.LogsCount = logsTFResult.Total
	}()
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		var rglProf RGLProfile
		if errGetRGL := GetRGLProfile(ctx, sid, &rglProf); errGetRGL != nil {
			return
		}
		hist.RGLDiv = rglProf.Division
		hist.RGLTeam = rglProf.Team
	}()
	waitGroup.Wait()
	return nil
}
