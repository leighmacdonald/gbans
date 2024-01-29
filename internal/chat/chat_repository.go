package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type chatRepository struct {
	db           database.Database
	log          *zap.Logger
	pu           domain.PersonUsecase
	wfu          domain.WordFilterUsecase
	broadcaster  *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	matchUUIDMap fp.MutexMap[int, uuid.UUID]
	WarningChan  chan domain.NewUserWarning
}

func NewChatRepository(database database.Database, log *zap.Logger, pu domain.PersonUsecase, wfu domain.WordFilterUsecase, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	matchUUIDMap fp.MutexMap[int, uuid.UUID],
) domain.ChatRepository {
	return &chatRepository{
		db:           database,
		log:          log,
		pu:           pu,
		wfu:          wfu,
		broadcaster:  broadcaster,
		matchUUIDMap: matchUUIDMap,
		WarningChan:  make(chan domain.NewUserWarning),
	}
}

func (r chatRepository) Start(ctx context.Context) {
	eventChan := make(chan logparse.ServerEvent)
	if errRegister := r.broadcaster.Consume(eventChan, logparse.Say, logparse.SayTeam); errRegister != nil {
		r.log.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventChan:
			switch evt.EventType {
			case logparse.Say:
				fallthrough
			case logparse.SayTeam:
				newServerEvent, ok := evt.Event.(logparse.SayEvt)
				if !ok {
					continue
				}

				if newServerEvent.Msg == "" {
					r.log.Warn("Empty Person message body, skipping")

					continue
				}

				var author domain.Person
				if errPerson := r.pu.GetOrCreatePersonBySteamID(ctx, newServerEvent.SID, &author); errPerson != nil {
					r.log.Error("Failed to add chat history, could not get author", zap.Error(errPerson))

					continue
				}

				matchID, _ := r.matchUUIDMap.Get(evt.ServerID)

				msg := domain.PersonMessage{
					SteamID:     newServerEvent.SID,
					PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
					ServerName:  evt.ServerName,
					ServerID:    evt.ServerID,
					Body:        strings.ToValidUTF8(newServerEvent.Msg, "_"),
					Team:        newServerEvent.Team,
					CreatedOn:   newServerEvent.CreatedOn,
					MatchID:     matchID,
				}

				if errChat := r.AddChatHistory(ctx, &msg); errChat != nil {
					r.log.Error("Failed to add chat history", zap.Error(errChat))

					continue
				}

				go func(userMsg domain.PersonMessage) {
					if msg.ServerName == "localhost-1" {
						r.log.Debug("Chat message",
							zap.Int64("id", msg.PersonMessageID),
							zap.String("server", evt.ServerName),
							zap.String("name", newServerEvent.Name),
							zap.String("steam_id", newServerEvent.SID.String()),
							zap.Bool("team", msg.Team),
							zap.String("message", msg.Body))
					}

					matchedFilter := r.wfu.Check(userMsg.Body)
					if len(matchedFilter) > 0 {
						if errSaveMatch := r.wfu.AddMessageFilterMatch(ctx, userMsg.PersonMessageID, matchedFilter[0].FilterID); errSaveMatch != nil {
							r.log.Error("Failed to save message findMatch status", zap.Error(errSaveMatch))
						}

						matchResult := matchedFilter[0]

						r.WarningChan <- domain.NewUserWarning{
							UserMessage: userMsg,
							UserWarning: domain.UserWarning{
								WarnReason: domain.Language,
								Message:    userMsg.Body,
								// todo
								// Matched:       matchResult,
								MatchedFilter: &matchResult,
								CreatedOn:     time.Now(),
								Personaname:   userMsg.PersonaName,
								Avatar:        userMsg.AvatarHash,
								ServerName:    userMsg.ServerName,
								ServerID:      userMsg.ServerID,
								SteamID:       userMsg.SteamID.String(),
							},
						}
					}
				}(msg)
			}
		}
	}
}

