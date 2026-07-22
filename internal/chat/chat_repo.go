package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/httphelper"
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
	rows, errRows := m.Db.Query(ctx,
		`SELECT steam_id, person_message_id, body
		   FROM person_messages
		  ORDER BY person_message_id OFFSET $1 LIMIT $2`, m.offset, count)
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return nil, nil
		}

		return nil, database.Err(errRows)
	}

	var res []slur.Message
	for rows.Next() {
		var sm SlurMessage
		if err := rows.Scan(&sm.steamID, &sm.id, &sm.message); err != nil {
			return nil, database.Err(err)
		}

		res = append(res, &sm)
	}

	m.offset += int(count) //nolint:gosec

	return res, nil
}

type Repository struct {
	database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

func (r Repository) TopChatters(ctx context.Context, count uint64) ([]TopChatterResult, error) {
	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("p.personaname", "p.steam_id", "count(person_message_id) as total").
		From("person_messages m").
		LeftJoin("public.person p USING(steam_id)").
		GroupBy("p.steam_id").
		OrderBy("total DESC").
		Limit(count))
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	defer rows.Close()

	var results []TopChatterResult
	for rows.Next() {
		var (
			tcr     TopChatterResult
			steamID int64
		)

		if errScan := rows.Scan(&tcr.Name, &steamID, &tcr.Count); errScan != nil {
			return nil, database.Err(errScan)
		}

		tcr.SteamID = steamid.New(steamID)
		results = append(results, tcr)
	}

	return results, nil
}

const minQueryLen = 2

func (r Repository) DeleteByDemoID(ctx context.Context, demoID int32) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("person_messages").Where(sq.Eq{"demo_id": demoID})))
}

func (r Repository) AddChatHistory(ctx context.Context, message *Message) error {
	const query = `INSERT INTO person_messages
    		(steam_id, server_id, body, created_on, persona_name, demo_id, demo_tick, match_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING person_message_id`

	if errScan := r.
		QueryRow(ctx, query, message.SteamID.Int64(), message.ServerID, message.Body,
			message.CreatedOn, message.PersonaName, message.DemoID, message.DemoTick, message.MatchID).
		Scan(&message.PersonMessageID); errScan != nil {
		return database.Err(errScan)
	}

	return nil
}

