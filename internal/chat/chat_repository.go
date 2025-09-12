package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ChatRepository struct {
	db          database.Database
	persons     *person.PersonUsecase
	wordFilters *WordFilterUsecase
	matches     match.MatchUsecase
	broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	WarningChan chan NewUserWarning
}

func NewChatRepository(database database.Database, personUsecase *person.PersonUsecase, wordFilterUsecase *WordFilterUsecase,
	matchUsecase match.MatchUsecase,
	broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
) *ChatRepository {
	return &ChatRepository{
		db:          database,
		persons:     personUsecase,
		wordFilters: wordFilterUsecase,
		matches:     matchUsecase,
		broadcaster: broadcaster,
		WarningChan: make(chan NewUserWarning),
	}
}

func (r ChatRepository) handleMessage(ctx context.Context, evt logparse.ServerEvent, person logparse.SourcePlayer, msg string, team bool, created time.Time, reason ban.Reason) {
	if msg == "" {
		slog.Warn("Empty message body, skipping")

		return
	}

	_, errPerson := r.persons.GetOrCreatePersonBySteamID(ctx, nil, person.SID)
	if errPerson != nil && !errors.Is(errPerson, database.ErrDuplicate) {
		slog.Error("Failed to handle message, could not get author", log.ErrAttr(errPerson), slog.String("message", msg))

		return
	}

	matchID, _ := r.matches.GetMatchIDFromServerID(evt.ServerID)

	personMsg := PersonMessage{
		SteamID:     person.SID,
		PersonaName: strings.ToValidUTF8(person.Name, "_"),
		ServerName:  evt.ServerName,
		ServerID:    evt.ServerID,
		Body:        strings.ToValidUTF8(msg, "_"),
		Team:        team,
		CreatedOn:   created,
		MatchID:     matchID,
	}

	if errChat := r.AddChatHistory(ctx, &personMsg); errChat != nil {
		slog.Error("Failed to add chat history", log.ErrAttr(errChat))

		return
	}

	go func(userMsg PersonMessage) {
		matchedFilter := r.wordFilters.Check(userMsg.Body)
		if len(matchedFilter) > 0 {
			if errSaveMatch := r.wordFilters.AddMessageFilterMatch(ctx, userMsg.PersonMessageID, matchedFilter[0].FilterID); errSaveMatch != nil {
				slog.Error("Failed to save message findMatch status", log.ErrAttr(errSaveMatch))
			}

			matchResult := matchedFilter[0]
			r.WarningChan <- NewUserWarning{
				UserMessage: userMsg,
				PlayerID:    person.PID,
				UserWarning: UserWarning{
					WarnReason: reason,
					Message:    userMsg.Body,
					// todo
					// Matched:       matchResult,
					MatchedFilter: matchResult,
					CreatedOn:     time.Now(),
					Personaname:   userMsg.PersonaName,
					Avatar:        userMsg.AvatarHash,
					ServerName:    userMsg.ServerName,
					ServerID:      userMsg.ServerID,
					SteamID:       userMsg.SteamID.String(),
				},
			}
		}
	}(personMsg)
}

