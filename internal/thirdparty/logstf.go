package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

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
func LogsTFOverview(ctx context.Context, sid steamid.SID64) (*LogsTFResult, error) {
	httpClient := util.NewHTTPClient()

	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(localCtx, http.MethodGet,
		fmt.Sprintf("https://logs.tf/api/v1/log?player=%s", sid), nil)
	if reqErr != nil {
		return nil, errors.Join(reqErr, errors.New("Failed to create request"))
	}

	response, errGet := httpClient.Do(req)
	if errGet != nil {
		return nil, errors.Join(errGet, errors.New("Failed to query logstf"))
	}

	defer func() {
		_ = response.Body.Close()
	}()

	body, errReadBody := io.ReadAll(response.Body)
	if errReadBody != nil {
		return nil, errors.Join(errGet, errors.New("Failed to read logstf body"))
	}

	var logsTFResult LogsTFResult
	if errUnmarshal := json.Unmarshal(body, &logsTFResult); errUnmarshal != nil {
		return nil, errors.Join(errGet, errors.New("Failed to unmarshal logstf body"))
	}

	return &logsTFResult, nil
}
