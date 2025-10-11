package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/slur"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type SlurMessage struct {
	steamID steamid.SteamID
	id      int64
	message string
}

func (s SlurMessage) UserID() string {
	return s.steamID.String()
}

func (s SlurMessage) MessageID() int64 {
	return s.id
}

func (s SlurMessage) Text() string {
	return s.message
}

type MessageProvider struct {
	Db     database.Database
	offset int
}

func (m *MessageProvider) Next(ctx context.Context, count uint64) ([]slur.Message, error) {
	query := fmt.Sprintf(`SELECT steam_id, person_message_id, body
					FROM person_messages
				   ORDER BY person_message_id OFFSET %d LIMIT %d`, m.offset, count)
	rows, errRows := m.Db.Query(ctx, query)
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return nil, nil
		}
		return nil, database.DBErr(errRows)
	}

	var res []slur.Message
	for rows.Next() {
		var sm SlurMessage
		if err := rows.Scan(&sm.steamID, &sm.id, &sm.message); err != nil {
			return nil, database.DBErr(err)
		}

		res = append(res, sm)
	}

	m.offset += int(count)

	return res, nil
}

type Repository struct {
	db database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{db: database}
}

func (r Repository) TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error) {
	rows, errRows := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select("p.personaname", "p.steam_id", "count(person_message_id) as total").
		From("person_messages m").
		LeftJoin("public.person p USING(steam_id)").
		GroupBy("p.steam_id").
		OrderBy("total DESC").
		Limit(count))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var results []TopChatterResult

	for rows.Next() {
		var (
			tcr     TopChatterResult
			steamID int64
		)

		if errScan := rows.Scan(&tcr.Name, &steamID, &tcr.Count); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		tcr.SteamID = steamid.New(steamID)
		results = append(results, tcr)
	}

	return results, nil
}

const minQueryLen = 2

func (r Repository) AddChatHistory(ctx context.Context, message *Message) error {
	const query = `INSERT INTO person_messages
    		(steam_id, server_id, body, team, created_on, persona_name, match_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING person_message_id`

	if errScan := r.db.
		QueryRow(ctx, query, message.SteamID.Int64(), message.ServerID, message.Body, message.Team,
			message.CreatedOn, message.PersonaName, message.MatchID).
		Scan(&message.PersonMessageID); errScan != nil {
		return database.DBErr(errScan)
	}

	return nil
}

func (r Repository) GetPersonMessageByID(ctx context.Context, personMessageID int64) (Message, error) {
	var msg Message

	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select(
			"m.person_message_id",
			"m.steam_id",
			"m.server_id",
			"m.body",
			"m.team",
			"m.created_on",
			"m.persona_name",
			"m.match_id",
			"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		Where(sq.Eq{"m.person_message_id": personMessageID}))

	if errRow != nil {
		return msg, database.DBErr(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&msg.PersonMessageID,
		&steamID,
		&msg.ServerID,
		&msg.Body,
		&msg.Team,
		&msg.CreatedOn,
		&msg.PersonaName,
		&msg.MatchID,
		&msg.ServerName); errScan != nil {
		return msg, database.DBErr(errScan)
	}

	msg.SteamID = steamid.New(steamID)

	return msg, nil
}

func (r Repository) QueryChatHistory(ctx context.Context, filters HistoryQueryFilter) ([]QueryChatHistoryResult, error) { //nolint:maintidx
	if filters.Query != "" && len(filters.Query) < minQueryLen {
		return nil, fmt.Errorf("%w: query", httphelper.ErrTooShort)
	}

	if filters.Personaname != "" && len(filters.Personaname) < minQueryLen {
		return nil, fmt.Errorf("%w: name", httphelper.ErrTooShort)
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

	rows, errQuery := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
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
			return nil, database.DBErr(errScan)
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

func (r Repository) GetPersonMessage(ctx context.Context, messageID int64) (QueryChatHistoryResult, error) {
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

	if err := database.DBErr(r.db.QueryRow(ctx, query, messageID).Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
		&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged)); err != nil {
		return msg, err
	}

	return msg, nil
}

func (r Repository) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]QueryChatHistoryResult, error) {
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

	rows, errRows := r.db.Query(ctx, query, messageID, paddedMessageCount, serverID)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}
	defer rows.Close()

	var messages []QueryChatHistoryResult

	for rows.Next() {
		var msg QueryChatHistoryResult

		if errScan := rows.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
			&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
