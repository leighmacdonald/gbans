package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	MinPlayers = 4
	MinDuraion = 300
)

var ErrInvalidState = errors.New("invalid demo state")

type Stats struct {
	repo Repository
	maps maps.Maps
}

func New(repo Repository, maps maps.Maps) Stats {
	return Stats{repo: repo, maps: maps}
}

func (s Stats) Import(ctx context.Context, serverID int32, demo demoparse.Demo) (*Result, error) {
	timeStart := time.Now().Add(-time.Duration(demo.Duration) * time.Second)
	if demo.DemoType != demoparse.HL2Demo {
		return nil, fmt.Errorf("%w: invalid demo type", ErrInvalidState)
	}

	if demo.Server == "" {
		return nil, fmt.Errorf("%w: invalid server name", ErrInvalidState)
	}

	if demo.Filename == "" {
		return nil, fmt.Errorf("%w: invalid file name", ErrInvalidState)
	}

	if len(demo.SteamIDs()) < MinPlayers {
		return nil, fmt.Errorf("%w: not enough players", ErrInvalidState)
	}

	if demo.Duration < MinDuraion {
		return nil, fmt.Errorf("%w: demo too short in length", ErrInvalidState)
	}

	if len(demo.SteamIDs()) < MinPlayers {
		return nil, fmt.Errorf("%w: not enough players", ErrInvalidState)
	}

	if demo.Duration < MinDuraion {
		return nil, fmt.Errorf("%w: demo too short in length", ErrInvalidState)
	}

	if demo.Map == "" {
		return nil, fmt.Errorf("%w: empty map invalid", ErrInvalidState)
	}

	newID, errID := uuid.NewV4()
	if errID != nil {
		return nil, fmt.Errorf("%w: failed to generate UUID", ErrInvalidState)
	}

	mapInfo, errMap := s.maps.Get(ctx, demo.Map)
	if errMap != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidState, errMap)
	}

	players := map[steamid.SteamID]*Player{}
	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			user := steamid.New(player.SteamID)
			if !user.Valid() {
				continue
			}
			plr, ok := players[user]
			if !ok {
				plr = &Player{MedicStats: &PlayerMedicStats{}}
				players[user] = plr
			}
			plr.ApplySummary(&player)
		}
	}

	var chat []PersonMessage //nolint:prealloc
	for _, message := range demo.Chat {
		user := steamid.New(message.User)
		if message.Message == "" || message.Tick <= 0 {
			continue
		}

		chat = append(chat, PersonMessage{
			MatchID: newID,
			SteamID: user,
			Body:    message.Message,
			Tick:    message.Tick,
		})
	}

	result := Result{
		MatchID:    newID,
		ServerID:   serverID,
		Title:      demo.Server,
		TimeStart:  timeStart,
		TimeEnd:    time.Now(),
		Map:        mapInfo,
		Winner:     demo.Winner(),
		TeamScores: demo.Scores(),
		Chat:       chat,
	}

	for _, player := range players {
		result.Players = append(result.Players, player)
	}

	return &result, nil
}
