package external

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const rglUrl = "https://rgl.gg/Public/PlayerProfile.aspx?p=%d"

var (
	ErrNoProfile = errors.New("no profile")
)

type RGLProfile struct {
	SteamId  steamid.SID64
	Division string
	Team     string
}

func GetRGLProfile(ctx context.Context, sid steamid.SID64, profile *RGLProfile) error {
	if !sid.Valid() {
		return errors.New("Invalid profile")
	}
	l := log.WithFields(log.Fields{"sid": sid.Int64(), "service": "rgl"})
	l.Debugf("Fetching profile")
	httpClient := &http.Client{Timeout: time.Second * 15}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(rglUrl, sid.Int64()), nil)
	if err != nil {
		return err
	}
	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errResp
	}
	// Load the HTML document
	doc, errR := goquery.NewDocumentFromReader(resp.Body)
	if errR != nil {
		return errR
	}
	errMsg := doc.Find("#ContentPlaceHolder1_Main_lblTimelineMessage").Text()
	if strings.Contains(strings.ToLower(errMsg), "player does not exist in rgl") {
		return ErrNoProfile
	}
	profile.Division = doc.Find("a#ContentPlaceHolder1_Main_hlDivisionName").Text()
	profile.Team = doc.Find("a#ContentPlaceHolder1_Main_hlTeamName").Text()

	l.WithFields(log.Fields{"team": profile.Team, "div": profile.Division}).
		Debugf("Fetched rgl profile successfully")
	return nil
}
