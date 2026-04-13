package votes

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/ptr"
	v1 "github.com/leighmacdonald/gbans/internal/votes/v1"
	"github.com/leighmacdonald/gbans/internal/votes/v1/votesv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	votesv1connect.UnimplementedVotesServiceHandler

	votes Votes
}

func NewService(votes Votes) Service {
	return Service{votes: votes}
}

func (s Service) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	votes, count, errVotes := s.votes.Query(ctx, Query{
		Filter:        query.FromRPC(req.Filter),
		SourceIDField: httphelper.SourceIDField{},
		TargetIDField: httphelper.TargetIDField{},
		ServerID:      ptr.From(req.ServerId),
		Name:          ptr.From(req.Name),
		Success:       ptr.From(req.Success),
		Code:          ptr.From(req.Code),
	})
	if errVotes != nil && !errors.Is(errVotes, database.ErrNoResult) {
		slog.Error("Failed to query votes", errVotes)

		return nil, connect.NewError(connect.CodeInternal, httphelper.ErrInternal)
	}

	if votes == nil {
		votes = []Result{}
	}

	resp := v1.QueryResponse{Results: make([]*v1.VoteResult, len(votes)), Count: &count}

	for idx, vote := range votes {
		resp.Results[idx] = &v1.VoteResult{
			VoteId:           &vote.VoteID,
			SourceId:         ptr.To(vote.SourceID.String()),
			SourceName:       &vote.SourceName,
			SourceAvatarHash: &vote.SourceAvatarHash,
			TargetId:         ptr.To(vote.TargetID.String()),
			TargetName:       &vote.TargetName,
			TargetAvatarHash: &vote.TargetAvatarHash,
			Name:             &vote.Name,
			Success:          &vote.Success,
			ServerId:         &vote.ServerID,
			ServerName:       &vote.ServerName,
			Code:             ptr.To(v1.VoteCode(vote.Code)),
			CreatedOn:        timestamppb.New(vote.CreatedOn),
		}
	}
	return &resp, nil
}
