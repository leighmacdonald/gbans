package thirdparty

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
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

func GetRGLProfile(ctx context.Context, sid64 steamid.SID64, profile *RGLProfile) error {
	if !sid64.Valid() {
		return errors.New("Invalid profile")
	}
	httpClient := &http.Client{Timeout: time.Second * 15}
	request, errNewRequest := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(rglUrl, sid64.Int64()), nil)
	if errNewRequest != nil {
		return errNewRequest
	}
	response, errDoRequest := httpClient.Do(request)
	if errDoRequest != nil {
		return errDoRequest
	}
	// Load the HTML document
	document, errDocumentReader := goquery.NewDocumentFromReader(response.Body)
	if errDocumentReader != nil {
		return errDocumentReader
	}
	errFind := document.Find("#ContentPlaceHolder1_Main_lblTimelineMessage").Text()
	if strings.Contains(strings.ToLower(errFind), "player does not exist in rgl") {
		return ErrNoProfile
	}
	profile.Division = document.Find("a#ContentPlaceHolder1_Main_hlDivisionName").Text()
	profile.Team = document.Find("a#ContentPlaceHolder1_Main_hlTeamName").Text()

	return nil
}
