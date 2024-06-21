package test_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/stretchr/testify/require"
)

func TestVotes(t *testing.T) {
	t.Parallel()

	router := testRouter()
	source := getUser()
	target := getUser()
	moderator := loginUser(getModerator())

	var results domain.LazyResult
	req := domain.VoteQueryFilter{
		Success: -1,
	}
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/votes", req, http.StatusOK, moderator, &results)
	require.Empty(t, results.Data)

	require.NoError(t, votesRepo.AddResult(context.Background(), domain.VoteResult{
		SourceID:         source.SteamID,
		SourceName:       source.PersonaName,
		SourceAvatarHash: source.AvatarHash,
		TargetID:         target.SteamID,
		TargetName:       target.PersonaName,
		TargetAvatarHash: target.AvatarHash,
		Name:             "kick",
		Success:          false,
		ServerID:         testServer.ServerID,
		ServerName:       testServer.ShortName,
		Code:             logparse.VoteCodeFailNoOutnumberYes,
		CreatedOn:        time.Now(),
	}))

	testEndpointWithReceiver(t, router, http.MethodPost, "/api/votes", req, http.StatusOK, moderator, &results)
	require.NotEmpty(t, results.Data)
}

func TestVotesPermissions(t *testing.T) {
	t.Parallel()

	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/votes",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: moderators,
		},
	})
}
