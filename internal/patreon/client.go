package patreon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/json"
)

const (
	defaultAPIBaseURL = "https://www.patreon.com"
	apiUserAgent      = "gbans - Patreon Sync"

	// campaignFields are explicitly requested: APIv2 returns no attributes
	// unless they are listed in fields[campaign].
	campaignFields = "creation_name,image_small_url,image_url,summary,thanks_msg,patron_count," +
		"url,vanity,is_monthly,created_at,published_at"
)

// Campaign is the subset of the Patreon v2 campaign resource used by gbans.
type Campaign struct {
	ID            string
	CreationName  string
	ImageSmallURL string
	ImageURL      string
	Summary       string
	ThanksMsg     string
	PatronCount   int32
	URL           string
	Vanity        string
	IsMonthly     bool
	CreatedAt     time.Time
	PublishedAt   time.Time
}

// Client is a minimal Patreon APIv2 (JSON:API) client. It uses bearer tokens
// directly; OAuth code/token exchange still goes through the token endpoint.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func newClient(token string) *Client {
	return &Client{baseURL: defaultAPIBaseURL, token: token, httpClient: httphelper.NewClient()}
}

func newClientForURL(baseURL string, token string) *Client {
	return &Client{baseURL: baseURL, token: token, httpClient: httphelper.NewClient()}
}

// jsonAPIResource is a single JSON:API resource object.
//
//nolint:tagliatelle // Patreon wire format is snake_case.
type jsonAPIResource[Attrs any] struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes Attrs  `json:"attributes"`
}

//nolint:tagliatelle // Patreon wire format is snake_case.
type jsonAPIList[Attrs any] struct {
	Data []jsonAPIResource[Attrs] `json:"data"`
}

//nolint:tagliatelle // Patreon wire format is snake_case.
type jsonAPISingle[Attrs any] struct {
	Data jsonAPIResource[Attrs] `json:"data"`
}

//nolint:tagliatelle // Patreon wire format is snake_case.
type jsonAPIError struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}

//nolint:tagliatelle // Patreon wire format is snake_case.
type jsonAPIErrors struct {
	Errors []jsonAPIError `json:"errors"`
}

//nolint:tagliatelle // Patreon wire format is snake_case.
type campaignAttributes struct {
	CreationName  *string `json:"creation_name"`
	ImageSmallURL *string `json:"image_small_url"`
	ImageURL      *string `json:"image_url"`
	Summary       *string `json:"summary"`
	ThanksMsg     *string `json:"thanks_msg"`
	PatronCount   *int    `json:"patron_count"`
	URL           *string `json:"url"`
	Vanity        *string `json:"vanity"`
	IsMonthly     *bool   `json:"is_monthly"`
	CreatedAt     *string `json:"created_at"`
	PublishedAt   *string `json:"published_at"`
}

type userAttributes struct{}

func valueOrZero(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func parseAPIv2Time(value *string) time.Time {
	if value == nil || *value == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrQueryPatreon, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", apiUserAgent)

	resp, errResp := c.httpClient.Do(req)
	if errResp != nil {
		return nil, fmt.Errorf("%w: %w", ErrQueryPatreon, errResp)
	}

	return resp, nil
}

func decodeAPIv2Error(body io.Reader) error {
	errDoc, errDec := json.Decode[jsonAPIErrors](body)
	if errDec != nil || len(errDoc.Errors) == 0 {
		return ErrQueryPatreon
	}

	first := errDoc.Errors[0]
	if first.Detail != "" {
		return fmt.Errorf("%w: %s", ErrQueryPatreon, first.Detail) //nolint:err113
	}

	return fmt.Errorf("%w: %s", ErrQueryPatreon, first.Title) //nolint:err113
}

// fetchCampaigns returns the campaigns owned by the token owner (creator
// token). Requires the campaigns scope.
func (c *Client) fetchCampaigns(ctx context.Context) ([]Campaign, error) {
	query := url.Values{}
	query.Set("fields[campaign]", campaignFields)

	resp, err := c.get(ctx, "/api/oauth2/v2/campaigns", query)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIv2Error(resp.Body)
	}

	doc, errDec := json.Decode[jsonAPIList[campaignAttributes]](resp.Body)
	if errDec != nil {
		return nil, fmt.Errorf("%w: %w", ErrQueryPatreon, errDec)
	}

	campaigns := make([]Campaign, 0, len(doc.Data))
	for _, item := range doc.Data {
		attrs := item.Attributes
		patronCount := 0
		if attrs.PatronCount != nil {
			patronCount = *attrs.PatronCount
		}
		isMonthly := false
		if attrs.IsMonthly != nil {
			isMonthly = *attrs.IsMonthly
		}
		campaigns = append(campaigns, Campaign{
			ID:            item.ID,
			CreationName:  valueOrZero(attrs.CreationName),
			ImageSmallURL: valueOrZero(attrs.ImageSmallURL),
			ImageURL:      valueOrZero(attrs.ImageURL),
			Summary:       valueOrZero(attrs.Summary),
			ThanksMsg:     valueOrZero(attrs.ThanksMsg),
			PatronCount:   int32(patronCount), //nolint:gosec
			URL:           valueOrZero(attrs.URL),
			Vanity:        valueOrZero(attrs.Vanity),
			IsMonthly:     isMonthly,
			CreatedAt:     parseAPIv2Time(attrs.CreatedAt),
			PublishedAt:   parseAPIv2Time(attrs.PublishedAt),
		})
	}

	return campaigns, nil
}

// fetchIdentityUserID returns the Patreon user id for the token owner.
// Requires the identity scope.
func (c *Client) fetchIdentityUserID(ctx context.Context) (string, error) {
	resp, err := c.get(ctx, "/api/oauth2/v2/identity", nil)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", decodeAPIv2Error(resp.Body)
	}

	doc, errDec := json.Decode[jsonAPISingle[userAttributes]](resp.Body)
	if errDec != nil {
		return "", fmt.Errorf("%w: %w", ErrQueryPatreon, errDec)
	}

	if doc.Data.ID == "" {
		return "", ErrQueryPatreon
	}

	return doc.Data.ID, nil
}
