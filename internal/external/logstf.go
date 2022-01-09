package external

import (
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io/ioutil"
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
func LogsTFOverview(sid steamid.SID64) (*LogsTFResult, error) {
	c := util.NewHTTPClient()
	resp, err := c.Get(fmt.Sprintf("https://logs.tf/api/v1/log?player=%d", sid.Int64()))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to query logstf")
	}
	b, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		return nil, errors.Wrapf(err, "Failed to read logstf body")
	}
	var r LogsTFResult
	if err3 := json.Unmarshal(b, &r); err3 != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal logstf body")
	}
	return &r, nil
}