func (r Repository) GetPersonMessageByID(ctx context.Context, personMessageID int64) (Message, error) {
	var msg Message

	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select(
			"m.person_message_id",
			"m.steam_id",
			"m.server_id",
			"m.body",
			"m.created_on",
			"m.persona_name",
			"m.demo_id",
			"m.demo_tick",
			"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		Where(sq.Eq{"m.person_message_id": personMessageID}))

	if errRow != nil {
		return msg, database.Err(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&msg.PersonMessageID,
		&steamID,
		&msg.ServerID,
		&msg.Body,
		&msg.CreatedOn,
		&msg.PersonaName,
		&msg.DemoID,
		&msg.DemoTick,
		&msg.ServerName); errScan != nil {
		return msg, database.Err(errScan)
	}

	msg.SteamID = steamid.New(steamID)

	return msg, nil
}

func (r Repository) loadDemoInfo(ctx context.Context, results []*QueryChatHistoryResult) error {
	var ids []uuid.UUID
	for _, res := range results {
		if res.MatchID == nil {
			continue
		}
		if !slices.Contains(ids, *res.MatchID) {
			ids = append(ids, *res.MatchID)
		}
	}

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("d.demo_id", "COALESCE(a.asset_id, '00000000-0000-0000-0000-000000000000') as asset_id").
		From("asset a").
		LeftJoin("demo d ON d.asset_id = a.asset_id").
		LeftJoin("match m ON m.demo_id = d.demo_id").
		Where(sq.Eq{"m.match_id": ids}))
	if errRows != nil {
		return errRows
	}

	for rows.Next() {
		var assetID uuid.UUID
		var demoID *int32
		if err := rows.Scan(&demoID, &assetID); err != nil {
			return database.Err(err)
		}
		for _, res := range results {
			if *res.DemoID == *demoID {
				res.AssetID = assetID
			}
		}
	}

	if err := rows.Err(); err != nil {
		return database.Err(err)
	}

	return nil
}

func (r Repository) QueryChatHistory(ctx context.Context, filters HistoryQueryFilter) ([]*QueryChatHistoryResult, error) { //nolint:maintidx
	if filters.Query != "" && len(filters.Query) < minQueryLen {
		return nil, fmt.Errorf("%w: query", httphelper.ErrTooShort)
	}

	if filters.Personaname != "" && len(filters.Personaname) < minQueryLen {
		return nil, fmt.Errorf("%w: name", httphelper.ErrTooShort)
	}

	builder := r.Builder().
		Select(
			"m.person_message_id",
			"m.steam_id",
			"m.server_id",
			"m.body",
			"m.created_on",
			"p.personaname",
			"m.demo_id",
			"m.demo_tick",
			"m.match_id",
			"p.avatarhash").
		From("person_messages m").
		LeftJoin("person p USING(steam_id)")

	// builder = filters.ApplySafeOrder(builder, map[string][]string{
	// 	"m.": {"persona_name", "person_message_id"},
	// }, "person_message_id")
	builder = builder.OrderBy("m.created_on DESC")
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

	if len(filters.ServerIDs) > 0 {
		constraints = append(constraints, sq.Eq{"m.server_id": filters.ServerIDs})
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

	var messages []*QueryChatHistoryResult
	rows, errQuery := r.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, database.Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			message = &QueryChatHistoryResult{}
			steamID int64
		)

		if errScan := rows.Scan(&message.PersonMessageID,
			&steamID,
			&message.ServerID,
			&message.Body,
			&message.CreatedOn,
			&message.PersonaName,
			&message.DemoID,
			&message.DemoTick,
			&message.MatchID,
			&message.AvatarHash); errScan != nil {
			return nil, database.Err(errScan)
		}

		message.SteamID = steamid.New(steamID)

		messages = append(messages, message)
	}

	if messages == nil {
		// Return empty list instead of null
		messages = []*QueryChatHistoryResult{}
	}

	if err := r.loadDemoInfo(ctx, messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r Repository) GetPersonMessage(ctx context.Context, messageID int64) (*QueryChatHistoryResult, error) {
	builder := r.Builder().
		Select(
			"m.person_message_id",
			"m.steam_id",
			"m.server_id",
			"m.body",
			"m.created_on",
			"p.personaname",
			"m.demo_id",
			"m.demo_tick",
			"m.match_id",
			"p.avatarhash").
		From("person_messages m").
		LeftJoin("person p USING(steam_id)").
		Where(sq.Eq{"m.person_message_id": messageID})

	msg := &QueryChatHistoryResult{}
	row, errRow := r.QueryRowBuilder(ctx, builder)
	if errRow != nil {
		return nil, errRow
	}
	if errRow := row.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.CreatedOn,
		&msg.PersonaName, &msg.DemoID, &msg.DemoTick, &msg.MatchID, &msg.AvatarHash); errRow != nil {
		return msg, database.Err(errRow)
	}

	return msg, nil
}

func (r Repository) GetPersonMessageContext(ctx context.Context, serverID int32, messageID int64, paddedMessageCount int32) ([]QueryChatHistoryResult, error) {
	if paddedMessageCount > 1000 {
		paddedMessageCount = 1000
	} else if paddedMessageCount <= 0 {
		paddedMessageCount = 5
	}

	query := fmt.Sprintf(`
		SELECT person_message_id, steam_id, server_id, body, created_on,
		       persona_name, demo_id, demo_tick, match_id, avatarhash
		FROM (
		    (SELECT m.person_message_id, m.steam_id, m.server_id, m.body, m.created_on,
		            p.personaname AS persona_name, m.demo_id, m.demo_tick,
		            m.match_id, p.avatarhash
		     FROM person_messages m
		     JOIN person p ON m.steam_id = p.steam_id
		     WHERE m.server_id = $1 AND m.person_message_id >= $2
		     ORDER BY m.person_message_id ASC
		     LIMIT %d + 1)

		    UNION ALL

		    (SELECT m.person_message_id, m.steam_id, m.server_id, m.body, m.created_on,
		            p.personaname AS persona_name, m.demo_id, m.demo_tick,
		            m.match_id, p.avatarhash
		     FROM person_messages m
		     JOIN person p ON m.steam_id = p.steam_id
		     WHERE m.server_id = $1 AND m.person_message_id < $2
		     ORDER BY m.person_message_id DESC
		     LIMIT %d)
		) sub
		ORDER BY person_message_id ASC`, paddedMessageCount, paddedMessageCount)

	rows, errRows := r.Query(ctx, query, serverID, messageID)
	if errRows != nil {
		return nil, database.Err(errRows)
	}
	defer rows.Close()

	var messages []QueryChatHistoryResult

	for rows.Next() {
		var msg QueryChatHistoryResult

		if errScan := rows.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.CreatedOn,
			&msg.PersonaName, &msg.DemoID, &msg.DemoTick, &msg.MatchID, &msg.AvatarHash); errScan != nil {
			return nil, database.Err(errScan)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
