package thirdparty

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/mxpv/patreon-go.v1"
	"time"
)

func NewPatreonClient() (*patreon.Client, error) {
	//ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	//
	//tc := oauth2.NewClient(ctx, ts)
	//
	//client := patreon.NewClient(tc)
	//u, err := client.FetchUser()
	//log.Println(u)
	//return client, err

	oAuthConfig := oauth2.Config{
		ClientID:     config.Patreon.ClientId,
		ClientSecret: config.Patreon.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  patreon.AuthorizationURL,
			TokenURL: patreon.AccessTokenURL,
		},
		Scopes: []string{"users", "pledges-to-me", "campaigns", "my-campaign"},
	}

	tok := &oauth2.Token{
		AccessToken:  config.Patreon.CreatorAccessToken,
		RefreshToken: config.Patreon.CreatorRefreshToken,
		// Must be non-nil, otherwise token will not be expired
		Expiry: time.Now().Add(15 * time.Second),
	}

	tc := oAuthConfig.Client(context.Background(), tok)
	client := patreon.NewClient(tc)

	_, err := client.FetchUser()
	if err != nil {
		panic(err)
	}

	return client, testPatreon(client)
}

func patreonGetTiers(client *patreon.Client) error {
	campaigns, campaignsErr := client.FetchCampaign()
	if campaignsErr != nil {
		return campaignsErr
	}
	for _, camp := range campaigns.Data {
		log.Print(camp.Attributes)
	}
	return nil
}

func testPatreon(client *patreon.Client) error {
	campaignResponse, err := client.FetchCampaign()
	if err != nil {
		panic(err)
	}

	campaignId := campaignResponse.Data[0].ID

	cursor := ""
	page := 1

	for {
		pledgesResponse, err := client.FetchPledges(campaignId,
			patreon.WithPageSize(25),
			patreon.WithCursor(cursor))

		if err != nil {
			panic(err)
		}

		// Get all the users in an easy-to-lookup way
		users := make(map[string]*patreon.User)
		for _, item := range pledgesResponse.Included.Items {
			u, ok := item.(*patreon.User)
			if !ok {
				continue
			}

			users[u.ID] = u
		}

		fmt.Printf("Page %d\r\n", page)

		// Loop over the pledges to get e.g. their amount and user name
		for _, pledge := range pledgesResponse.Data {
			amount := pledge.Attributes.AmountCents
			patronId := pledge.Relationships.Patron.Data.ID
			patronFullName := users[patronId].Attributes.FullName

			fmt.Printf("%s is pledging %d cents\r\n", patronFullName, amount)
		}

		// Get the link to the next page of pledges
		nextLink := pledgesResponse.Links.Next
		if nextLink == "" {
			break
		}

		cursor = nextLink
		page++
	}

	return nil
}
