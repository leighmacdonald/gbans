package thirdparty

import (
	"context"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/mxpv/patreon-go.v1"
)

func NewPatreonClient(ctx context.Context, token string) (*patreon.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	tc := oauth2.NewClient(ctx, ts)

	client := patreon.NewClient(tc)
	u, err := client.FetchUser()
	log.Println(u)
	return client, err
}
