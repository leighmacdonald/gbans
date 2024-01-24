package domain

import "gopkg.in/mxpv/patreon-go.v1"

type Patreon interface {
	Tiers() ([]patreon.Campaign, error)
	Pledges() ([]patreon.Pledge, map[string]*patreon.User, error)
}
