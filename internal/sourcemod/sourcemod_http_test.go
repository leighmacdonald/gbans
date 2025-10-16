package sourcemod_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/tests"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}
