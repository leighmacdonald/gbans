package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

const bdAPIURL = "https://bd-api.roto.lol/sourcebans/%s"

type BDSourceBansRecord struct {
	BanID       int           `json:"ban_id"`
	SiteName    string        `json:"site_name"`
	SiteID      int           `json:"site_id"`
	PersonaName string        `json:"persona_name"`
	SteamID     steamid.SID64 `json:"steam_id"`
	Reason      string        `json:"reason"`
	Duration    time.Duration `json:"duration"`
	Permanent   bool          `json:"permanent"`
	CreatedOn   time.Time     `json:"created_on"`
}

func BDSourceBans(ctx context.Context, steamID steamid.SID64) ([]BDSourceBansRecord, error) {
	client := &http.Client{Timeout: time.Second * 10}
	url := fmt.Sprintf(bdAPIURL, steamID)

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errReq != nil {
		return nil, errors.Join(errReq, errs.ErrCreateRequest)
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Join(errResp, errs.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var records []BDSourceBansRecord
	if errJSON := json.NewDecoder(resp.Body).Decode(&records); errJSON != nil {
		return nil, errors.Join(errJSON, errs.ErrRequestDecode)
	}

	return records, nil
}
