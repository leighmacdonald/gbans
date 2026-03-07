package demostats_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/leighmacdonald/gbans/internal/fs"
	"github.com/leighmacdonald/gbans/pkg/demostats"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	var parsed demostats.Demo

	demoPath := fs.FindFile("testdata/koth_ashville_final.dem.json", "gbans")
	file, err := os.Open(demoPath)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(file).Decode(&parsed))
	require.Equal(t, "tf", parsed.Game)
}