func (r chatRepository) TopChatters(ctx context.Context, count uint64) ([]domain.TopChatterResult, error) {
	rows, errRows := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select("p.personaname", "p.steam_id", "count(person_message_id) as total").
		From("person_messages m").
		LeftJoin("public.person p USING(steam_id)").
		GroupBy("p.steam_id").
		OrderBy("total DESC").
		Limit(count))
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	var results []domain.TopChatterResult

	for rows.Next() {
		var (
			tcr     domain.TopChatterResult
			steamID int64
		)

		if errScan := rows.Scan(&tcr.Name, &steamID, &tcr.Count); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		tcr.SteamID = steamid.New(steamID)
		results = append(results, tcr)
	}

	return results, nil
}

const minQueryLen = 2

func (r chatRepository) AddChatHistory(ctx context.Context, message *domain.PersonMessage) error {
	const query = `INSERT INTO person_messages 
    		(steam_id, server_id, body, team, created_on, persona_name, match_id) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING person_message_id`

	if errScan := r.db.
		QueryRow(ctx, query, message.SteamID.Int64(), message.ServerID, message.Body, message.Team,
			message.CreatedOn, message.PersonaName, message.MatchID).
		Scan(&message.PersonMessageID); errScan != nil {
		return r.db.DBErr(errScan)
	}

	return nil
}

func (r chatRepository) QueryChatHistory(ctx context.Context, filters domain.ChatHistoryQueryFilter) ([]domain.QueryChatHistoryResult, int64, error) { //nolint:maintidx
	if filters.Query != "" && len(filters.Query) < minQueryLen {
		return nil, 0, fmt.Errorf("%w: query", domain.ErrTooShort)
	}

	if filters.Personaname != "" && len(filters.Personaname) < minQueryLen {
		return nil, 0, fmt.Errorf("%w: name", domain.ErrTooShort)
	}

	builder := r.db.
		Builder().
		Select("m.person_message_id",
			"m.steam_id ",
			"m.server_id",
			"m.body",
			"m.team ",
			"m.created_on",
			"m.persona_name",
			"m.match_id",
			"r.short_name",
			"CASE WHEN mf.person_message_id::int::boolean THEN mf.person_message_filter_id ELSE 0 END as flagged",
			"r.avatarhash",
			"CASE WHEN f.pattern IS NULL THEN '' ELSE f.pattern END").
		From("person_messages m").
		LeftJoin("server r USING(server_id)").
		LeftJoin("person_messages_filter mf USING(person_message_id)").
		LeftJoin("filtered_word f USING(filter_id)").
		LeftJoin("person r USING(steam_id)")

	builder = filters.ApplySafeOrder(builder, map[string][]string{
		"m.": {"persona_name", "person_message_id"},
	}, "person_message_id")
	builder = filters.ApplyLimitOffsetDefault(builder)

	var constraints sq.And

	now := time.Now()

	if !filters.Unrestricted {
		unrTime := now.AddDate(0, 0, -14)
		if filters.DateStart != nil && filters.DateStart.Before(unrTime) {
			return nil, 0, util.ErrInvalidDuration
		}
	}

	switch {
	case filters.DateStart != nil && filters.DateEnd != nil:
		constraints = append(constraints, sq.Expr("m.created_on BETWEEN ? AND ?", filters.DateStart, filters.DateEnd))
	case filters.DateStart != nil:
		constraints = append(constraints, sq.Expr("? > m.created_on", filters.DateStart))
	case filters.DateEnd != nil:
		constraints = append(constraints, sq.Expr("? < m.created_on", filters.DateEnd))
	}

	if filters.ServerID > 0 {
		constraints = append(constraints, sq.Eq{"m.server_id": filters.ServerID})
	}

	if filters.SourceID != "" {
		sid, errSID := filters.SourceID.SID64(ctx)
		if errSID != nil {
			return nil, 0, errors.Join(errSID, domain.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"m.steam_id": sid.Int64()})
	}

	if filters.Personaname != "" {
		constraints = append(constraints, sq.Expr(`name_search @@ websearch_to_tsquery('simple', ?)`, filters.Personaname))
	}

	if filters.Query != "" {
		constraints = append(constraints, sq.Expr(`message_search @@ websearch_to_tsquery('simple', ?)`, filters.Query))
	}

	if filters.FlaggedOnly {
		constraints = append(constraints, sq.Eq{"flagged": true})
	}

	var messages []domain.QueryChatHistoryResult

	rows, errQuery := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, 0, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			message domain.QueryChatHistoryResult
			steamID int64
			matchID []byte
		)

		if errScan := rows.Scan(&message.PersonMessageID,
			&steamID,
			&message.ServerID,
			&message.Body,
			&message.Team,
			&message.CreatedOn,
			&message.PersonaName,
			&matchID,
			&message.ServerName,
			&message.AutoFilterFlagged,
			&message.AvatarHash,
			&message.Pattern); errScan != nil {
			return nil, 0, r.db.DBErr(errScan)
		}

		if matchID != nil {
			// Support for old messages which existed before matches
			message.MatchID = uuid.FromBytesOrNil(matchID)
		}

		message.SteamID = steamid.New(steamID)

		messages = append(messages, message)
	}

	if messages == nil {
		// Return empty list instead of null
		messages = []domain.QueryChatHistoryResult{}
	}

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("count(m.created_on) as count").
		From("person_messages m").
		LeftJoin("server r on m.server_id = r.server_id").
		LeftJoin("person_messages_filter f on m.person_message_id = f.person_message_id").
		LeftJoin("person r on r.steam_id = m.steam_id").
		Where(constraints))

	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	return messages, count, nil
}

