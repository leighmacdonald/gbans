package depotdownloader

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestFetchVersion(t *testing.T) {
	depotChan := func(depot Depot) error {
		log.Infof("Updated!")
		return nil
	}
	u, _ := url.Parse("wss://update.uncletopia.com")
	require.NoError(t, VersionChangeListener(u, 232250, depotChan))
}
