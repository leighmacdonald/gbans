package votes

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

type voteUsecase struct {
	voteRepository voteRepository
}

func NewVoteUsecase(voteRepository voteRepository) domain.VoteUsecase {
	return &voteUsecase{
		voteRepository: voteRepository,
	}
}

func (v voteUsecase) Query(ctx context.Context, filter domain.VoteQueryFilter) ([]domain.VoteRepository, error) {
	//TODO implement me
	panic("implement me")
}

func (r voteRepository) Start(ctx context.Context) {
	eventChan := make(chan logparse.ServerEvent)
	if errRegister := r.broadcaster.Consume(eventChan, logparse.VoteDetails); errRegister != nil {
		slog.Warn("logWriter Tried to register duplicate reader channel", log.ErrAttr(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventChan:
			serverEvent, ok := evt.Event.(logparse.VoteKickDetailsEvt)
			if !ok {
				continue
			}

			matchID, _ := r.matchUsecase.GetMatchIDFromServerID(evt.ServerID)

			result := domain.VoteResult{
				ServerID: evt.ServerID,
				MatchID:  matchID,
				SourceID: serverEvent.SID,
				TargetID: serverEvent.SID2,
				Valid:    serverEvent.Valid,
				Name:     serverEvent.Name,
			}

			if err := r.addVoteHistory(ctx, result); err != nil {
				slog.Error("Failed to add vote result", log.ErrAttr(err))
			}
		}
	}
}
