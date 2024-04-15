package votes

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"log/slog"
)

type voteRepository struct {
	db            database.Database
	personUsecase domain.PersonUsecase
	matchUsecase      domain.MatchUsecase
	broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
}

func NewVoteRepository(database database.Database, personUsecase domain.PersonUsecase, matchUsecase      domain.MatchUsecase, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]) domain.VoteRepository {
	return &voteRepository{
		db:            database,
		personUsecase: personUsecase,
		matchUsecase: matchUsecase,
		broadcaster:   broadcaster,
	}
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

		}
	}
}

func (r voteRepository) AddVoteHistory() error {

}
