package ban

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/leighmacdonald/gbans/internal/opengraph"
)

// openGraphBanTags generates the tags for ban detail pages. The values are
// intentionally generic since ban details are not publicly accessible, avoiding
// leaking any personal data to crawler bots.
func openGraphBanTags(_ context.Context, _ *http.Request, params map[string]string) (opengraph.Tags, error) {
	banID, errParse := parseID(params["ban_id"])
	if errParse != nil {
		return opengraph.Tags{}, errParse
	}

	return opengraph.Tags{
		Title:       fmt.Sprintf("Ban #%d", banID),
		Description: "Ban details and appeal information.",
	}, nil
}

// openGraphReportTags generates the tags for report detail pages. The values are
// intentionally generic since report details are not publicly accessible,
// avoiding leaking any personal data to crawler bots.
func openGraphReportTags(_ context.Context, _ *http.Request, params map[string]string) (opengraph.Tags, error) {
	reportID, errParse := parseID(params["report_id"])
	if errParse != nil {
		return opengraph.Tags{}, errParse
	}

	return opengraph.Tags{
		Title:       fmt.Sprintf("Report #%d", reportID),
		Description: "Player report details.",
	}, nil
}

func parseID(value string) (int64, error) {
	parsed, errParse := strconv.ParseInt(value, 10, 32)
	if errParse != nil {
		return 0, fmt.Errorf("failed to parse entity id %q: %w", value, errParse)
	}

	return parsed, nil
}