func (r chatRepository) GetPersonMessage(ctx context.Context, messageID int64, msg *domain.QueryChatHistoryResult) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("m.person_message_id", "m.steam_id", "m.server_id", "m.body", "m.team", "m.created_on",
			"m.persona_name", "m.match_id", "r.short_name", "COUNT(f.person_message_id)::int::boolean as flagged").
		From("person_messages m").
		LeftJoin("server r USING(server_id)").
		LeftJoin("person_messages_filter f USING(person_message_id)").
		Where(sq.Eq{"m.person_message_id": messageID}).
		GroupBy("m.person_message_id", "r.short_name"))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	return r.db.DBErr(row.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
		&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged))
}

func (r chatRepository) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]domain.QueryChatHistoryResult, error) {
	const query = `
		(
			SELECT m.person_message_id, m.steam_id,	m.server_id, m.body, m.team, m.created_on, 
			       m.persona_name,  m.match_id, r.short_name, COUNT(f.person_message_id)::int::boolean as flagged
			FROM person_messages m 
			LEFT JOIN server r on m.server_id = r.server_id
			LEFT JOIN person_messages_filter f on m.person_message_id = f.person_message_id
		 	WHERE m.server_id = $3 AND m.person_message_id >= $1 
		 	GROUP BY m.person_message_id, r.short_name 
		 	ORDER BY m.person_message_id ASC
		 	
		 	LIMIT $2+1
		)
		UNION
		(
			SELECT m.person_message_id, m.steam_id, m.server_id, m.body, m.team, m.created_on, 
			       m.persona_name,  m.match_id, r.short_name, COUNT(f.person_message_id)::int::boolean as flagged
		 	FROM person_messages m 
		 	    LEFT JOIN server r on m.server_id = r.server_id 
		 	LEFT JOIN person_messages_filter f on m.person_message_id = f.person_message_id
		 	WHERE m.server_id = $3 AND  m.person_message_id < $1
		 	GROUP BY m.person_message_id, r.short_name
		 	ORDER BY m.person_message_id DESC
		 	LIMIT $2
		)
		ORDER BY person_message_id DESC`

	if paddedMessageCount > 1000 {
		paddedMessageCount = 1000
	}

	if paddedMessageCount <= 0 {
		paddedMessageCount = 5
	}

	rows, errRows := r.db.Query(ctx, query, messageID, paddedMessageCount, serverID)
	if errRows != nil {
		return nil, errors.Join(errRows, domain.ErrMessageContext)
	}
	defer rows.Close()

	var messages []domain.QueryChatHistoryResult

	for rows.Next() {
		var msg domain.QueryChatHistoryResult

		if errScan := rows.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
			&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged); errScan != nil {
			return nil, errors.Join(errRows, domain.ErrScanResult)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