func (r ChatRepository) Start(ctx context.Context) {
	eventChan := make(chan logparse.ServerEvent)
	if errRegister := r.broadcaster.Consume(eventChan, logparse.Connected, logparse.Say, logparse.SayTeam); errRegister != nil {
		slog.Warn("logWriter Tried to register duplicate reader channel", log.ErrAttr(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventChan:
			switch evt.EventType {
			case logparse.Connected:
				connectEvent, ok := evt.Event.(logparse.ConnectedEvt)
				if !ok {
					continue
				}

				connectMsg := "Player connected with username: " + connectEvent.Name

				r.handleMessage(ctx, evt, connectEvent.SourcePlayer, connectMsg, false, connectEvent.CreatedOn, ban.Username)
			case logparse.Say:
				fallthrough
			case logparse.SayTeam:
				sayEvent, ok := evt.Event.(logparse.SayEvt)
				if !ok {
					continue
				}

				r.handleMessage(ctx, evt, sayEvent.SourcePlayer, sayEvent.Msg, sayEvent.Team, sayEvent.CreatedOn, ban.Language)
			}
		}
	}
}

func (r ChatRepository) GetWarningChan() chan NewUserWarning {
	return r.WarningChan
}

func (r ChatRepository) TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error) {
	rows, errRows := r.db.QueryBuilder(ctx, nil, r.db.
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

	var results []TopChatterResult

	for rows.Next() {
		var (
			tcr     TopChatterResult
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

func (r ChatRepository) AddChatHistory(ctx context.Context, message *PersonMessage) error {
	const query = `INSERT INTO person_messages
    		(steam_id, server_id, body, team, created_on, persona_name, match_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING person_message_id`

	if errScan := r.db.
		QueryRow(ctx, nil, query, message.SteamID.Int64(), message.ServerID, message.Body, message.Team,
			message.CreatedOn, message.PersonaName, message.MatchID).
		Scan(&message.PersonMessageID); errScan != nil {
		return r.db.DBErr(errScan)
	}

	return nil
}

func (r ChatRepository) QueryChatHistory(ctx context.Context, filters ChatHistoryQueryFilter) ([]QueryChatHistoryResult, error) { //nolint:maintidx
	if filters.Query != "" && len(filters.Query) < minQueryLen {
		return nil, fmt.Errorf("%w: query", domain.ErrTooShort)
	}

	if filters.Personaname != "" && len(filters.Personaname) < minQueryLen {
		return nil, fmt.Errorf("%w: name", domain.ErrTooShort)
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
			"s.short_name",
			"mf.person_message_filter_id",
			"p.avatarhash",
			"CASE WHEN f.pattern IS NULL THEN '' ELSE f.pattern END").
		From("person_messages m").
		LeftJoin("server s USING(server_id)").
		LeftJoin("person_messages_filter mf USING(person_message_id)").
		LeftJoin("filtered_word f USING(filter_id)").
		LeftJoin("person p USING(steam_id)")

	builder = filters.ApplySafeOrder(builder, map[string][]string{
		"m.": {"persona_name", "person_message_id"},
	}, "person_message_id")
	builder = filters.ApplyLimitOffset(builder, 10000)

	var constraints sq.And

	now := time.Now()

	if !filters.Unrestricted {
		unrTime := now.AddDate(0, 0, -30)
		if filters.DateStart != nil && filters.DateStart.Before(unrTime) {
			return nil, datetime.ErrInvalidDuration
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

	if sid, ok := filters.SourceSteamID(); ok {
		constraints = append(constraints, sq.Eq{"m.steam_id": sid})
	}

	if filters.Personaname != "" {
		constraints = append(constraints, sq.Expr(`name_search @@ websearch_to_tsquery('simple', ?)`, filters.Personaname))
	}

	if filters.Query != "" {
		constraints = append(constraints, sq.Expr(`message_search @@ websearch_to_tsquery('simple', ?)`, filters.Query))
	}

	if filters.FlaggedOnly {
		constraints = append(constraints, sq.Gt{"mf.person_message_filter_id": 0})
	}

	var messages []QueryChatHistoryResult

	rows, errQuery := r.db.QueryBuilder(ctx, nil, builder.Where(constraints))
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			message QueryChatHistoryResult
			steamID int64
			matchID []byte
			flagged *int64
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
			&flagged,
			&message.AvatarHash,
			&message.Pattern); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		if matchID != nil {
			// Support for old messages which existed before matches
			message.MatchID = uuid.FromBytesOrNil(matchID)
		}

		if flagged != nil {
			message.AutoFilterFlagged = *flagged
		}

		message.SteamID = steamid.New(steamID)

		messages = append(messages, message)
	}

	if messages == nil {
		// Return empty list instead of null
		messages = []QueryChatHistoryResult{}
	}

	return messages, nil
}

func (r ChatRepository) GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error) {
	const query = `
		SELECT x.person_message_id, x.steam_id, x.server_id, x.body, x.team, x.created_on, x.persona_name, x.match_id, s.short_name, COALESCE(f.person_message_filter_id, 0) as flagged
		FROM (
		SELECT m.person_message_id,
			   m.steam_id,
			   m.server_id,
			   m.body,
			   m.team,
			   m.created_on,
			   m.persona_name,
			   m.match_id
		FROM person_messages m
		WHERE m.person_message_id = $1) x
		LEFT JOIN server s ON x.server_id = s.server_id
		LEFT JOIN person_messages_filter f on x.person_message_id = f.person_message_id`

	var msg QueryChatHistoryResult

	if err := r.db.DBErr(r.db.QueryRow(ctx, nil, query, messageID).Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
		&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged)); err != nil {
		return msg, err
	}

	return msg, nil
}

func (r ChatRepository) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error) {
	const query = `
		SELECT x.person_message_id, x.steam_id, x.server_id, x.body, x.team, x.created_on, x.persona_name, x.match_id, s.short_name, COALESCE(f.person_message_filter_id, 0) as flagged
		FROM ((SELECT m.person_message_id,
					  m.steam_id,
					  m.server_id,
					  m.body,
					  m.team,
					  m.created_on,
					  m.persona_name,
					  m.match_id
			   FROM person_messages m
						LEFT JOIN server s on m.server_id = s.server_id
			   WHERE m.server_id = $3
				 AND m.person_message_id >= $1
			   GROUP BY m.person_message_id
			   ORDER BY person_message_id ASC
			   LIMIT $2+1)
			  UNION
			  (SELECT m.person_message_id,
					  m.steam_id,
					  m.server_id,
					  m.body,
					  m.team,
					  m.created_on,
					  m.persona_name,
					  m.match_id
			   FROM person_messages m
						LEFT JOIN server s on m.server_id = s.server_id
			   WHERE m.server_id = $3
				 AND m.person_message_id < $1
			   GROUP BY m.person_message_id
			   ORDER BY person_message_id DESC
			   LIMIT $2)
			  ORDER BY person_message_id ASC) x
				 LEFT JOIN server s ON x.server_id = s.server_id
				 LEFT JOIN person_messages_filter f on x.person_message_id = f.person_message_id`

	if paddedMessageCount > 1000 {
		paddedMessageCount = 1000
	}

	if paddedMessageCount <= 0 {
		paddedMessageCount = 5
	}

	rows, errRows := r.db.Query(ctx, nil, query, messageID, paddedMessageCount, serverID)
	if errRows != nil {
		return nil, errors.Join(errRows, domain.ErrMessageContext)
	}
	defer rows.Close()

	var messages []QueryChatHistoryResult

	for rows.Next() {
		var msg QueryChatHistoryResult

		if errScan := rows.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
			&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged); errScan != nil {
			return nil, errors.Join(errRows, domain.ErrScanResult)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
