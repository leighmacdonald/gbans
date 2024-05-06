package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const bdAPIURL = "https://bd-api.roto.lol/sourcebans?steamids=%s"

type BDSourceBansRecord struct {
	BanID       int             `json:"ban_id"`
	SiteName    string          `json:"site_name"`
	SiteID      int             `json:"site_id"`
	PersonaName string          `json:"persona_name"`
	SteamID     steamid.SteamID `json:"steam_id"`
	Reason      string          `json:"reason"`
	Duration    time.Duration   `json:"duration"`
	Permanent   bool            `json:"permanent"`
	CreatedOn   time.Time       `json:"created_on"`
}

func BDSourceBans(ctx context.Context, steamID steamid.SteamID) (map[string][]BDSourceBansRecord, error) {
	client := &http.Client{Timeout: time.Second * 10}
	url := fmt.Sprintf(bdAPIURL, steamID.String())

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errReq != nil {
		return nil, errors.Join(errReq, domain.ErrCreateRequest)
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var records map[string][]BDSourceBansRecord
	if errJSON := json.NewDecoder(resp.Body).Decode(&records); errJSON != nil {
		return nil, errors.Join(errJSON, domain.ErrRequestDecode)
	}

	return records, nil
}
