package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrCreateRequest = errors.New("failed to create logstf request")
	ErrRequestFailed = errors.New("failed to query logstf")
	ErrDecode        = errors.New("failed to decode logstf response")
)

const logsTFURL = "https://logs.tf/api/v1/log?player=%s"

type LogsTFResult struct {
	Success    bool `json:"success"`
	Results    int  `json:"results"`
	Total      int  `json:"total"`
	Parameters struct {
		Player   string `json:"player"`
		Uploader any    `json:"uploader"`
		Title    any    `json:"title"`
		Map      any    `json:"map"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
	} `json:"parameters"`
	Logs []struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Map     string `json:"map"`
		Date    int    `json:"date"`
		Views   int    `json:"views"`
		Players int    `json:"players"`
	} `json:"logs"`
}

// LogsTFOverview queries the logstf api for metadata about a players logs
// http://logs.tf/api/v1/log?title=X&uploader=Y&player=Z&limit=N&offset=N
func LogsTFOverview(ctx context.Context, sid steamid.SteamID) (*LogsTFResult, error) {
	httpClient := httphelper.NewHTTPClient()

	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(localCtx, http.MethodGet, fmt.Sprintf(logsTFURL, sid.String()), nil)
	if reqErr != nil {
		return nil, errors.Join(reqErr, ErrCreateRequest)
	}

	response, errGet := httpClient.Do(req)
	if errGet != nil {
		return nil, errors.Join(errGet, ErrRequestFailed)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	var logsTFResult LogsTFResult
	if errUnmarshal := json.NewDecoder(response.Body).Decode(&logsTFResult); errUnmarshal != nil {
		return nil, errors.Join(errGet, ErrDecode)
	}

	return &logsTFResult, nil
}
