package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
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

func BDSourceBans(ctx context.Context, steamID steamid.SteamID) (map[int64][]BDSourceBansRecord, error) {
	client := &http.Client{Timeout: time.Second * 10}
	queryURL, _ := url.Parse(bdAPIURL)
	query := queryURL.Query()
	query.Set("steamids", steamID.String())
	queryURL.RawQuery = query.Encode()
	fullURL := queryURL.String()

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if errReq != nil {
		return nil, errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Accept", " application/json")

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	records := map[int64][]BDSourceBansRecord{}

	if errJSON := json.NewDecoder(resp.Body).Decode(&records); errJSON != nil {
		return nil, errors.Join(errJSON, domain.ErrRequestDecode)
	}

	return records, nil
}
